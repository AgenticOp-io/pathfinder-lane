//go:build !windows

package winexec

import "os/exec"

// Command returns an exec.Cmd (no window hiding on non-Windows).
func Command(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}
