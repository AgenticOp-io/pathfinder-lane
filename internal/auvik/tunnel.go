package auvik

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/scottpeterman/pathfinderssh/internal/sessions"
)

// TunnelManager keeps AuvikTunnel processes for reuse across sessions.
type TunnelManager struct {
	BinPath string
	mu      sync.Mutex
	active  map[string]*tunnelProc
}

type tunnelProc struct {
	key       string
	localPort int
	cmd       *exec.Cmd
}

// NewTunnelManager builds a manager. BinPath may be empty; ResolveTunnelBinary
// is used per launch.
func NewTunnelManager(binPath string) *TunnelManager {
	return &TunnelManager{
		BinPath: strings.TrimSpace(binPath),
		active:  map[string]*tunnelProc{},
	}
}

// ResolveTunnelBinary returns an explicit path, AUVIK_TUNNEL_BIN, or a common install location.
func ResolveTunnelBinary(explicit string) string {
	if p := strings.TrimSpace(explicit); p != "" {
		return p
	}
	if p := strings.TrimSpace(os.Getenv("AUVIK_TUNNEL_BIN")); p != "" {
		return p
	}
	name := "AuvikTunnel"
	if runtime.GOOS == "windows" {
		name = "AuvikTunnel.exe"
	}
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	if runtime.GOOS == "windows" {
		local := os.Getenv("LOCALAPPDATA")
		candidates := []string{
			filepath.Join(local, "Programs", "Auvik", name),
			filepath.Join(local, "Auvik", name),
		}
		for _, c := range candidates {
			if fileExists(c) {
				return c
			}
		}
	}
	return ""
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Ensure starts or reuses a tunnel to deviceIP:remotePort via the tenant domain.
func (m *TunnelManager) Ensure(ctx context.Context, domain, deviceIP string, remotePort, wantLocal int) (int, error) {
	domain = strings.TrimSpace(domain)
	deviceIP = strings.TrimSpace(deviceIP)
	if domain == "" || deviceIP == "" {
		return 0, fmt.Errorf("Auvik tunnel needs domain and device IP")
	}
	if remotePort <= 0 {
		remotePort = 22
	}
	key := domain + ":" + deviceIP + ":" + strconv.Itoa(remotePort)

	m.mu.Lock()
	if p := m.active[key]; p != nil && portListening(p.localPort) {
		port := p.localPort
		m.mu.Unlock()
		return port, nil
	}
	m.mu.Unlock()

	bin := ResolveTunnelBinary(m.BinPath)
	if bin == "" {
		return 0, fmt.Errorf("AuvikTunnel not found — install from Auvik or set AUVIK_TUNNEL_BIN / Settings → Auvik tunnel path")
	}

	localPort := wantLocal
		if localPort <= 0 {
			p, err := pickListenPort()
			if err != nil {
				return 0, err
			}
			wantLocal = p
		}

	spec := fmt.Sprintf("tcp:%d:%s:%d:%s", wantLocal, deviceIP, remotePort, domain)
	cmd := exec.CommandContext(ctx, bin, spec)
	cmd.Stdout = nil
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("start AuvikTunnel: %w", err)
	}

	deadline := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			_ = cmd.Process.Kill()
			return 0, ctx.Err()
		}
		if portListening(wantLocal) {
			m.mu.Lock()
			m.active[key] = &tunnelProc{key: key, localPort: wantLocal, cmd: cmd}
			m.mu.Unlock()
			return wantLocal, nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	_ = cmd.Process.Kill()
	msg := strings.TrimSpace(stderr.String())
	if msg != "" {
		return 0, fmt.Errorf("AuvikTunnel did not open local port %d within 45s: %s", wantLocal, msg)
	}
	return 0, fmt.Errorf("AuvikTunnel did not open local port %d within 45s", wantLocal)
}

// StopAll terminates tracked tunnels.
func (m *TunnelManager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, p := range m.active {
		if p.cmd != nil && p.cmd.Process != nil {
			_ = p.cmd.Process.Kill()
		}
		delete(m.active, k)
	}
}

func pickListenPort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

func portListening(port int) bool {
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	conn, err := net.DialTimeout("tcp", addr, 300*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// ShouldTryTunnel reports whether a failed dial should attempt AuvikTunnel.
func ShouldTryTunnel(n sessions.Node, err error, autoTunnel bool) bool {
	if err == nil {
		return false
	}
	if !autoTunnel && !n.AuvikUseTunnel {
		return false
	}
	domain := strings.TrimSpace(n.AuvikDomain)
	host := strings.TrimSpace(n.Host)
	if domain == "" || host == "" {
		return false
	}
	return isReachFailure(err)
}

func isReachFailure(err error) bool {
	msg := strings.ToLower(err.Error())
	for _, sub := range []string{
		"not reachable",
		"connection refused",
		"timed out",
		"timeout",
		"no route",
		"network unreachable",
		"unreachable",
		"connection reset",
		"actively refused",
	} {
		if strings.Contains(msg, sub) {
			return true
		}
	}
	return false
}

// DialNodeViaTunnel returns a copy of n dialed through localhost:localPort.
func DialNodeViaTunnel(n sessions.Node, localPort int) sessions.Node {
	out := n.Normalize()
	out.Host = "127.0.0.1"
	out.Port = localPort
	return out
}
