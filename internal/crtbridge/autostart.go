package crtbridge

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/crtapp"
)

const (
	launchdLabel     = "io.agenticop.pflane-serve"
	systemdUnitName  = "pflane-serve.service"
	servePIDFileName = "pflane-serve.pid"
)

func isPflaneExe(exe string) bool {
	base := strings.ToLower(filepath.Base(exe))
	return base == "pflane" || base == "pflane.exe"
}

func agentExtraArgs(exe string) []string {
	if isPflaneExe(exe) {
		return []string{"serve"}
	}
	return nil
}

func servePIDPath(appHome string) string {
	if appHome == "" {
		appHome = crtapp.Home()
	}
	return filepath.Join(appHome, servePIDFileName)
}

func writeServePID(appHome string) error {
	return os.WriteFile(servePIDPath(appHome), []byte(strconv.Itoa(os.Getpid())+"\n"), 0o644)
}

func removeServePID(appHome string) {
	_ = os.Remove(servePIDPath(appHome))
}

func readServePID(appHome string) int {
	raw, err := os.ReadFile(servePIDPath(appHome))
	if err != nil {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil || n <= 0 {
		return 0
	}
	return n
}

func stopServePID() {
	pid := readServePID("")
	if pid <= 0 || pid == os.Getpid() {
		return
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return
	}
	_ = p.Kill()
	removeServePID("")
}

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, `"`, "&quot;")
	return s
}

func launchdPlistPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "Library", "LaunchAgents", launchdLabel+".plist")
}

func systemdUnitPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "systemd", "user", systemdUnitName)
}

func launchdPlist(args []string) string {
	logDir := crtapp.LogsDir()
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>` + launchdLabel + `</string>
  <key>ProgramArguments</key>
  <array>
`)
	for _, a := range args {
		b.WriteString("    <string>" + xmlEscape(a) + "</string>\n")
	}
	b.WriteString(`  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <dict>
    <key>Crashed</key>
    <true/>
  </dict>
  <key>StandardOutPath</key>
  <string>` + xmlEscape(filepath.Join(logDir, "pflane-serve.log")) + `</string>
  <key>StandardErrorPath</key>
  <string>` + xmlEscape(filepath.Join(logDir, "pflane-serve.err")) + `</string>
</dict>
</plist>
`)
	return b.String()
}

func systemdUnit(exe string, extra []string) string {
	cmd := systemdEscape(exe)
	for _, a := range extra {
		cmd += " " + systemdEscape(a)
	}
	return `[Unit]
Description=Pathfinder last-mile CRT agent (pflane serve)

[Service]
ExecStart=` + cmd + `
Restart=on-failure

[Install]
WantedBy=default.target
`
}

func systemdEscape(s string) string {
	if s == "" {
		return `""`
	}
	if !strings.ContainsAny(s, " \t\"'") {
		return s
	}
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}
