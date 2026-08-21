// Package buttons is the Pathfinder button-bar model: label → send string.
//
// Stored as YAML under ~/.pathfinderssh/buttons.yaml so it is editable and
// portable. The Fyne host wires Send into the active terminal; this package
// never imports the UI toolkit.
package buttons

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const FileName = "buttons.yaml"

// Button is one toolbar action.
type Button struct {
	Label string `yaml:"label"`
	// Send is the literal text pushed to the session. Use \n for Return.
	Send string `yaml:"send"`
	// Scope is "active" (default) or "all" SSH tabs.
	Scope string `yaml:"scope,omitempty"`
}

// File is the on-disk envelope.
type File struct {
	Version int      `yaml:"version"`
	Buttons []Button `yaml:"buttons"`
}

// Defaults returns a starter set useful on network gear.
func Defaults() File {
	return File{
		Version: 1,
		Buttons: []Button{
			{Label: "term len 0", Send: "terminal length 0\n"},
			{Label: "show run", Send: "show running-config\n"},
			{Label: "show lldp", Send: "show lldp neighbors\n"},
			{Label: "show cdp", Send: "show cdp neighbors\n"},
			{Label: "enable", Send: "enable\n"},
		},
	}
}

// Path is the default buttons file in app home.
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
		return File{}, fmt.Errorf("buttons file corrupt: %w", err)
	}
	if f.Version == 0 {
		f.Version = 1
	}
	for i := range f.Buttons {
		if f.Buttons[i].Scope == "" {
			f.Buttons[i].Scope = "active"
		}
		f.Buttons[i].Send = unescape(f.Buttons[i].Send)
	}
	return f, nil
}

// Save writes the file, expanding newlines as \n in YAML for readability.
func Save(path string, f File) error {
	f.Version = 1
	out := f
	out.Buttons = make([]Button, len(f.Buttons))
	for i, b := range f.Buttons {
		b.Send = escape(b.Send)
		if b.Scope == "active" {
			b.Scope = ""
		}
		out.Buttons[i] = b
	}
	data, err := yaml.Marshal(out)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".buttons-*.yaml")
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
