//go:build windows

package auvik

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/scottpeterman/pathfinderssh/internal/winexec"
)

const (
	// Bounded so a wedged taskkill/WMI cannot freeze Pathfinder
	// (Ensure/Release/StopAll used to wait forever on these).
	tunnelKillTimeout  = 5 * time.Second
	tunnelStartTimeout = 20 * time.Second
)

// startTunnelOS launches AuvikTunnel via PowerShell Start-Process.
// Go's default CreateProcess kills Auvik's release supervisor ("supervisor
// process terminated"); Start-Process keeps supervisor + client alive (same as
// the official UI / manual start).
func startTunnelOS(bin string, args []string) (pid int, cmd *exec.Cmd, err error) {
	var b strings.Builder
	b.WriteString("@(")
	for i, a := range args {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(psSingleQuote(a))
	}
	b.WriteByte(')')

	script := fmt.Sprintf(
		"$p = Start-Process -FilePath %s -WorkingDirectory %s -WindowStyle Hidden -PassThru -ArgumentList %s; if ($null -eq $p) { exit 1 }; Write-Output $p.Id",
		psSingleQuote(bin),
		psSingleQuote(filepath.Dir(bin)),
		b.String(),
	)

	ctx, cancel := context.WithTimeout(context.Background(), tunnelStartTimeout)
	defer cancel()
	ps := winexec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-ExecutionPolicy", "Bypass", "-Command", script)
	out, err := ps.Output()
	if err != nil {
		var stderr []byte
		if ee, ok := err.(*exec.ExitError); ok {
			stderr = ee.Stderr
		}
		if ctx.Err() != nil {
			return 0, nil, fmt.Errorf("Start-Process AuvikTunnel timed out after %s", tunnelStartTimeout)
		}
		return 0, nil, fmt.Errorf("Start-Process AuvikTunnel: %w (%s)", err, bytes.TrimSpace(stderr))
	}
	pidStr := strings.TrimSpace(string(out))
	pid, err = strconv.Atoi(pidStr)
	if err != nil || pid <= 0 {
		return 0, nil, fmt.Errorf("Start-Process AuvikTunnel: bad pid %q", pidStr)
	}
	go hideSlotWindows(filepath.Dir(bin), 4*time.Second)
	return pid, nil, nil
}

func killTunnelOS(pid int, cmd *exec.Cmd) {
	if pid > 0 {
		// /T kills the release supervisor and its tunnel client child.
		runKillCmd(tunnelKillTimeout, "taskkill", "/F", "/T", "/PID", strconv.Itoa(pid))
		return
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}

// killSlotProcesses terminates any AuvikTunnel still running from this slot
// directory. Needed after Pathfinder restarts (in-memory pid map is empty) so
// a relaunch is not blocked by the one-supervisor-per-cwd rule.
//
// Uses PowerShell/CIM (not Toolhelp): Toolhelp path matching left dead listeners
// up, Ensure then treated them as "ready", and SSH handshakes timed out.
func killSlotProcesses(workDir string) {
	workDir = strings.TrimSpace(workDir)
	if workDir == "" {
		return
	}
	script := fmt.Sprintf(`
$root = [System.IO.Path]::GetFullPath(%s)
Get-CimInstance Win32_Process -Filter "Name='AuvikTunnel.exe'" | ForEach-Object {
  $exe = $_.ExecutablePath
  if (-not $exe) { return }
  try {
    $full = [System.IO.Path]::GetFullPath($exe)
  } catch { return }
  if ($full.StartsWith($root, [System.StringComparison]::OrdinalIgnoreCase)) {
    Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue
  }
}
`, psSingleQuote(workDir))
	runKillCmd(tunnelKillTimeout, "powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-ExecutionPolicy", "Bypass", "-Command", script)
}

func runKillCmd(timeout time.Duration, name string, args ...string) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	c := winexec.CommandContext(ctx, name, args...)
	_ = c.Run()
}

func psSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// slotTunnelPID returns the PID of AuvikTunnel.exe running from workDir, or 0.
func slotTunnelPID(workDir string) int {
	workDir = strings.TrimSpace(workDir)
	if workDir == "" {
		return 0
	}
	script := fmt.Sprintf(`
$root = [System.IO.Path]::GetFullPath(%s)
Get-CimInstance Win32_Process -Filter "Name='AuvikTunnel.exe'" | ForEach-Object {
  $exe = $_.ExecutablePath
  if (-not $exe) { return }
  try { $full = [System.IO.Path]::GetFullPath($exe) } catch { return }
  if ($full.StartsWith($root, [System.StringComparison]::OrdinalIgnoreCase)) {
    Write-Output $_.ProcessId
    break
  }
}
`, psSingleQuote(workDir))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := winexec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-ExecutionPolicy", "Bypass", "-Command", script).Output()
	if err != nil {
		return 0
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil || pid <= 0 {
		return 0
	}
	return pid
}

func tunnelWaitUsesProcessExit() bool {
	// PowerShell returns as soon as Start-Process succeeds; the tunnel outlives it.
	return false
}
