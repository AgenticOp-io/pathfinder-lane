// Package fortivpn launches official FortiClient tools only.
// It does not reimplement SSL/IPsec or pass passwords on the command line.
package fortivpn

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/scottpeterman/pathfinderssh/internal/winexec"
)

func DefaultBin() string {
	var cands []string
	switch runtime.GOOS {
	case "windows":
		cands = []string{
			filepath.Join(os.Getenv("ProgramFiles"), "Fortinet", "FortiClient", "FortiVPN.exe"),
			filepath.Join(os.Getenv("ProgramFiles"), "Fortinet", "FortiClient", "fortivpn.exe"),
		}
	case "darwin":
		cands = []string{
			"/Applications/FortiClient.app/Contents/Resources/FortiVPN",
			"/Applications/FortiClient.app/Contents/MacOS/FortiVPN",
		}
	default:
		cands = []string{"/opt/forticlient/fortivpn", "/usr/bin/fortivpn", "/usr/bin/forticlient"}
	}
	for _, p := range cands {
		if fileExists(p) {
			return p
		}
	}
	return ""
}

func DefaultTools() string {
	home, _ := os.UserHomeDir()
	local := os.Getenv("LOCALAPPDATA")
	name := "FortiSSLVPNclient.exe"
	cands := []string{
		filepath.Join(local, "PathfinderCRT-Bridge", "bin", name),
		filepath.Join(home, "FortiClientTools", "SSLVPNcmdline", "x64", name),
		filepath.Join(home, "Downloads", "SSLVPNcmdline", "x64", name),
	}
	for _, p := range cands {
		if fileExists(p) {
			return p
		}
	}
	return ""
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

var (
	cliMu    sync.Mutex
	cliCache = map[string]bool{}
	switchMu sync.Mutex
	lastOK   string
)

// SupportsCLI reports whether FortiVPN.exe accepts --cli (7.4.7+ / 8.0).
func SupportsCLI(bin string) bool {
	bin = strings.TrimSpace(bin)
	if bin == "" || !fileExists(bin) {
		return false
	}
	cliMu.Lock()
	if v, ok := cliCache[bin]; ok {
		cliMu.Unlock()
		return v
	}
	cliMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	out, _ := winexec.CommandContext(ctx, bin, "--cli", "--list").CombinedOutput()
	s := strings.ToLower(string(out))
	ok := true
	if strings.Contains(s, "does not exist") || strings.Contains(s, "error parsing options") {
		ok = false
	} else if strings.Contains(s, "ssl:") || strings.Contains(s, "ipsec:") || strings.Contains(s, "--connect") {
		ok = true
	} else if ctx.Err() != nil || strings.Contains(s, "usage:") {
		ok = false
	}
	cliMu.Lock()
	cliCache[bin] = ok
	cliMu.Unlock()
	return ok
}

// StatusLine is one FortiVPN --cli --status row ("name :: State").
type StatusLine struct {
	Name  string
	State string
}

func ParseStatus(raw string) []StatusLine {
	var out []StatusLine
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" {
			continue
		}
		name, state, ok := strings.Cut(line, " :: ")
		if !ok {
			continue
		}
		name, state = strings.TrimSpace(name), strings.TrimSpace(state)
		if name == "" || state == "" {
			continue
		}
		out = append(out, StatusLine{Name: name, State: state})
	}
	return out
}

func activeState(state string) bool {
	switch strings.ToLower(strings.TrimSpace(state)) {
	case "connected", "connecting":
		return true
	default:
		return false
	}
}

func connectedState(state string) bool {
	return strings.EqualFold(strings.TrimSpace(state), "connected")
}

// Plan reports whether the wanted tunnel is already up, and whether any other
// tunnel must be disconnected first (FortiClient allows one VPN at a time).
func Plan(lines []StatusLine, want string) (alreadyUp, disconnectFirst bool) {
	want = strings.TrimSpace(want)
	if want == "" {
		return true, false
	}
	var wantState string
	for _, l := range lines {
		if strings.EqualFold(l.Name, want) {
			wantState = l.State
			continue
		}
		if activeState(l.State) {
			disconnectFirst = true
		}
	}
	if disconnectFirst {
		return false, true
	}
	return connectedState(wantState), false
}

// ParseList reads FortiVPN.exe --cli --list (official SSL: / IPSEC: sections).
func ParseList(raw string) []string {
	seen := map[string]bool{}
	var out []string
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		key := strings.ToLower(name)
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, name)
	}
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if line == "" {
			continue
		}
		low := strings.ToLower(line)
		if strings.HasPrefix(low, "error") || strings.HasPrefix(low, "usage") ||
			strings.Contains(low, "does not exist") || strings.HasPrefix(low, "options:") ||
			strings.HasPrefix(low, "fortivpn.exe") || strings.HasPrefix(low, "forticlient vpn") {
			continue
		}
		if name, _, ok := strings.Cut(line, " :: "); ok {
			add(name)
			continue
		}
		if strings.Contains(line, ":") && !strings.Contains(line, "://") {
			head, rest, _ := strings.Cut(line, ":")
			rest = strings.TrimSpace(rest)
			if rest != "" {
				add(rest)
				continue
			}
			h := strings.ToLower(strings.TrimSpace(head))
			if h == "ssl" || h == "ipsec" || strings.HasPrefix(h, "ssl ") || strings.HasPrefix(h, "ipsec ") {
				continue
			}
		}
		add(line)
	}
	return out
}

// ListTunnels runs official FortiVPN.exe --cli --list (7.4.7+ / 8.0).
func ListTunnels(ctx context.Context, bin string) ([]string, error) {
	bin = strings.TrimSpace(bin)
	if bin == "" {
		bin = DefaultBin()
	}
	if bin == "" || !fileExists(bin) {
		return nil, fmt.Errorf("FortiVPN.exe not found")
	}
	out, err := winexec.CommandContext(ctx, bin, "--cli", "--list").CombinedOutput()
	raw := string(bytes.TrimSpace(out))
	low := strings.ToLower(raw)
	if strings.Contains(low, "option 'cli' does not exist") || strings.Contains(low, "error parsing options") {
		return nil, fmt.Errorf("this FortiClient has no --cli (need 7.4.7+). Type the connection name from the FortiClient GUI, or install FortiClientTools")
	}
	names := ParseList(raw)
	if len(names) == 0 {
		if st, stErr := queryStatus(ctx, bin); stErr == nil {
			names = ParseList(st)
		}
	}
	if len(names) == 0 && err != nil {
		return nil, fmt.Errorf("FortiVPN --cli --list: %w (%s)", err, raw)
	}
	return names, nil
}

func queryStatus(ctx context.Context, bin string) (string, error) {
	out, err := winexec.CommandContext(ctx, bin, "--cli", "--status").CombinedOutput()
	raw := string(bytes.TrimSpace(out))
	if err != nil && raw == "" {
		return "", err
	}
	return raw, nil
}

func disconnectAll(ctx context.Context, bin string) error {
	// Official: omit --tunnel to disconnect whatever is currently connected.
	out, err := winexec.CommandContext(ctx, bin, "--cli", "--disconnect").CombinedOutput()
	if err != nil {
		return fmt.Errorf("FortiVPN disconnect: %w (%s)", err, bytes.TrimSpace(out))
	}
	return nil
}

func connectTunnel(ctx context.Context, bin, tunnel string) error {
	// Do not use --keeprunning: that holds the process until disconnect and
	// blocks automation. FortiClient keeps the tunnel after connect returns.
	// Do not pass --password.
	out, err := winexec.CommandContext(ctx, bin, "--cli", "--connect", "--tunnel", tunnel).CombinedOutput()
	if err != nil {
		return fmt.Errorf("FortiVPN connect %s: %w (%s)", tunnel, err, bytes.TrimSpace(out))
	}
	return nil
}

func waitStatus(ctx context.Context, bin, tunnel string, wantConnected bool) error {
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		raw, err := queryStatus(ctx, bin)
		if err == nil {
			up, drop := Plan(ParseStatus(raw), tunnel)
			if wantConnected && up {
				return nil
			}
			if !wantConnected && !up && !drop {
				return nil
			}
		}
		time.Sleep(1 * time.Second)
	}
	if wantConnected {
		return fmt.Errorf("FortiClient tunnel %q did not reach Connected (saved credentials / MFA in FortiClient if required)", tunnel)
	}
	return fmt.Errorf("FortiClient did not disconnect in time")
}

func DisconnectTools(ctx context.Context, tools string) error {
	tools = strings.TrimSpace(tools)
	if tools == "" || !fileExists(tools) {
		return fmt.Errorf("FortiClientTools FortiSSLVPNclient.exe not found")
	}
	out, err := winexec.CommandContext(ctx, tools, "disconnect").CombinedOutput()
	if err != nil {
		return fmt.Errorf("FortiSSLVPNclient disconnect: %w (%s)", err, bytes.TrimSpace(out))
	}
	return nil
}

func ConnectTools(ctx context.Context, tools, tunnel string) error {
	tools = strings.TrimSpace(tools)
	tunnel = strings.TrimSpace(tunnel)
	if tools == "" || !fileExists(tools) {
		return fmt.Errorf("FortiClientTools FortiSSLVPNclient.exe not found")
	}
	if tunnel == "" {
		return fmt.Errorf("FortiClient tunnel name required")
	}
	// Saved connection in FortiClientTools; -q -m avoid message boxes / extra clicks.
	cmd := winexec.CommandContext(ctx, tools, "connect", "-s", tunnel, "-q", "-m")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("FortiSSLVPNclient connect %s: %w (%s)", tunnel, err, bytes.TrimSpace(out))
	}
	return nil
}

// DisconnectActive tears down whatever FortiClient tunnel is up, if a CLI
// exists. Missing tools are not an error (another provider may be in use).
func DisconnectActive(ctx context.Context, bin, tools string) error {
	if bin == "" {
		bin = DefaultBin()
	}
	if tools == "" {
		tools = DefaultTools()
	}
	if SupportsCLI(bin) {
		return disconnectAll(ctx, bin)
	}
	if tools != "" && fileExists(tools) {
		return DisconnectTools(ctx, tools)
	}
	return nil
}

// Ensure brings the named tunnel up automatically. If another customer VPN is
// connected, it disconnects first, then connects the mapped tunnel. No UI click.
func Ensure(ctx context.Context, bin, tools, tunnel string) error {
	tunnel = strings.TrimSpace(tunnel)
	if tunnel == "" {
		return nil
	}
	if bin == "" {
		bin = DefaultBin()
	}
	if tools == "" {
		tools = DefaultTools()
	}

	switchMu.Lock()
	defer switchMu.Unlock()

	if SupportsCLI(bin) {
		raw, err := queryStatus(ctx, bin)
		if err != nil {
			return err
		}
		already, drop := Plan(ParseStatus(raw), tunnel)
		if already {
			lastOK = tunnel
			return nil
		}
		if drop {
			if err := disconnectAll(ctx, bin); err != nil {
				return err
			}
			if err := waitStatus(ctx, bin, tunnel, false); err != nil {
				return err
			}
		}
		if err := connectTunnel(ctx, bin, tunnel); err != nil {
			return err
		}
		if err := waitStatus(ctx, bin, tunnel, true); err != nil {
			return err
		}
		lastOK = tunnel
		return nil
	}

	if tools != "" {
		if lastOK != "" && strings.EqualFold(lastOK, tunnel) {
			return nil
		}
		_ = DisconnectTools(ctx, tools)
		time.Sleep(2 * time.Second)
		if err := ConnectTools(ctx, tools, tunnel); err != nil {
			return err
		}
		lastOK = tunnel
		return nil
	}
	return fmt.Errorf("no FortiClient CLI: install FortiClient 7.4.7+ or FortiClientTools SSLVPNcmdline")
}
