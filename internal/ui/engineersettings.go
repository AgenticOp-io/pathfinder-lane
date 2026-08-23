package ui

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/scottpeterman/pathfinderssh/internal/appinstall"
)

const engineerSettingsName = "msp-engineer-settings.json"

// SaveEngineerSettingsSnapshot writes integration settings for engineer packages.
func SaveEngineerSettingsSnapshot() error {
	s, err := LoadSettings(SettingsPath())
	if err != nil {
		s = Defaults()
	}
	envelope := settingsFile{Version: settingsFileVersion, Settings: s.Normalized()}
	raw, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	dest := filepath.Join(appinstall.Root(), engineerSettingsName)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	tmp := dest + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, dest); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// EngineerSettingsSnapshotPath is the admin-side snapshot path in the install root.
func EngineerSettingsSnapshotPath() string {
	return filepath.Join(appinstall.Root(), engineerSettingsName)
}
