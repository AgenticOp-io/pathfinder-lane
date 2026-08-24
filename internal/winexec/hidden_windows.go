//go:build windows

package winexec

import (
	"context"
	"os/exec"
	"syscall"
)

const createNoWindow = 0x08000000

func hideConsole(cmd *exec.Cmd) *exec.Cmd {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
	}
	return cmd
}

// Command returns an exec.Cmd that does not flash a console or PowerShell window.
func Command(name string, args ...string) *exec.Cmd {
	return hideConsole(exec.Command(name, args...))
}

// CommandContext is Command with a deadline, still with no console window.
func CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	return hideConsole(exec.CommandContext(ctx, name, args...))
}
