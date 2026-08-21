// Package scripts is Pathfinder's session scripting model: named sequences of
// send/delay steps stored as YAML under ~/.pathfinderssh/scripts.yaml.
//
// The Fyne host wires Send into open terminals; this package never imports the
// UI toolkit. Prompt-aware automation stays in netexec — these scripts inject
// keystrokes the same way the button bar does.
package scripts

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const FileName = "scripts.yaml"

// Step is one action in a script.
type Step struct {
	// Send is literal text pushed to the session. Use \n for Return.
	Send string `yaml:"send,omitempty"`
	// DelayMs waits after the send (or alone if Send is empty).
	DelayMs int `yaml:"delay_ms,omitempty"`
}

// Script is a named runnable sequence.
type Script struct {
	Name string `yaml:"name"`
	// Scope is "active" (default) or "all" SSH tabs.
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
					{Send: "terminal length 0\n", DelayMs: 300},
					{Send: "terminal width 0\n", DelayMs: 200},
				},
			},
			{
				Name:  "Show version",
				Scope: "active",
				Steps: []Step{
					{Send: "show version\n", DelayMs: 500},
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

// Run executes sc using sender. Context cancel stops between steps.
func Run(ctx context.Context, sc Script, sender Sender) error {
	if sender == nil {
		return fmt.Errorf("scripts: nil sender")
	}
	scopeAll := strings.EqualFold(sc.Scope, "all")
	for i, st := range sc.Steps {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("script %q stopped at step %d: %w", sc.Name, i+1, err)
		}
		if text := st.Send; text != "" {
			if scopeAll {
				sender.SendAll(text)
			} else {
				sender.SendActive(text)
			}
		}
		if st.DelayMs > 0 {
			select {
			case <-ctx.Done():
				return fmt.Errorf("script %q stopped at step %d: %w", sc.Name, i+1, ctx.Err())
			case <-time.After(time.Duration(st.DelayMs) * time.Millisecond):
			}
		}
	}
	return nil
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
