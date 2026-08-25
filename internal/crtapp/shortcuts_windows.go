//go:build windows

package crtapp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/winexec"
)

func startMenuLnk() string {
	return filepath.Join(os.Getenv("APPDATA"),
		"Microsoft", "Windows", "Start Menu", "Programs", ShortcutBase+".lnk")
}

func desktopLnk() string {
	return filepath.Join(os.Getenv("USERPROFILE"), "Desktop", ShortcutBase+".lnk")
}

// CreateShortcuts points Start Menu and Desktop at the CRT installer (reconfigure).
func CreateShortcuts(installerExe string) error {
	installerExe, err := filepath.Abs(installerExe)
	if err != nil {
		return err
	}
	work := filepath.Dir(installerExe)
	for _, lnk := range []string{startMenuLnk(), desktopLnk()} {
		if err := os.MkdirAll(filepath.Dir(lnk), 0o755); err != nil {
			return err
		}
		if err := writeShortcut(lnk, installerExe, work); err != nil {
			return err
		}
	}
	return nil
}

func RemoveShortcuts() {
	_ = os.Remove(startMenuLnk())
	_ = os.Remove(desktopLnk())
}

func writeShortcut(lnk, target, workdir string) error {
	ps := fmt.Sprintf(
		`$s = (New-Object -ComObject WScript.Shell).CreateShortcut(%s); $s.TargetPath = %s; $s.WorkingDirectory = %s; $s.WindowStyle = 1; $s.Description = 'Lane installer'; $s.Save()`,
		psQuote(lnk), psQuote(target), psQuote(workdir),
	)
	cmd := winexec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-ExecutionPolicy", "Bypass", "-Command", ps)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("shortcut: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func psQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
