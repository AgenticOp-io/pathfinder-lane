//go:build windows

package winexec

import (
	"os/exec"
	"syscall"
)

const createNoWindow = 0x08000000

// Command returns an exec.Cmd that does not flash a console or PowerShell window.
func Command(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow,
	}
	return cmd
}
