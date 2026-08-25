package auvik

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/scottpeterman/pathfinderssh/internal/crashlog"
	"github.com/scottpeterman/pathfinderssh/internal/sessions"
)

const (
	tunnelLogName = "auvik-tunnel.log"
	// Keep unused tunnels warm so close→reopen does not pay a full Auvik relaunch.
	tunnelIdleGrace = 2 * time.Minute
	// After the local port is bound, give Auvik a moment before the first real
	// SSH dial. Never Dial/Close the tunnel port to "probe" it — Auvik accepts
	// that TCP, starts a work websocket, then aborts the session when the probe
	// closes (TunnelClient: terminating because of termination of TcpAccepted…).
	tunnelListenSettle = 750 * time.Millisecond
)

// TunnelManager keeps AuvikTunnel processes for reuse across sessions.
type TunnelManager struct {
	BinPath  string
	AppHome  string
	Username string
	APIKey   string
	HostBase string // e.g. us2.my.auvik.com
	mu       sync.Mutex
	active   map[string]*tunnelProc
	keyLocks sync.Map // key → *sync.Mutex, serializes Ensure per mapping
}

type tunnelProc struct {
	key       string
	localPort int
	pid       int
	cmd       *exec.Cmd
	workDir   string // per-tunnel slot dir (isolates Auvik release supervisor)
	exited    *atomic.Bool
	refs      int // terminal tabs holding this mapping
	idleStop  *time.Timer
}

func NewTunnelManager(binPath string) *TunnelManager {
	return &TunnelManager{
		BinPath: strings.TrimSpace(binPath),
		active:  map[string]*tunnelProc{},
	}
}

func (m *TunnelManager) SetAppHome(appHome string) {
	m.AppHome = strings.TrimSpace(appHome)
}

func (m *TunnelManager) SetCredentials(username, apiKey, apiBaseURL string) {
	m.Username = strings.TrimSpace(username)
	m.APIKey = strings.TrimSpace(apiKey)
	m.HostBase = tunnelHostBase(apiBaseURL)
}

func tunnelHostBase(apiBase string) string {
	u := strings.TrimSpace(apiBase)
	if u == "" {
		return ""
	}
	u = strings.TrimPrefix(u, "https://")
	u = strings.TrimPrefix(u, "http://")
	if i := strings.IndexAny(u, "/#?"); i >= 0 {
		u = u[:i]
	}
	u = strings.TrimPrefix(strings.ToLower(u), "auvikapi.")
	return strings.TrimSpace(u)
}

// ResolveTunnelBinary prefers the Auvik-installed client (from AuvikTunnel -i).
func ResolveTunnelBinary(explicit string) string {
	if p := strings.TrimSpace(explicit); p != "" && fileExists(p) {
		return p
	}
	if p := strings.TrimSpace(os.Getenv("AUVIK_TUNNEL_BIN")); p != "" && fileExists(p) {
		return p
	}
	name := "AuvikTunnel"
	if runtime.GOOS == "windows" {
		name = "AuvikTunnel.exe"
	}
	if runtime.GOOS == "windows" {
		home, _ := os.UserHomeDir()
		local := os.Getenv("LOCALAPPDATA")
		candidates := []string{
			// Packaged with PathfinderSSH MSP — prefer this over a separate Auvik UI install.
			filepath.Join(local, "PathfinderSSH-MSP", "bin", name),
			filepath.Join(home, "auvik", "Auvik Tunnel", name),
			filepath.Join(local, "PathfinderSSH", "bin", name),
			filepath.Join(local, "pathfinderssh", "bin", name),
			filepath.Join(local, "Programs", "Auvik", name),
			filepath.Join(local, "Auvik", name),
		}
		for _, c := range candidates {
			if fileExists(c) {
				return c
			}
		}
	}
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	return ""
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (m *TunnelManager) lockKey(key string) func() {
	v, _ := m.keyLocks.LoadOrStore(key, &sync.Mutex{})
	mu := v.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

// Ensure starts or reuses a tunnel. ctx bounds wait-for-listen only.
func (m *TunnelManager) Ensure(ctx context.Context, domain, deviceIP string, remotePort, wantLocal int) (port int, err error) {
	defer func() {
		if r := recover(); r != nil {
			crashlog.Write(m.AppHome, "auvik-ensure", r)
			err = fmt.Errorf("Auvik tunnel panicked: %v", r)
			port = 0
			m.log("error", domain, "tunnel panic", fmt.Sprint(r))
		}
	}()
	started := time.Now()
	domain = strings.TrimSpace(domain)
	deviceIP = strings.TrimSpace(deviceIP)
	m.log("info", domain, "ensure begin", fmt.Sprintf("ip=%s port=%d", deviceIP, remotePort))
	if domain == "" || deviceIP == "" {
		m.log("error", "", "tunnel needs domain and device IP", fmt.Sprintf("domain=%q ip=%q", domain, deviceIP))
		return 0, fmt.Errorf("Auvik tunnel needs domain and device IP")
	}
	if remotePort <= 0 {
		remotePort = 22
	}
	key := domain + ":" + deviceIP + ":" + strconv.Itoa(remotePort)
	unlock := m.lockKey(key)
	defer unlock()

	m.mu.Lock()
	var stale *tunnelProc
	if p := m.active[key]; p != nil {
		if p.alive() && portListening(p.localPort) {
			m.cancelIdleLocked(p)
			p.refs++
			port := p.localPort
			refs := p.refs
			m.mu.Unlock()
			m.log("info", domain, fmt.Sprintf("reuse localhost:%d → %s:%d (refs=%d, %s)", port, deviceIP, remotePort, refs, time.Since(started).Round(time.Millisecond)), "")
			return port, nil
		}
		// Never kill while holding m.mu — taskkill/WMI can hang and would
		// freeze StopAll on the UI quit path.
		m.cancelIdleLocked(p)
		stale = p
		delete(m.active, key)
	}
	m.mu.Unlock()
	if stale != nil {
		// Kill off the dial path so a wedged taskkill cannot stall Connecting.
		go killTunnelProc(stale)
	}

	bin := ResolveTunnelBinary(m.BinPath)
	if bin == "" {
		msg := "AuvikTunnel not found — re-run the PathfinderSSH MSP installer, or set Settings → Integrations → AuvikTunnel path"
		m.log("error", domain, "binary not found", msg)
		return 0, fmt.Errorf("%s", msg)
	}

	// Stable local port per device mapping so a Pathfinder restart can reuse an
	// already-running AuvikTunnel instead of launching a second supervisor in
	// the same slot dir (which never binds and times out).
	if wantLocal <= 0 {
		wantLocal = stableLocalPort(key)
	}

	// AuvikTunnel's release supervisor is one-per-working-directory. Running two
	// tunnels from the same install dir fails; each mapping gets its own slot
	// copy so multiple local ports can stay up at once.
	slotBin, err := m.prepareTunnelSlot(bin, key)
	if err != nil {
		m.log("error", domain, "tunnel slot failed", err.Error())
		return 0, err
	}
	workDir := filepath.Dir(slotBin)

	// Orphans from a prior Pathfinder process may still accept TCP. Adopt a
	// listener backed by a live AuvikTunnel in this slot instead of relaunching.
	if portListening(wantLocal) {
		if pid := slotTunnelPID(workDir); pid > 0 {
			time.Sleep(tunnelListenSettle)
			m.mu.Lock()
			m.active[key] = &tunnelProc{key: key, localPort: wantLocal, pid: pid, workDir: workDir, refs: 1}
			m.mu.Unlock()
			m.log("info", domain, fmt.Sprintf("adopted orphan localhost:%d → %s:%d (%s)", wantLocal, deviceIP, remotePort, time.Since(started).Round(time.Millisecond)), "")
			return wantLocal, nil
		}
		m.log("info", domain, fmt.Sprintf("clearing prior listener on localhost:%d", wantLocal), "")
	}
	killSlotProcesses(workDir)
	deadlineClear := time.Now().Add(5 * time.Second)
	for portListening(wantLocal) && time.Now().Before(deadlineClear) {
		time.Sleep(50 * time.Millisecond)
	}
	if portListening(wantLocal) {
		// Second pass: Toolhelp can miss a path; PowerShell fallback inside killSlot.
		killSlotProcesses(workDir)
		deadlineClear = time.Now().Add(5 * time.Second)
		for portListening(wantLocal) && time.Now().Before(deadlineClear) {
			time.Sleep(50 * time.Millisecond)
		}
	}
	if portListening(wantLocal) {
		err := fmt.Errorf("localhost:%d still in use after clearing AuvikTunnel slot %s", wantLocal, workDir)
		m.log("error", domain, "port not free for tunnel", err.Error())
		return 0, err
	}

	spec := fmt.Sprintf("tcp:%d:%s:%d:%s", wantLocal, deviceIP, remotePort, domain)
	args := []string{"-r", "-l"} // persist listener; write TunnelClient.log beside the binary
	if m.Username != "" && m.APIKey != "" {
		args = append(args, "-u", m.Username+":"+m.APIKey)
	}
	if m.HostBase != "" {
		args = append(args, "-b", m.HostBase)
	}
	args = append(args, spec)

	m.log("info", domain, "starting tunnel", fmt.Sprintf("bin=%s workDir=%s args=%v", slotBin, workDir, redactTunnelArgs(args)))

	// Only drop mappings whose local port is gone — never mass-kill AuvikTunnel
	// processes (supervisor PID ≠ listener child PID).
	m.reapStaleTunnels()

	pid, cmd, err := startTunnelOS(slotBin, args)
	if err != nil {
		// Slot may still hold a stuck supervisor with no listener — clear once and retry.
		m.log("warn", domain, "start failed, clearing slot and retrying", err.Error())
		killSlotProcesses(workDir)
		pid, cmd, err = startTunnelOS(slotBin, args)
		if err != nil {
			m.log("error", domain, "start failed", err.Error())
			return 0, err
		}
	}

	exited := &atomic.Bool{}
	if cmd != nil {
		go func() {
			_ = cmd.Wait()
			exited.Store(true)
		}()
	}

	// Cap listen wait so a dead tunnel fails fast; still honor ctx (Cancel).
	listenWait := 25 * time.Second
	if dl, ok := ctx.Deadline(); ok {
		if remain := time.Until(dl); remain > 0 && remain < listenWait {
			listenWait = remain
		}
	}
	deadline := time.Now().Add(listenWait)
	var sawExit bool
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			killTunnelOS(pid, cmd)
			m.log("error", domain, "cancelled waiting for port", ctx.Err().Error())
			return 0, ctx.Err()
		}
		if portListening(wantLocal) {
			// Port is bound (checked without connecting). Sleep once — do not
			// keep probing — then confirm it is still held before SSH dials.
			if err := sleepCtx(ctx, tunnelListenSettle); err != nil {
				killTunnelOS(pid, cmd)
				m.log("error", domain, "cancelled waiting for port", err.Error())
				return 0, err
			}
			if portListening(wantLocal) {
				m.mu.Lock()
				m.active[key] = &tunnelProc{key: key, localPort: wantLocal, pid: pid, cmd: cmd, workDir: workDir, exited: exited, refs: 1}
				m.mu.Unlock()
				m.log("info", domain, fmt.Sprintf("listening localhost:%d → %s:%d (%s)", wantLocal, deviceIP, remotePort, time.Since(started).Round(time.Millisecond)), "")
				return wantLocal, nil
			}
			// Dropped during settle (supervisor swap); keep waiting.
			continue
		}
		if tunnelWaitUsesProcessExit() && exited.Load() {
			sawExit = true
			// Release supervisor sometimes swaps processes; give the replacement
			// a moment to bind before declaring failure.
			if err := sleepCtx(ctx, 500*time.Millisecond); err != nil {
				killTunnelOS(pid, cmd)
				m.log("error", domain, "cancelled waiting for port", err.Error())
				return 0, err
			}
			if portListening(wantLocal) {
				m.mu.Lock()
				m.active[key] = &tunnelProc{key: key, localPort: wantLocal, pid: pid, cmd: cmd, workDir: workDir, exited: exited, refs: 1}
				m.mu.Unlock()
				m.log("info", domain, fmt.Sprintf("listening localhost:%d → %s:%d (after relaunch, %s)", wantLocal, deviceIP, remotePort, time.Since(started).Round(time.Millisecond)), "")
				return wantLocal, nil
			}
			detail := readSiblingTunnelLog(workDir)
			m.log("error", domain, "AuvikTunnel exited before listen", detail)
	if detail != "" {
		return 0, fmt.Errorf("AuvikTunnel exited before opening port %d", wantLocal)
	}
	return 0, fmt.Errorf("AuvikTunnel exited before opening port %d (check %s\\TunnelClient.log)", wantLocal, workDir)
		}
		time.Sleep(50 * time.Millisecond)
	}
	killTunnelOS(pid, cmd)
	detail := readSiblingTunnelLog(workDir)
	m.log("error", domain, fmt.Sprintf("port %d not ready within %s (exited=%v)", wantLocal, listenWait, sawExit), detail)
	if detail != "" {
		return 0, fmt.Errorf("AuvikTunnel did not open local port %d within %s", wantLocal, listenWait)
	}
	return 0, fmt.Errorf("AuvikTunnel did not open local port %d within %s", wantLocal, listenWait)
}

// prepareTunnelSlot copies AuvikTunnel into a unique working directory so each
// device:port mapping can run concurrently (release supervisor is per-cwd).
func (m *TunnelManager) prepareTunnelSlot(srcBin, key string) (string, error) {
	root := m.tunnelSlotRoot()
	sum := sha1.Sum([]byte(key))
	dir := filepath.Join(root, hex.EncodeToString(sum[:8]))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create tunnel slot: %w", err)
	}
	dst := filepath.Join(dir, filepath.Base(srcBin))
	if sameTunnelBinary(srcBin, dst) {
		return dst, nil
	}
	if err := copyFile(srcBin, dst); err != nil {
		return "", fmt.Errorf("copy AuvikTunnel into slot: %w", err)
	}
	return dst, nil
}

func (m *TunnelManager) tunnelSlotRoot() string {
	if m.AppHome != "" {
		return filepath.Join(m.AppHome, "auvik-tunnels")
	}
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		return filepath.Join(local, "PathfinderSSH-MSP", "auvik-tunnels")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".pathfinderssh", "auvik-tunnels")
}

func sameTunnelBinary(src, dst string) bool {
	si, err1 := os.Stat(src)
	di, err2 := os.Stat(dst)
	if err1 != nil || err2 != nil {
		return false
	}
	return si.Size() == di.Size() && si.ModTime().Equal(di.ModTime())
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp := dst + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	_ = os.Remove(dst)
	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if si, err := os.Stat(src); err == nil {
		_ = os.Chtimes(dst, si.ModTime(), si.ModTime())
	}
	return nil
}

func readSiblingTunnelLog(dir string) string {
	for _, name := range []string{"TunnelClient.log", "TunnelRelease.log"} {
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || len(raw) == 0 {
			continue
		}
		lines := strings.Split(strings.TrimSpace(string(raw)), "\n")
		if len(lines) > 8 {
			lines = lines[len(lines)-8:]
		}
		// Never echo credentials if Auvik debug logged them.
		var out []string
		for _, ln := range lines {
			if strings.Contains(strings.ToLower(ln), "authenticating with") {
				out = append(out, "[redacted auth line]")
				continue
			}
			out = append(out, ln)
		}
		return strings.Join(out, " | ")
	}
	return ""
}

func (p *tunnelProc) alive() bool {
	if p == nil {
		return false
	}
	if p.exited != nil && p.exited.Load() {
		return false
	}
	if p.cmd != nil && p.cmd.Process != nil {
		return true
	}
	// Windows Start-Process path: track by pid + listen port.
	return p.pid > 0
}

func (m *TunnelManager) log(level, domain, message, detail string) {
	if m.AppHome == "" {
		return
	}
	ts := time.Now().Format(time.RFC3339)
	line := fmt.Sprintf(`{"time":%q,"source":"auvik-tunnel","level":%q,"client":%q,"message":%q,"detail":%q}`+"\n",
		ts, level, domain, message, detail)
	path := filepath.Join(m.AppHome, "logs", "msp-sync.log")
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	_, _ = f.WriteString(line)
	_ = f.Close()
}

// StopAll terminates tracked tunnels and any leftover AuvikTunnel processes
// under the slot root (covers orphans after a crash or missed Release).
func (m *TunnelManager) StopAll() {
	m.mu.Lock()
	procs := make([]*tunnelProc, 0, len(m.active))
	for k, p := range m.active {
		if p != nil {
			m.cancelIdleLocked(p)
			procs = append(procs, p)
		}
		delete(m.active, k)
	}
	root := m.tunnelSlotRoot()
	m.mu.Unlock()
	// Kill off the UI/quit path — taskkill and WMI are capped, but many
	// slots in series can still stall Quit past the emergency os.Exit.
	go func(procs []*tunnelProc, root string) {
		for _, p := range procs {
			killTunnelProc(p)
		}
		if root != "" {
			killSlotProcesses(root)
		}
	}(procs, root)
	m.log("info", "", "stopped all Auvik tunnels", root)
}

// Release drops one user of a tunnel mapping. When the last tab closes, the
// tunnel stays warm for tunnelIdleGrace so a quick reopen reuses it.
func (m *TunnelManager) Release(domain, deviceIP string, remotePort int) {
	domain = strings.TrimSpace(domain)
	deviceIP = strings.TrimSpace(deviceIP)
	if domain == "" || deviceIP == "" {
		return
	}
	if remotePort <= 0 {
		remotePort = 22
	}
	key := domain + ":" + deviceIP + ":" + strconv.Itoa(remotePort)
	m.mu.Lock()
	p := m.active[key]
	if p == nil {
		m.mu.Unlock()
		return
	}
	if p.refs > 0 {
		p.refs--
	}
	if p.refs > 0 {
		refs := p.refs
		port := p.localPort
		m.mu.Unlock()
		m.log("info", domain, fmt.Sprintf("release localhost:%d → %s:%d (refs=%d)", port, deviceIP, remotePort, refs), "")
		return
	}
	m.cancelIdleLocked(p)
	port := p.localPort
	p.idleStop = time.AfterFunc(tunnelIdleGrace, func() {
		m.evictIfIdle(key)
	})
	m.mu.Unlock()
	m.log("info", domain, fmt.Sprintf("idle localhost:%d → %s:%d (grace %s)", port, deviceIP, remotePort, tunnelIdleGrace), "")
}

func (m *TunnelManager) cancelIdleLocked(p *tunnelProc) {
	if p == nil || p.idleStop == nil {
		return
	}
	p.idleStop.Stop()
	p.idleStop = nil
}

func (m *TunnelManager) evictIfIdle(key string) {
	m.mu.Lock()
	p := m.active[key]
	if p == nil || p.refs > 0 {
		m.mu.Unlock()
		return
	}
	m.cancelIdleLocked(p)
	delete(m.active, key)
	m.mu.Unlock()
	go killTunnelProc(p)
	m.log("info", "", fmt.Sprintf("closed idle tunnel %s", key), "")
}

func killTunnelProc(p *tunnelProc) {
	if p == nil {
		return
	}
	if p.cmd != nil && p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}
	killTunnelOS(p.pid, nil)
	if p.workDir != "" {
		killSlotProcesses(p.workDir)
	}
}

// Invalidate drops a cached mapping and kills its slot so the next Ensure
// starts a fresh tunnel (used after SSH-via-tunnel handshake failures).
func (m *TunnelManager) Invalidate(domain, deviceIP string, remotePort int) {
	domain = strings.TrimSpace(domain)
	deviceIP = strings.TrimSpace(deviceIP)
	if remotePort <= 0 {
		remotePort = 22
	}
	key := domain + ":" + deviceIP + ":" + strconv.Itoa(remotePort)
	m.mu.Lock()
	p := m.active[key]
	if p != nil {
		m.cancelIdleLocked(p)
	}
	delete(m.active, key)
	m.mu.Unlock()
	if p == nil {
		return
	}
	go killTunnelProc(p)
	m.log("info", domain, fmt.Sprintf("invalidated localhost:%d → %s:%d", p.localPort, deviceIP, remotePort), "")
}

// reapStaleTunnels drops dead entries from m.active. It does not mass-kill
// other AuvikTunnel processes: the release supervisor relaunches a child with
// a different PID, so "kill untracked" tears down live tunnels when a second
// mapping is started.
func (m *TunnelManager) reapStaleTunnels() {
	m.mu.Lock()
	var dead []*tunnelProc
	for k, p := range m.active {
		if p != nil && portListening(p.localPort) {
			continue
		}
		if p != nil {
			dead = append(dead, p)
		}
		delete(m.active, k)
	}
	m.mu.Unlock()
	for _, p := range dead {
		killTunnelOS(p.pid, p.cmd)
	}
}
func redactTunnelArgs(args []string) []string {
	out := append([]string(nil), args...)
	for i := 0; i+1 < len(out); i++ {
		if out[i] == "-u" {
			if j := strings.IndexByte(out[i+1], ':'); j > 0 {
				out[i+1] = out[i+1][:j] + ":***"
			} else {
				out[i+1] = "***"
			}
		}
	}
	return out
}

func pickListenPort() (int, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// stableLocalPort maps a tunnel key to a deterministic high port so reconnects
// after a Pathfinder restart land on the same listener.
func stableLocalPort(key string) int {
	sum := sha1.Sum([]byte(key))
	// 42000–51999: avoid well-known services and typical ephemeral ranges.
	n := int(sum[0])<<8 | int(sum[1])
	return 42000 + n%10000
}

// MappingKey is domain:ip:remote used to reuse AuvikTunnel processes.
func MappingKey(domain, deviceIP string, remotePort int) string {
	if remotePort <= 0 {
		remotePort = 22
	}
	return strings.TrimSpace(domain) + ":" + strings.TrimSpace(deviceIP) + ":" + strconv.Itoa(remotePort)
}

// MappingListenPort is the localhost port AuvikTunnel binds for this mapping.
func MappingListenPort(domain, deviceIP string, remotePort int) int {
	return stableLocalPort(MappingKey(domain, deviceIP, remotePort))
}

// portListening reports whether 127.0.0.1:port is taken without Dial+Close
// (those probes abort Auvik work sessions).
func portListening(port int) bool {
	if port <= 0 {
		return false
	}
	ln, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err == nil {
		_ = ln.Close()
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "address already in use") ||
		strings.Contains(msg, "only one usage of each socket address")
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func IsAuvikManaged(n sessions.Node, appHome string) bool {
	if strings.TrimSpace(n.Host) == "" {
		return false
	}
	if ResolveTunnelDomain(appHome, n) == "" {
		return false
	}
	if n.AuvikUseTunnel {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(n.IntegrationSource), "auvik") {
		return true
	}
	if strings.TrimSpace(n.AuvikDeviceID) != "" {
		return true
	}
	return false
}

// ShouldUseTunnelFirst is true when an Auvik domain is known for this session
// (explicit Auvik fields, or customer-level domain via WithCustomerTunnel).
func ShouldUseTunnelFirst(n sessions.Node, appHome string) bool {
	if strings.TrimSpace(n.Host) == "" {
		return false
	}
	return ResolveTunnelDomain(appHome, n) != ""
}

func ShouldTryTunnel(n sessions.Node, err error, autoTunnel bool, appHome string) bool {
	if err == nil {
		return false
	}
	domain := ResolveTunnelDomain(appHome, n)
	host := strings.TrimSpace(n.Host)
	if domain == "" || host == "" {
		return false
	}
	// Domain present (including customer-inherited) or operator auto-tunnel.
	if !(domain != "" || autoTunnel || n.AuvikUseTunnel || IsAuvikManaged(n, appHome)) {
		return false
	}
	return isReachFailure(err)
}

func isReachFailure(err error) bool {
	msg := strings.ToLower(err.Error())
	for _, sub := range []string{
		"not reachable", "connection refused", "timed out", "timeout",
		"no route", "network unreachable", "unreachable", "connection reset",
		"actively refused", "i/o timeout", "wsasend", "wsaeconnrefused",
		"no such host", "host unreachable",
	} {
		if strings.Contains(msg, sub) {
			return true
		}
	}
	return false
}

func DialNodeViaTunnel(n sessions.Node, localPort int) sessions.Node {
	out := n.Normalize()
	out.Host = "127.0.0.1"
	out.Port = localPort
	return out
}
