package itglue

import (
	"fmt"
	"strings"
)

// ImportDialogOptions are UI choices for an IT Glue import.
type ImportDialogOptions struct {
	UpdateVault    bool
	LinkSessions   bool
	OnlyEmptyCreds bool
	SSHFilter      bool
	CustomerName   string
}

// ImportResult summarizes vault + session linking.
type ImportResult struct {
	Vault  ImportStats
	Link   LinkStats
	Errors []string
}

func (r ImportResult) Summary() string {
	parts := []string{r.Vault.Summary()}
	if r.Link.Linked > 0 || r.Link.Skipped > 0 {
		parts = append(parts, r.Link.Summary())
	}
	return strings.Join(parts, "; ")
}

// FilterSSHPasswords drops entries that are unlikely to be device logins.
func FilterSSHPasswords(list []Password) []Password {
	out := make([]Password, 0, len(list))
	for _, p := range list {
		name := strings.ToLower(p.Name)
		cat := strings.ToLower(p.CategoryName)
		if strings.TrimSpace(p.Name) == "" {
			continue
		}
		if strings.Contains(cat, "wifi") || strings.Contains(name, "wifi") {
			continue
		}
		out = append(out, p)
	}
	return out
}

// MergeStats combines errors from sub-steps.
func MergeStats(v ImportStats, l LinkStats, errs []string) ImportResult {
	return ImportResult{Vault: v, Link: l, Errors: errs}
}

// FormatImportError wraps a step failure.
func FormatImportError(step string, err error) string {
	return fmt.Sprintf("%s: %v", step, err)
}
