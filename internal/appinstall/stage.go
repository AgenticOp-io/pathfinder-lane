package appinstall

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// StageOrgBundleFrom copies MSP enrollment, branding, and logo from a distribution
// folder (engineer installer package) into the install root.
func StageOrgBundleFrom(srcDir string) error {
	srcDir = strings.TrimSpace(srcDir)
	if srcDir == "" {
		return nil
	}
	root := Root()
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	for _, name := range []string{
		"msp-enrollment.json",
		"msp-branding.json",
		"msp-security-policy.json",
		"msp-engineer-settings.json",
		"logo.png",
	} {
		src := filepath.Join(srcDir, name)
		dst := filepath.Join(root, name)
		if err := copyIfPresent(src, dst); err != nil {
			return fmt.Errorf("stage %s: %w", name, err)
		}
	}
	return nil
}

func copyIfPresent(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer in.Close()
	tmp := dst + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	_ = os.Remove(dst)
	return os.Rename(tmp, dst)
}
