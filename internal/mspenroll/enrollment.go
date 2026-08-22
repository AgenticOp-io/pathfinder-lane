package mspenroll

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/scottpeterman/pathfinderssh/internal/appinstall"
	"github.com/scottpeterman/pathfinderssh/internal/idp"
)

const enrollmentFileName = "msp-enrollment.json"

// Enrollment is org-level MSP identity configuration (super admin enrolls once).
type Enrollment struct {
	Provider idp.Provider `json:"provider"`
	TenantID string       `json:"tenant_id,omitempty"`
	ClientID string       `json:"client_id,omitempty"`
	Domain   string       `json:"domain,omitempty"`
	Authority string      `json:"authority,omitempty"`
	EnrolledBy string     `json:"enrolled_by,omitempty"`
	EnrolledAt time.Time  `json:"enrolled_at,omitempty"`
	AllowLocalFallback bool `json:"allow_local_fallback,omitempty"`
}

// LoginConfig converts enrollment to an idp login configuration.
func (e Enrollment) LoginConfig() idp.LoginConfig {
	return idp.LoginConfigFromEnrollment(e.Provider, e.TenantID, e.ClientID, e.Domain, e.Authority)
}

// Path returns the enrollment file path for this install.
func Path() string {
	if p := strings.TrimSpace(os.Getenv("PATHFINDER_MSP_ENROLLMENT")); p != "" {
		return p
	}
	if base := os.Getenv("PROGRAMDATA"); base != "" {
		p := filepath.Join(base, "PathfinderSSH-MSP", enrollmentFileName)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return filepath.Join(appinstall.Root(), enrollmentFileName)
}

// Load reads MSP enrollment. Missing file means not enrolled.
func Load() (Enrollment, bool, error) {
	path := Path()
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Enrollment{}, false, nil
		}
		return Enrollment{}, false, fmt.Errorf("read enrollment: %w", err)
	}
	var e Enrollment
	if err := json.Unmarshal(raw, &e); err != nil {
		return Enrollment{}, false, fmt.Errorf("parse enrollment: %w", err)
	}
	e.Provider = e.Provider.Normalize()
	return e, true, nil
}

// Save writes enrollment atomically.
func Save(e Enrollment) error {
	e.Provider = e.Provider.Normalize()
	if e.EnrolledAt.IsZero() {
		e.EnrolledAt = time.Now()
	}
	path := Path()
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	raw, err := json.MarshalIndent(e, "", "  ")
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

// Validate checks super-admin enrollment before save.
func Validate(e Enrollment) error {
	return idp.ValidateLoginConfig(e.LoginConfig())
}
