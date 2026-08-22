// Package mspauth orchestrates MSP enrollment (mspenroll), OIDC (idp), and sessions.
package mspauth

import (
	"github.com/scottpeterman/pathfinderssh/internal/idp"
	"github.com/scottpeterman/pathfinderssh/internal/mspenroll"
)

// Provider re-exports idp.Provider for callers that already import mspauth.
type Provider = idp.Provider

const (
	ProviderLocal  = idp.ProviderLocal
	ProviderEntra  = idp.ProviderEntra
	ProviderGoogle = idp.ProviderGoogle
)

// Enrollment re-exports org enrollment record.
type Enrollment = mspenroll.Enrollment

// LoadEnrollment reads MSP enrollment configuration.
func LoadEnrollment() (Enrollment, bool, error) { return mspenroll.Load() }

// SaveEnrollment writes MSP enrollment configuration.
func SaveEnrollment(e Enrollment) error { return mspenroll.Save(e) }

// ValidateEnrollment validates enrollment fields.
func ValidateEnrollment(e Enrollment) error { return mspenroll.Validate(e) }

// EnrollmentPath returns the enrollment file path.
func EnrollmentPath() string { return mspenroll.Path() }
