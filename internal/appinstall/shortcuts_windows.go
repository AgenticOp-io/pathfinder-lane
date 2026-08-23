//go:build windows

package appinstall

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/product"
	"github.com/scottpeterman/pathfinderssh/internal/winexec"
)

// CreateShortcuts writes Start Menu and Desktop .lnk files pointing at exe.
func CreateShortcuts(exe string) error {
	exe, err := filepath.Abs(exe)
	if err != nil {
		return err
	}
	work := Root()
	_ = os.MkdirAll(work, 0o755)

	lnk := product.ShortcutBase + ".lnk"
	startMenu := startMenuDir()
	_ = os.MkdirAll(startMenu, 0o755)
	if err := writeShortcut(filepath.Join(startMenu, lnk), exe, work); err != nil {
		return err
	}

	desktop := desktopDir()
	if desktop != "" {
		_ = writeShortcut(filepath.Join(desktop, lnk), exe, work)
	}
	// Installer shortcut when pfinstall is bundled.
	inst := filepath.Join(filepath.Dir(exe), exeName("pfinstall"))
	if st, err := os.Stat(inst); err == nil && !st.IsDir() {
		instLnk := "Install PathfinderSSH.lnk"
		_ = writeShortcut(filepath.Join(startMenu, instLnk), inst, work)
		if desktop != "" {
			_ = writeShortcut(filepath.Join(desktop, instLnk), inst, work)
		}
	}
	return removeObsoleteShortcuts()
}

func removeShortcuts() error {
	_ = removeObsoleteShortcuts()
	lnk := product.ShortcutBase + ".lnk"
	_ = os.Remove(filepath.Join(startMenuDir(), lnk))
	if d := desktopDir(); d != "" {
		_ = os.Remove(filepath.Join(d, lnk))
	}
	// Also clear the pre-MSP shortcut names so upgrades do not leave two icons.
	for _, old := range []string{"PathfinderSSH.lnk", "Install PathfinderSSH.lnk"} {
		_ = os.Remove(filepath.Join(startMenuDir(), old))
		if d := desktopDir(); d != "" {
			_ = os.Remove(filepath.Join(d, old))
		}
	}
	return nil
}

func removeObsoleteShortcuts() error {
	startMenu := startMenuDir()
	desktop := desktopDir()
	for _, p := range []string{
		filepath.Join(startMenu, "Pathfinder Setup.lnk"),
		filepath.Join(startMenu, "Pathfinder Hub.lnk"),
	} {
		_ = os.Remove(p)
	}
	if desktop != "" {
		for _, name := range []string{"Pathfinder Setup.lnk", "Pathfinder Hub.lnk"} {
			_ = os.Remove(filepath.Join(desktop, name))
		}
	}
	return nil
}

func startMenuDir() string {
	return filepath.Join(os.Getenv("APPDATA"), "Microsoft", "Windows", "Start Menu", "Programs")
}

func desktopDir() string {
	home, err := os.UserHomeDir()
	if err == nil {
		for _, cand := range []string{
			filepath.Join(home, "OneDrive", "Desktop"),
			filepath.Join(home, "Desktop"),
		} {
			if st, err := os.Stat(cand); err == nil && st.IsDir() {
				return cand
			}
		}
	}
	out, err := winexec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-Command",
		"[Environment]::GetFolderPath('Desktop')").Output()
	if err == nil {
		p := strings.TrimSpace(string(out))
		if p != "" {
			if st, err := os.Stat(p); err == nil && st.IsDir() {
				return p
			}
		}
	}
	return ""
}

func writeShortcut(lnk, target, workdir string) error {
	ps := fmt.Sprintf(
		`$s = (New-Object -ComObject WScript.Shell).CreateShortcut(%s); $s.TargetPath = %s; $s.WorkingDirectory = %s; $s.Description = %s; $s.Save()`,
		psQuote(lnk), psQuote(target), psQuote(workdir), psQuote(product.Name),
	)
	cmd := winexec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-ExecutionPolicy", "Bypass", "-Command", ps)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("shortcut %s: %w (%s)", lnk, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func psQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
