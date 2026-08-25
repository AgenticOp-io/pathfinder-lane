package crtbridge

import (
	"os"
	"path/filepath"

	"github.com/scottpeterman/pathfinderssh/internal/crtapp"
)

// MigrateLegacyState copies CRT-bridge state from ~/.pathfinderssh when the
// standalone companion home is empty. Pathfinder settings.json is not copied
// (different schema).
func MigrateLegacyState(destHome string) error {
	src := crtapp.LegacyPathfinderHome()
	if src == "" || destHome == "" {
		return nil
	}
	if filepath.Clean(src) == filepath.Clean(destHome) {
		return nil
	}
	if err := os.MkdirAll(destHome, 0o755); err != nil {
		return err
	}
	for _, name := range []string{stateFileName, "auvik-tenant-map.json"} {
		from := filepath.Join(src, name)
		to := filepath.Join(destHome, name)
		if _, err := os.Stat(to); err == nil {
			continue
		}
		st, err := os.Stat(from)
		if err != nil || st.IsDir() {
			continue
		}
		if err := copyFile(from, to); err != nil {
			return err
		}
	}
	return nil
}
