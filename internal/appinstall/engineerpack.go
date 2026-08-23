package appinstall

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/mspbranding"
)

// Engineer bundle: MSP client apps only — no admin setup/security tools.
var EngineerBundleTools = []string{"pathfinder", "pfseed"}

// EngineerPackOptions configures branded engineer standalone installer output.
type EngineerPackOptions struct {
	DestDir       string
	InstallerName string
}

// BuildEngineerPack writes an engineer distribution folder (standalone MSP client, no admin setup).
func BuildEngineerPack(opts EngineerPackOptions) (installExe string, err error) {
	dest := strings.TrimSpace(opts.DestDir)
	if dest == "" {
		return "", fmt.Errorf("destination folder required")
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return "", err
	}

	srcBin := BinDir()
	if st, err := os.Stat(srcBin); err != nil || !st.IsDir() {
		exe, e := osExecutable()
		if e != nil {
			return "", fmt.Errorf("no installed bundle at %s", srcBin)
		}
		srcBin = filepath.Dir(exe)
	}

	bundleDir := filepath.Join(dest, "bundle")
	if err := os.MkdirAll(bundleDir, 0o755); err != nil {
		return "", err
	}
	for _, tool := range EngineerBundleTools {
		src := filepath.Join(srcBin, exeName(tool))
		if st, err := os.Stat(src); err != nil || st.IsDir() {
			return "", fmt.Errorf("missing %s — build installers first", exeName(tool))
		}
		if err := copyFile(src, filepath.Join(bundleDir, exeName(tool))); err != nil {
			return "", fmt.Errorf("copy %s: %w", tool, err)
		}
	}

	root := Root()
	for _, pair := range []struct{ src, name string }{
		{filepath.Join(root, "msp-enrollment.json"), "msp-enrollment.json"},
		{filepath.Join(root, "msp-branding.json"), "msp-branding.json"},
		{filepath.Join(root, "msp-security-policy.json"), "msp-security-policy.json"},
		{filepath.Join(root, "msp-engineer-settings.json"), "msp-engineer-settings.json"},
		{filepath.Join(root, "logo.png"), "logo.png"},
	} {
		if err := copyIfPresent(pair.src, filepath.Join(dest, pair.name)); err != nil {
			return "", err
		}
	}

	name := strings.TrimSpace(opts.InstallerName)
	if name == "" {
		name = defaultEngineerInstallerName()
	}
	if !strings.HasSuffix(strings.ToLower(name), ".exe") {
		name += ".exe"
	}
	installExe = filepath.Join(dest, name)

	srcInstaller := filepath.Join(srcBin, exeName("pfengineer-install"))
	if st, err := os.Stat(srcInstaller); err != nil || st.IsDir() {
		return "", fmt.Errorf("pfengineer-install.exe missing — rebuild with -Targets installers")
	}
	if err := copyFile(srcInstaller, installExe); err != nil {
		return "", fmt.Errorf("copy engineer installer: %w", err)
	}

	readme := fmt.Sprintf(`PathfinderSSH MSP — engineer standalone installer

FOR ENGINEERS (not MSP admins):
1. Double-click %s
2. Click Install
3. Open Pathfinder and sign in with your work account

This package includes organization branding, sign-in, security policy, API settings, and Cursor AI.
Engineers do NOT run Azure/Google registration or security admin tools.
Add or change Auvik and other integrations in Pathfinder Settings → Tools.

FOR MSP ADMINS:
Use pfsetup-o365.exe or pfsetup-google.exe for full MSP setup.

Files:
  %s
  msp-enrollment.json
  msp-branding.json
  msp-security-policy.json
  msp-engineer-settings.json
  logo.png
  bundle\pathfinder.exe, pfseed.exe

`, name, name)
	if err := os.WriteFile(filepath.Join(dest, "README.txt"), []byte(readme), 0o644); err != nil {
		return "", err
	}
	return installExe, nil
}

func defaultEngineerInstallerName() string {
	if b, ok, _ := mspbranding.Load(); ok {
		if o := sanitizeFileStem(b.OrgDisplayName); o != "" {
			return o + "-Engineer-Install"
		}
	}
	return "PathfinderMSP-Engineer-Install"
}

var invalidStem = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func sanitizeFileStem(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	s = invalidStem.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-.")
	if len(s) > 48 {
		s = s[:48]
	}
	return s
}
