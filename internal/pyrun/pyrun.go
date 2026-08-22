// Package pyrun runs SecureCRT-style Python session scripts.
//
// User scripts may call crt.Screen.Send / WaitForString via a injected helper
// module. The runner translates those into Callbacks the host wires to a terminal.
package pyrun

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Callbacks are how the host applies Python script actions to a live session.
type Callbacks struct {
	Send    func(text string) error
	WaitFor func(ctx context.Context, pattern string, timeout time.Duration) error
}

// Run executes path with python, injecting a crt shim that prints protocol lines.
func Run(ctx context.Context, pythonBin, scriptPath string, cb Callbacks) error {
	if strings.TrimSpace(scriptPath) == "" {
		return fmt.Errorf("script path required")
	}
	if pythonBin == "" {
		pythonBin = findPython()
	}
	if pythonBin == "" {
		return fmt.Errorf("python not found on PATH (tried python3, python, py)")
	}
	dir, err := os.MkdirTemp("", "pf-pyrun-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	if err := os.WriteFile(filepath.Join(dir, "crt.py"), []byte(crtShim), 0o644); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, pythonBin, scriptPath)
	cmd.Env = append(os.Environ(), "PYTHONPATH="+dir+string(os.PathListSeparator)+os.Getenv("PYTHONPATH"))
	cmd.Dir = filepath.Dir(scriptPath)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if err := ctx.Err(); err != nil {
			_ = cmd.Process.Kill()
			return err
		}
		if err := applyLine(ctx, line, cb); err != nil {
			_ = cmd.Process.Kill()
			return err
		}
	}
	if err := sc.Err(); err != nil {
		_ = cmd.Process.Kill()
		return err
	}
	return cmd.Wait()
}

func applyLine(ctx context.Context, line string, cb Callbacks) error {
	switch {
	case strings.HasPrefix(line, "SEND:"):
		text := unescape(strings.TrimPrefix(line, "SEND:"))
		if cb.Send != nil {
			return cb.Send(text)
		}
	case strings.HasPrefix(line, "WAIT:"):
		rest := strings.TrimPrefix(line, "WAIT:")
		pat, timeoutMs := splitWait(rest)
		if cb.WaitFor != nil {
			to := time.Duration(timeoutMs) * time.Millisecond
			if to <= 0 {
				to = 30 * time.Second
			}
			return cb.WaitFor(ctx, pat, to)
		}
	case strings.HasPrefix(line, "DELAY:"):
		ms := atoi(strings.TrimPrefix(line, "DELAY:"))
		if ms > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(ms) * time.Millisecond):
			}
		}
	}
	return nil
}

func splitWait(rest string) (pattern string, timeoutMs int) {
	// WAIT:pattern|timeout_ms
	if i := strings.LastIndex(rest, "|"); i >= 0 {
		pattern = unescape(rest[:i])
		timeoutMs = atoi(rest[i+1:])
		return pattern, timeoutMs
	}
	return unescape(rest), 30000
}

func unescape(s string) string {
	s = strings.ReplaceAll(s, `\n`, "\n")
	s = strings.ReplaceAll(s, `\r`, "\r")
	s = strings.ReplaceAll(s, `\t`, "\t")
	return s
}

func atoi(s string) int {
	n := 0
	for _, c := range strings.TrimSpace(s) {
		if c < '0' || c > '9' {
			return n
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func findPython() string {
	for _, name := range []string{"python3", "python", "py"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	return ""
}

const crtShim = `
"""PathfinderSSH MSP SecureCRT-compatible shim for session scripts."""
import sys

def _emit(line):
    sys.stdout.write(line + "\n")
    sys.stdout.flush()

class _Screen:
    def Send(self, s):
        s = s.replace("\\", "\\\\").replace("\n", "\\n").replace("\r", "\\r").replace("\t", "\\t")
        _emit("SEND:" + s)
    def WaitForString(self, s, timeout=30):
        s = s.replace("\\", "\\\\").replace("\n", "\\n").replace("\r", "\\r").replace("\t", "\\t")
        ms = int(float(timeout) * 1000)
        _emit("WAIT:%s|%d" % (s, ms))

class _Dialog:
    def MessageBox(self, *a, **k):
        pass

class _crt:
    Screen = _Screen()
    Dialog = _Dialog()
    def Sleep(self, ms):
        _emit("DELAY:%d" % int(ms))

crt = _crt()
`
