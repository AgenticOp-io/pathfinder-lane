//go:build windows

package crtbridge

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/winexec"
)

func startupLnks() []string {
	dir := filepath.Join(os.Getenv("APPDATA"),
		"Microsoft", "Windows", "Start Menu", "Programs", "Startup")
	return []string{
		filepath.Join(dir, "Pathfinder CRT Bridge.lnk"),
		filepath.Join(dir, "PathfinderSSH CRT Bridge.lnk"),
	}
}

func startupLnk() string {
	return startupLnks()[0]
}

// EnableAutostart launches the CRT bridge agent at user logon.
func EnableAutostart(agentExe string) error {
	agentExe, err := filepath.Abs(agentExe)
	if err != nil {
		return err
	}
	_ = DisableAutostart()
	lnk := startupLnk()
	if err := os.MkdirAll(filepath.Dir(lnk), 0o755); err != nil {
		return err
	}
	args := strings.Join(agentExtraArgs(agentExe), " ")
	return writeStartupShortcut(lnk, agentExe, filepath.Dir(agentExe), args)
}

// DisableAutostart removes the logon shortcut.
func DisableAutostart() error {
	stopServePID()
	var last error
	for _, lnk := range startupLnks() {
		if err := os.Remove(lnk); err != nil && !os.IsNotExist(err) {
			last = err
		}
	}
	return last
}

// StartAgent launches the companion in the background (no console window).
func StartAgent(agentExe string) error {
	agentExe, err := filepath.Abs(agentExe)
	if err != nil {
		return err
	}
	extra := strings.Join(agentExtraArgs(agentExe), " ")
	argBit := ""
	if extra != "" {
		argBit = " -ArgumentList " + psQuote(extra)
	}
	ps := fmt.Sprintf(
		`Start-Process -FilePath %s -WorkingDirectory %s%s -WindowStyle Hidden`,
		psQuote(agentExe), psQuote(filepath.Dir(agentExe)), argBit,
	)
	cmd := winexec.Command("powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden",
		"-ExecutionPolicy", "Bypass", "-Command", ps)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("start CRT bridge: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// StopAgent ends any running pathfinder-crt agent so binaries and session
// files can be updated.
func StopAgent() error {
	stopServePID()
	cmd := winexec.Command("taskkill", "/IM", "pathfinder-crt.exe", "/F")
	out, err := cmd.CombinedOutput()
	s := strings.ToLower(string(out))
	if err == nil || strings.Contains(s, "not found") || strings.Contains(s, "not running") {
		return nil
	}
	return fmt.Errorf("stop CRT agent: %w (%s)", err, strings.TrimSpace(string(out)))
}

// RestartAgent stops the old agent (if any) and starts exe.
func RestartAgent(agentExe string) error {
	_ = StopAgent()
	return StartAgent(agentExe)
}

func ProcessRunning(image string) bool {
	image = strings.TrimSpace(image)
	if image == "" {
		return false
	}
	cmd := winexec.Command("tasklist", "/NH", "/FI", "IMAGENAME eq "+image)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToLower(string(out)), strings.ToLower(image))
}

func writeStartupShortcut(lnk, target, workdir, arguments string) error {
	argPS := ""
	if strings.TrimSpace(arguments) != "" {
		argPS = fmt.Sprintf("; $s.Arguments = %s", psQuote(arguments))
	}
	ps := fmt.Sprintf(
		`$s = (New-Object -ComObject WScript.Shell).CreateShortcut(%s); $s.TargetPath = %s; $s.WorkingDirectory = %s%s; $s.WindowStyle = 7; $s.Description = 'Pathfinder CRT Bridge agent'; $s.Save()`,
		psQuote(lnk), psQuote(target), psQuote(workdir), argPS,
	)
	cmd := winexec.Command("powershell", "-NoProfile", "-WindowStyle", "Hidden", "-ExecutionPolicy", "Bypass", "-Command", ps)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("startup shortcut: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func psQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
