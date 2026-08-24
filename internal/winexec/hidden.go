//go:build !windows

package winexec

import (
	"context"
	"os/exec"
)

// Command returns an exec.Cmd (no window hiding on non-Windows).
func Command(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}

// CommandContext is Command with a deadline.
func CommandContext(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}
