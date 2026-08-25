//go:build !windows

package crtbridge

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// EnableAutostart registers lane serve (or lane-crt) at user login
// on macOS (launchd) or Linux (systemd --user). Other Unix is a no-op.
func EnableAutostart(agentExe string) error {
	agentExe, err := filepath.Abs(agentExe)
	if err != nil {
		return err
	}
	if st, err := os.Stat(agentExe); err != nil || st.IsDir() {
		return fmt.Errorf("autostart: agent %s not found", agentExe)
	}
	switch runtime.GOOS {
	case "darwin":
		return enableLaunchd(agentExe)
	case "linux":
		return enableSystemd(agentExe)
	default:
		return nil
	}
}

// DisableAutostart removes the login unit/plist.
func DisableAutostart() error {
	stopServePID()
	var last error
	switch runtime.GOOS {
	case "darwin":
		last = disableLaunchd()
	case "linux":
		last = disableSystemd()
	}
	return last
}

// StartAgent loads the login unit or starts the exe in the background.
func StartAgent(agentExe string) error {
	agentExe, err := filepath.Abs(agentExe)
	if err != nil {
		return err
	}
	switch runtime.GOOS {
	case "darwin":
		if err := enableLaunchd(agentExe); err == nil {
			return nil
		}
	case "linux":
		if err := enableSystemd(agentExe); err == nil {
			return nil
		}
	}
	args := append([]string{agentExe}, agentExtraArgs(agentExe)...)
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Start()
}

func StopAgent() error {
	stopServePID()
	switch runtime.GOOS {
	case "darwin":
		_ = disableLaunchd()
	case "linux":
		_ = exec.Command("systemctl", "--user", "stop", systemdUnitName).Run()
	}
	return nil
}

func RestartAgent(agentExe string) error {
	_ = StopAgent()
	return StartAgent(agentExe)
}

func ProcessRunning(image string) bool {
	_ = image
	pid := readServePID("")
	return pid > 0 && pid != os.Getpid()
}

func enableLaunchd(exe string) error {
	plist := launchdPlistPath()
	if plist == "" {
		return fmt.Errorf("home directory required")
	}
	if err := os.MkdirAll(filepath.Dir(plist), 0o755); err != nil {
		return err
	}
	args := append([]string{exe}, agentExtraArgs(exe)...)
	if err := os.WriteFile(plist, []byte(launchdPlist(args)), 0o644); err != nil {
		return err
	}
	domain := fmt.Sprintf("gui/%d", os.Getuid())
	_ = exec.Command("launchctl", "bootout", domain+"/"+launchdLabel).Run()
	cmd := exec.Command("launchctl", "bootstrap", domain, plist)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	cmd = exec.Command("launchctl", "load", "-w", plist)
	out2, err2 := cmd.CombinedOutput()
	if err2 != nil {
		return fmt.Errorf("launchctl: %w (%s; load %s)", err, bytes.TrimSpace(out), bytes.TrimSpace(out2))
	}
	return nil
}

func disableLaunchd() error {
	plist := launchdPlistPath()
	domain := fmt.Sprintf("gui/%d", os.Getuid())
	_ = exec.Command("launchctl", "bootout", domain+"/"+launchdLabel).Run()
	if plist != "" {
		_ = exec.Command("launchctl", "unload", "-w", plist).Run()
		if err := os.Remove(plist); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func enableSystemd(exe string) error {
	unit := systemdUnitPath()
	if unit == "" {
		return fmt.Errorf("home directory required")
	}
	if err := os.MkdirAll(filepath.Dir(unit), 0o755); err != nil {
		return err
	}
	body := systemdUnit(exe, agentExtraArgs(exe))
	if err := os.WriteFile(unit, []byte(body), 0o644); err != nil {
		return err
	}
	_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
	cmd := exec.Command("systemctl", "--user", "enable", "--now", systemdUnitName)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("wrote %s; systemctl enable: %w (%s)", unit, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func disableSystemd() error {
	_ = exec.Command("systemctl", "--user", "disable", "--now", systemdUnitName).Run()
	unit := systemdUnitPath()
	if unit == "" {
		return nil
	}
	if err := os.Remove(unit); err != nil && !os.IsNotExist(err) {
		return err
	}
	_ = exec.Command("systemctl", "--user", "daemon-reload").Run()
	return nil
}
