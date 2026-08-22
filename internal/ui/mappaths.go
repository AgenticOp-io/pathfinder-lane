// Customer-scoped topology map paths (MSP).
//
// Crawl maps live under ~/.pathfinderssh/maps/<Customer>/ — one folder per
// customer, matching the session tree's Customers/<name> model. The JSON
// itself has no customer field; the directory is the association.
package ui

import (
	"os"
	"path/filepath"
	"strings"
)

// MapsRootDir is the parent of every per-customer map folder.
func MapsRootDir(appHome string) string {
	if strings.TrimSpace(appHome) == "" {
		appHome = GetAppHome()
	}
	return filepath.Join(appHome, "maps")
}

// CustomerMapsDir is where crawls for one customer write map JSON.
func CustomerMapsDir(appHome, customer string) string {
	return filepath.Join(MapsRootDir(appHome), SanitizePathSegment(customer))
}

// EnsureCustomerMapsDir creates the customer's maps folder.
func EnsureCustomerMapsDir(appHome, customer string) (string, error) {
	dir := CustomerMapsDir(appHome, customer)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// SanitizePathSegment makes a customer name safe as a single path component.
func SanitizePathSegment(s string) string {
	s = strings.TrimSpace(s)
	repl := strings.NewReplacer(`/`, `_`, `\`, `_`, `:`, `_`, `*`, `_`, `?`, `_`, `"`, `_`, `<`, `_`, `>`, `_`, `|`, `_`)
	return repl.Replace(s)
}

// InferCustomerFromMapsPath returns the customer leaf when path is under
// …/maps/<Customer>/…. Empty when the path is not in that layout.
func InferCustomerFromMapsPath(path string) string {
	path = filepath.Clean(ExpandHome(path))
	parts := strings.Split(filepath.ToSlash(path), "/")
	for i := 0; i+1 < len(parts); i++ {
		if strings.EqualFold(parts[i], "maps") {
			name := strings.TrimSpace(parts[i+1])
			if name == "" || name == "." {
				return ""
			}
			// maps/<customer>/<file.json> — customer is the segment before the file
			// or the only segment under maps when path is a directory.
			if i+2 < len(parts) {
				return name
			}
			return name
		}
	}
	return ""
}

// ListMapCustomers returns subdirectory names under maps root (sorted).
// Missing root yields an empty list, not an error.
func ListMapCustomers(appHome string) []string {
	root := MapsRootDir(appHome)
	ents, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range ents {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	return out
}
