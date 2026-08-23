package mspsecurity

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/scottpeterman/pathfinderssh/internal/product"
)

const policyFileName = "msp-security-policy.json"

// Policy is organization security posture applied to engineer workstations.
type Policy struct {
	ReadOnlyMode      bool   `json:"read_only_mode,omitempty"`
	ChangeWindowStart string `json:"change_window_start,omitempty"`
	ChangeWindowEnd   string `json:"change_window_end,omitempty"`
	VaultBreakGlass   bool   `json:"vault_break_glass,omitempty"`
	CaptureByDefault  bool   `json:"capture_by_default,omitempty"`
	UpdatedAt         time.Time `json:"updated_at,omitempty"`
}

func installRoot() string {
	if runtime.GOOS == "windows" {
		if base := os.Getenv("LOCALAPPDATA"); base != "" {
			return filepath.Join(base, product.InstallDir)
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return product.InstallDir
	}
	return filepath.Join(home, ".pathfinderssh-msp-app")
}

// Path returns the staged policy file in the install root.
func Path() string {
	return filepath.Join(installRoot(), policyFileName)
}

// Load reads policy from the install root.
func Load() (Policy, bool, error) {
	return LoadFile(Path())
}

// LoadFile reads policy from an explicit path.
func LoadFile(path string) (Policy, bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Policy{}, false, nil
		}
		return Policy{}, false, err
	}
	var p Policy
	if err := json.Unmarshal(raw, &p); err != nil {
		return Policy{}, false, fmt.Errorf("parse security policy: %w", err)
	}
	return p, true, nil
}

// Save writes policy to the install root.
func Save(p Policy) error {
	p.ChangeWindowStart = strings.TrimSpace(p.ChangeWindowStart)
	p.ChangeWindowEnd = strings.TrimSpace(p.ChangeWindowEnd)
	if p.UpdatedAt.IsZero() {
		p.UpdatedAt = time.Now()
	}
	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
