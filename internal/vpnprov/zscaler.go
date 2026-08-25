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

func DefaultZscalerBin() string {
	var cands []string
	switch runtime.GOOS {
	case "windows":
		root := filepath.Join(os.Getenv("ProgramFiles"), "Zscaler", "ZSACli")
		cands = []string{filepath.Join(root, "ZSACli.exe"), filepath.Join(root, "ZSAcli.exe")}
	case "darwin":
		cands = []string{
			"/Applications/Zscaler/ZSACli/ZSACli",
			"/Applications/Zscaler/ZSACli/zsacli",
			"/Applications/Zscaler/Zscaler.app/Contents/MacOS/ZSACli",
		}
	default:
		cands = []string{"/opt/zscaler/ZSACli/ZSACli", "/usr/local/bin/ZSACli", "/usr/bin/ZSACli"}
	}
	for _, p := range cands {
		if fileExists(p) {
			return p
		}
	}
	if p, err := exec.LookPath("ZSACli"); err == nil {
		return p
	}
	return ""
}

// ParseZscalerName splits zpa or zpa:partner-user (official -u).
func ParseZscalerName(name string) (service, partner string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "zpa", ""
	}
	head, rest, ok := strings.Cut(name, ":")
	if !ok {
		if strings.EqualFold(name, "zpa") || strings.EqualFold(name, "zia") {
			return strings.ToLower(name), ""
		}
		return "zpa", name
	}
	svc := strings.ToLower(strings.TrimSpace(head))
	if svc == "" {
		svc = "zpa"
	}
	return svc, strings.TrimSpace(rest)
}

func zpaEnabled(raw string) bool {
	s := strings.ToLower(raw)
	if s == "" {
		return false
	}
	if strings.Contains(s, `"enabled":true`) || strings.Contains(s, `"enabled": true`) {
		return true
	}
	if strings.Contains(s, `"enabled":false`) || strings.Contains(s, `"enabled": false`) {
		return false
	}
	if strings.Contains(s, `"state":"enabled"`) || strings.Contains(s, `"state": "enabled"`) {
		return true
	}
	if strings.Contains(s, "disabled") {
		return false
	}
	return strings.Contains(s, "connected")
}

func zscalerStatus(ctx context.Context, bin, service string) (string, error) {
	out, err := winexec.CommandContext(ctx, bin, "status", "-s", service).CombinedOutput()
	raw := string(bytes.TrimSpace(out))
	if err != nil && raw == "" {
		return "", err
	}
	return raw, nil
}

func enableZscaler(ctx context.Context, bin, name string) error {
	svc, partner := ParseZscalerName(name)
	if svc != "zpa" {
		return fmt.Errorf("Zscaler: only Private Access (zpa) is switched; will not enable %s", svc)
	}
	bin = strings.TrimSpace(bin)
	if bin == "" {
		bin = DefaultZscalerBin()
	}
	if bin == "" || !fileExists(bin) {
		return fmt.Errorf("ZSACli.exe not found (Zscaler Client Connector 4.4+ with CLI enabled in the app profile)")
	}
	if raw, err := zscalerStatus(ctx, bin, "zpa"); err == nil && zpaEnabled(raw) && partner == "" {
		return nil
	}
	args := []string{"enable", "-s", "zpa"}
	if partner != "" {
		args = append(args, "-u", partner)
	}
	out, err := winexec.CommandContext(ctx, bin, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("ZSACli enable -s zpa: %w (%s)", err, bytes.TrimSpace(out))
	}
	return nil
}

func disableZPA(ctx context.Context, bin string) error {
	bin = strings.TrimSpace(bin)
	if bin == "" {
		bin = DefaultZscalerBin()
	}
	if bin == "" || !fileExists(bin) {
		return nil
	}
	// Official disable. Do not pass -p; if the profile requires a disable
	// password the CLI fails and the session still dials the original host.
	out, err := winexec.CommandContext(ctx, bin, "disable", "-s", "zpa").CombinedOutput()
	if err != nil {
		return fmt.Errorf("ZSACli disable -s zpa: %w (%s)", err, bytes.TrimSpace(out))
	}
	return nil
}

func listZscaler(bin string) []string {
	if strings.TrimSpace(bin) == "" {
		bin = DefaultZscalerBin()
	}
	if bin == "" || !fileExists(bin) {
		return nil
	}
	return []string{"zpa"}
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}
