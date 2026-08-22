package appinstall

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed legal/LICENSE legal/NOTICE
var legalFS embed.FS

// InstallLegalDocs writes LICENSE and NOTICE next to the installed binary so
// a LocalAppData install carries GPLv3 text and upstream attribution without
// requiring the source tree beside the exe.
func InstallLegalDocs(dir string) error {
	if dir == "" {
		dir = BinDir()
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	for _, name := range []string{"LICENSE", "NOTICE"} {
		data, err := legalFS.ReadFile("legal/" + name)
		if err != nil {
			return fmt.Errorf("embed %s: %w", name, err)
		}
		dst := filepath.Join(dir, name)
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", dst, err)
		}
	}
	return nil
}
