package ui

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/scottpeterman/pathfinderssh/internal/mspsecurity"
)

// ApplyStagedMSPConfig merges engineer-package files into live settings and install root.
func ApplyStagedMSPConfig(bundleDir string) error {
	bundleDir = filepath.Clean(bundleDir)
	if err := applyStagedSecurity(bundleDir); err != nil {
		return err
	}
	return applyStagedEngineerSettings(bundleDir)
}

func applyStagedSecurity(bundleDir string) error {
	src := filepath.Join(bundleDir, "msp-security-policy.json")
	p, ok, err := mspsecurity.LoadFile(src)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	dest := mspsecurity.Path()
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	// Copy staged policy into install root for consistency.
	if err := mspsecurity.Save(p); err != nil {
		return err
	}
	base, err := LoadSettings(SettingsPath())
	if err != nil {
		base = Defaults()
	}
	merged := ApplyPolicyToSettings(base, p)
	SetSettings(merged)
	return SaveSettings(SettingsPath(), merged)
}

func applyStagedEngineerSettings(bundleDir string) error {
	src := filepath.Join(bundleDir, "msp-engineer-settings.json")
	raw, err := os.ReadFile(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var envelope settingsFile
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return err
	}
	base, err := LoadSettings(SettingsPath())
	if err != nil {
		base = Defaults()
	}
	merged := mergeEngineerSettings(base, envelope.Settings)
	SetSettings(merged)
	return SaveSettings(SettingsPath(), merged)
}
