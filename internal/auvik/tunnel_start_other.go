//go:build !windows

package auvik

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

func startTunnelOS(bin string, args []string) (pid int, cmd *exec.Cmd, err error) {
	cmd = exec.Command(bin, args...)
	cmd.Dir = filepath.Dir(bin)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return 0, nil, fmt.Errorf("start AuvikTunnel: %w", err)
	}
	if cmd.Process != nil {
		pid = cmd.Process.Pid
	}
	return pid, cmd, nil
}

func killTunnelOS(pid int, cmd *exec.Cmd) {
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
		return
	}
	if pid > 0 {
		_ = exec.Command("kill", fmt.Sprintf("%d", pid)).Run()
	}
}

func tunnelWaitUsesProcessExit() bool {
	return true
}

func killSlotProcesses(workDir string) {
	// Non-Windows: slot reuse is uncommon; best-effort no-op.
	_ = workDir
}
