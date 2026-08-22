package mspauth

import (
	"context"
	"fmt"
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/idp"
)

// ParseSetupMode maps install/first-run flags to a provider.
// The bool is true when mode was recognized (non-empty).
func ParseSetupMode(mode string) (idp.Provider, bool) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "prompt":
		return idp.ProviderLocal, false
	case "solo", "local", "none", "justme":
		return idp.ProviderLocal, true
	case "o365", "microsoft", "entra", "m365", "office365":
		return idp.ProviderEntra, true
	case "google", "workspace", "googleworkspace":
		return idp.ProviderGoogle, true
	default:
		return idp.ProviderLocal, false
	}
}

// HeadlessSetup reports whether mode can finish without opening the GUI wizard.
func HeadlessSetup(mode string) bool {
	p, ok := ParseSetupMode(mode)
	return ok && p == idp.ProviderLocal
}

// SaveSoloSetup writes local enrollment and a local session (no browser).
func SaveSoloSetup(home string) error {
	e := Enrollment{Provider: ProviderLocal}
	if err := ValidateEnrollment(e); err != nil {
		return err
	}
	if err := SaveEnrollment(e); err != nil {
		return fmt.Errorf("save enrollment: %w", err)
	}
	auth := NewAuthenticator(home)
	_, err := auth.SignIn(context.Background(), e)
	if err != nil {
		return fmt.Errorf("local session: %w", err)
	}
	return nil
}

// ApplySetup runs a -setup flag. Returns provider preset for cloud modes.
func ApplySetup(mode, home string) (idp.Provider, bool, error) {
	p, ok := ParseSetupMode(mode)
	if !ok {
		return idp.ProviderLocal, false, fmt.Errorf("unknown setup mode %q (use solo, o365, or google)", mode)
	}
	if p == ProviderLocal {
		if err := SaveSoloSetup(home); err != nil {
			return p, true, err
		}
		return p, true, nil
	}
	return p, true, nil
}
