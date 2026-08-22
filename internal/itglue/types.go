package itglue

import (
	"fmt"
	"strings"
)

// Organization is an IT Glue client (maps to Customers/<name>/).
type Organization struct {
	ID   string
	Name string
}

// Password is a credential record. Password plaintext is only set after GetPassword.
type Password struct {
	ID               string
	Name             string
	Username         string
	Password         string
	OrganizationID   string
	OrganizationName string
	URL              string
	CategoryName     string
	ResourceURL      string
}

// VaultCredentialName builds a stable vault entry name for a password.
func VaultCredentialName(p Password) string {
	org := strings.TrimSpace(p.OrganizationName)
	name := strings.TrimSpace(p.Name)
	if org == "" {
		org = "org"
	}
	if name == "" {
		name = "password"
	}
	return sanitizeVaultName("ITG " + org + " / " + name)
}

// ITGlueTag returns the vault tag used to track sync source.
func ITGlueTag(id string) string {
	return "itglue-id:" + strings.TrimSpace(id)
}

func sanitizeVaultName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "ITG credential"
	}
	const max = 72
	if len(s) > max {
		s = s[:max]
	}
	return s
}

// ImportStats summarizes a vault import pass.
type ImportStats struct {
	Added    int
	Updated  int
	Skipped  int
	Failed   int
	Errors   []string
}

func (s ImportStats) Summary() string {
	return fmt.Sprintf("vault: added %d, updated %d, skipped %d, failed %d",
		s.Added, s.Updated, s.Skipped, s.Failed)
}

// LinkStats summarizes session ↔ credential linking.
type LinkStats struct {
	Linked int
	Skipped int
}

func (s LinkStats) Summary() string {
	return fmt.Sprintf("sessions linked %d, skipped %d", s.Linked, s.Skipped)
}
