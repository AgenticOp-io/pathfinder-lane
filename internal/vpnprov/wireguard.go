package vpnprov

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/winexec"
)

func execLookPath(name string) (string, error) {
	return exec.LookPath(name)
}

const wgServicePrefix = "WireGuardTunnel$"

func DefaultWireGuardBin() string {
	switch runtime.GOOS {
	case "windows":
		p := filepath.Join(os.Getenv("ProgramFiles"), "WireGuard", "wireguard.exe")
		if fileExists(p) {
			return p
		}
	default:
		if p, err := execLookPath("wg-quick"); err == nil {
			return p
		}
	}
	return ""
}

func wgDataDir() string {
	return filepath.Join(os.Getenv("ProgramFiles"), "WireGuard", "Data", "Configurations")
}

func unixWGDirs() []string {
	home, _ := os.UserHomeDir()
	return []string{
		"/etc/wireguard",
		"/usr/local/etc/wireguard",
		"/opt/homebrew/etc/wireguard",
		filepath.Join(home, ".config", "wireguard"),
		filepath.Join(home, "Library", "Application Support", "WireGuard"),
	}
}

func listConfDir(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		files = append(files, e.Name())
	}
	return ParseWGConfs(files)
}

// ParseWGServices reads `sc query` output for WireGuardTunnel$ names.
func ParseWGServices(raw string) []string {
	seen := map[string]bool{}
	var out []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		_, name, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		if !strings.HasPrefix(name, wgServicePrefix) {
			continue
		}
		tun := strings.TrimPrefix(name, wgServicePrefix)
		if tun == "" || seen[strings.ToLower(tun)] {
			continue
		}
		seen[strings.ToLower(tun)] = true
		out = append(out, tun)
	}
	return out
}

// ParseWGConfs lists tunnel names from WireGuard configuration filenames
// without opening the files (keys stay on disk).
func ParseWGConfs(names []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, n := range names {
		base := filepath.Base(n)
		switch {
		case strings.HasSuffix(strings.ToLower(base), ".conf.dpapi"):
			base = strings.TrimSuffix(strings.TrimSuffix(base, ".dpapi"), ".conf")
		case strings.HasSuffix(strings.ToLower(base), ".conf"):
			base = strings.TrimSuffix(base, ".conf")
		default:
			continue
		}
		base = strings.TrimSpace(base)
		if base == "" || seen[strings.ToLower(base)] {
			continue
		}
		seen[strings.ToLower(base)] = true
		out = append(out, base)
	}
	return out
}

func listWireGuard(ctx context.Context) []string {
	seen := map[string]bool{}
	var out []string
	add := func(names []string) {
		for _, n := range names {
			k := strings.ToLower(n)
			if n == "" || seen[k] {
				continue
			}
			seen[k] = true
			out = append(out, n)
		}
	}
	if runtime.GOOS == "windows" {
		raw, err := winexec.CommandContext(ctx, "sc", "query", "type=", "service", "state=", "all").CombinedOutput()
		if err == nil {
			add(ParseWGServices(string(raw)))
		}
		add(listConfDir(wgDataDir()))
		return out
	}
	if raw, err := winexec.CommandContext(ctx, "wg", "show", "interfaces").CombinedOutput(); err == nil {
		add(strings.Fields(string(raw)))
	}
	for _, dir := range unixWGDirs() {
		add(listConfDir(dir))
	}
	return out
}

func wgService(name string) string {
	return wgServicePrefix + strings.TrimSpace(name)
}

func wgRunning(ctx context.Context, name string) bool {
	raw, err := winexec.CommandContext(ctx, "sc", "query", wgService(name)).CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToUpper(string(raw)), "RUNNING")
}

func stopWireGuard(ctx context.Context, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}
	if runtime.GOOS != "windows" {
		out, err := winexec.CommandContext(ctx, "wg-quick", "down", name).CombinedOutput()
		if err != nil && !wgUnixDownOK(string(out)) {
			return fmt.Errorf("wg-quick down %s: %w (%s)", name, err, bytes.TrimSpace(out))
		}
		return nil
	}
	out, err := winexec.CommandContext(ctx, "sc", "stop", wgService(name)).CombinedOutput()
	if err != nil && !wgStopOK(string(out)) {
		return fmt.Errorf("WireGuard stop %s: %w (%s)", name, err, bytes.TrimSpace(out))
	}
	return nil
}

func wgUnixDownOK(raw string) bool {
	s := strings.ToLower(raw)
	return strings.Contains(s, "is not a") || strings.Contains(s, "unable to access") || strings.Contains(s, "does not exist")
}

func wgStopOK(raw string) bool {
	s := strings.ToLower(raw)
	return strings.Contains(s, "1062") || strings.Contains(s, "not been started") || strings.Contains(s, "does not exist")
}

func startWireGuard(ctx context.Context, bin, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("WireGuard tunnel name required")
	}
	if runtime.GOOS != "windows" {
		out, err := winexec.CommandContext(ctx, "wg-quick", "up", name).CombinedOutput()
		if err != nil && !strings.Contains(strings.ToLower(string(out)), "already exists") {
			return fmt.Errorf("wg-quick up %s: %w (%s)", name, err, bytes.TrimSpace(out))
		}
		return nil
	}
	if wgRunning(ctx, name) {
		return nil
	}
	out, err := winexec.CommandContext(ctx, "sc", "start", wgService(name)).CombinedOutput()
	if err == nil {
		return nil
	}
	raw := string(bytes.TrimSpace(out))
	low := strings.ToLower(raw)
	if !strings.Contains(low, "does not exist") && !strings.Contains(low, "1060") {
		return fmt.Errorf("WireGuard start %s: %w (%s)", name, err, raw)
	}
	bin = strings.TrimSpace(bin)
	if bin == "" {
		bin = DefaultWireGuardBin()
	}
	conf := wgConfPath(name)
	if bin == "" || conf == "" {
		return fmt.Errorf("WireGuard tunnel %q is not installed as a service; add it in the WireGuard app, or run as admin so wireguard.exe /installtunnelservice can register it", name)
	}
	inst, ierr := winexec.CommandContext(ctx, bin, "/installtunnelservice", conf).CombinedOutput()
	if ierr != nil {
		return fmt.Errorf("wireguard /installtunnelservice %s: %w (%s)", name, ierr, bytes.TrimSpace(inst))
	}
	out, err = winexec.CommandContext(ctx, "sc", "start", wgService(name)).CombinedOutput()
	if err != nil {
		return fmt.Errorf("WireGuard start %s: %w (%s)", name, err, bytes.TrimSpace(out))
	}
	return nil
}

func wgConfPath(name string) string {
	dir := wgDataDir()
	for _, n := range []string{name + ".conf.dpapi", name + ".conf"} {
		p := filepath.Join(dir, n)
		if fileExists(p) {
			return p
		}
	}
	return ""
}

func stopOtherWireGuard(ctx context.Context, keep string) {
	keep = strings.ToLower(strings.TrimSpace(keep))
	for _, n := range listWireGuard(ctx) {
		if keep != "" && strings.EqualFold(n, keep) {
			continue
		}
		_ = stopWireGuard(ctx, n)
	}
}
