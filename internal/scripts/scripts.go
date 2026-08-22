// Package scripts is Pathfinder's session scripting model: named sequences of
// send / wait / delay steps stored as YAML under ~/.pathfinderssh/scripts.yaml.
//
// The Fyne host wires Send and Wait into open terminals; this package never
// imports the UI toolkit. Prompt-aware automation for crawls stays in netexec —
// these scripts inject keystrokes the same way the button bar does, and can
// pause until the active session prints a marker.
package scripts

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const FileName = "scripts.yaml"

// DefaultWaitTimeoutMs is used when a wait_for / wait_regex step omits timeout_ms.
const DefaultWaitTimeoutMs = 30000

// Step is one action in a script.
//
// Execution order within a step:
//  1. Send (if non-empty)
//  2. Wait for wait_for / wait_regex (if set) — watches the active terminal
//  3. DelayMs sleep (if > 0), including after a successful wait
type Step struct {
	// Send is literal text pushed to the session. Use \n for Return.
	Send string `yaml:"send,omitempty"`
	// DelayMs waits after the send/wait (or alone if both are empty).
	DelayMs int `yaml:"delay_ms,omitempty"`
	// WaitFor pauses until this literal substring appears in session output.
	WaitFor string `yaml:"wait_for,omitempty"`
	// WaitRegex pauses until this RE2 pattern matches session output.
	// Ignored when WaitFor is also set (literal wins).
	WaitRegex string `yaml:"wait_regex,omitempty"`
	// TimeoutMs bounds a wait. 0 means DefaultWaitTimeoutMs.
	TimeoutMs int `yaml:"timeout_ms,omitempty"`
}

// Script is a named runnable sequence.
type Script struct {
	Name string `yaml:"name"`
	// Scope is "active" (default) or "all" SSH tabs for Send.
	// Wait always watches the active tab.
	Scope string `yaml:"scope,omitempty"`
	Steps []Step `yaml:"steps"`
}

// File is the on-disk envelope.
type File struct {
	Version int      `yaml:"version"`
	Scripts []Script `yaml:"scripts"`
}

// Defaults seeds a few network-gear helpers.
func Defaults() File {
	return File{
		Version: 1,
		Scripts: []Script{
			{
				Name:  "Disable paging (Cisco)",
				Scope: "active",
				Steps: []Step{
					{Send: "terminal length 0\n", WaitFor: "#", TimeoutMs: 10000, DelayMs: 100},
					{Send: "terminal width 0\n", WaitFor: "#", TimeoutMs: 10000, DelayMs: 100},
				},
			},
			{
				Name:  "Show version",
				Scope: "active",
				Steps: []Step{
					{Send: "show version\n", WaitRegex: `[>#]\s*$`, TimeoutMs: 20000, DelayMs: 200},
				},
			},
			{
				Name:  "Enter enable (wait for #)",
				Scope: "active",
				Steps: []Step{
					{Send: "enable\n", WaitFor: "Password:", TimeoutMs: 10000},
					// Operator still types the password interactively, or add a send step.
					{WaitFor: "#", TimeoutMs: 60000},
				},
			},
		},
	}
}

// Path is the default scripts file in app home.
func Path(appHome string) string {
	return filepath.Join(appHome, FileName)
}

// Load reads path, or returns Defaults when missing.
func Load(path string) (File, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Defaults(), nil
		}
		return File{}, err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return Defaults(), nil
	}
	var f File
	if err := yaml.Unmarshal(raw, &f); err != nil {
		return File{}, fmt.Errorf("scripts file corrupt: %w", err)
	}
	if f.Version == 0 {
		f.Version = 1
	}
	for i := range f.Scripts {
		if f.Scripts[i].Scope == "" {
			f.Scripts[i].Scope = "active"
		}
		for j := range f.Scripts[i].Steps {
			f.Scripts[i].Steps[j].Send = unescape(f.Scripts[i].Steps[j].Send)
			f.Scripts[i].Steps[j].WaitFor = unescape(f.Scripts[i].Steps[j].WaitFor)
		}
	}
	return f, nil
}

// Save writes the file, expanding newlines as \n in YAML for readability.
func Save(path string, f File) error {
	f.Version = 1
	out := f
	out.Scripts = make([]Script, len(f.Scripts))
	for i, sc := range f.Scripts {
		if sc.Scope == "active" {
			sc.Scope = ""
		}
		steps := make([]Step, len(sc.Steps))
		for j, st := range sc.Steps {
			st.Send = escape(st.Send)
			st.WaitFor = escape(st.WaitFor)
			steps[j] = st
		}
		sc.Steps = steps
		out.Scripts[i] = sc
	}
	data, err := yaml.Marshal(out)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".scripts-*.yaml")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

// Sender delivers bytes to one or more terminals.
type Sender interface {
	SendActive(text string)
	SendAll(text string)
}

// Expecter watches the active session's output for a match.
// Hosts that cannot expect should still implement it and return a clear error.
type Expecter interface {
	// WaitForPattern blocks until pattern appears in output (literal or regex).
	WaitForPattern(ctx context.Context, pattern string, regex bool, timeout time.Duration) error
}

// Runner is Sender plus optional Expecter.
type Runner interface {
	Sender
}

// Run executes sc using sender. Context cancel stops between steps.
// When a step has wait_for / wait_regex, sender must also implement Expecter.
func Run(ctx context.Context, sc Script, sender Sender) error {
	if sender == nil {
		return fmt.Errorf("scripts: nil sender")
	}
	scopeAll := strings.EqualFold(sc.Scope, "all")
	exp, _ := sender.(Expecter)

	for i, st := range sc.Steps {
		stepN := i + 1
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("script %q stopped at step %d: %w", sc.Name, stepN, err)
		}
		if text := st.Send; text != "" {
			if scopeAll {
				sender.SendAll(text)
			} else {
				sender.SendActive(text)
			}
		}

		waitPat := strings.TrimSpace(st.WaitFor)
		waitRE := strings.TrimSpace(st.WaitRegex)
		if waitPat != "" || waitRE != "" {
			if exp == nil {
				return fmt.Errorf("script %q step %d: wait_for requires an active SSH session that can expect", sc.Name, stepN)
			}
			timeout := time.Duration(st.TimeoutMs) * time.Millisecond
			if timeout <= 0 {
				timeout = time.Duration(DefaultWaitTimeoutMs) * time.Millisecond
			}
			useRE := waitPat == "" && waitRE != ""
			pat := waitPat
			if useRE {
				pat = waitRE
			}
			if err := exp.WaitForPattern(ctx, pat, useRE, timeout); err != nil {
				return fmt.Errorf("script %q step %d: wait for %q: %w", sc.Name, stepN, pat, err)
			}
		}

		if st.DelayMs > 0 {
			select {
			case <-ctx.Done():
				return fmt.Errorf("script %q stopped at step %d: %w", sc.Name, stepN, ctx.Err())
			case <-time.After(time.Duration(st.DelayMs) * time.Millisecond):
			}
		}
	}
	return nil
}

// MatchBuffer reports whether buf contains a literal or regex match for pattern.
func MatchBuffer(buf, pattern string, useRegex bool) (bool, error) {
	if pattern == "" {
		return true, nil
	}
	if !useRegex {
		return strings.Contains(buf, pattern), nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false, fmt.Errorf("bad wait_regex: %w", err)
	}
	return re.MatchString(buf), nil
}

func unescape(s string) string {
	s = strings.ReplaceAll(s, `\n`, "\n")
	s = strings.ReplaceAll(s, `\r`, "\r")
	s = strings.ReplaceAll(s, `\t`, "\t")
	return s
}

func escape(s string) string {
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	return s
}
