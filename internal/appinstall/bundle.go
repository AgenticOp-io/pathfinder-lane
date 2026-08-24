package appinstall

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// BundledTools are optional CLI/GUI companions copied beside pathfinder.exe.
var BundledTools = []string{
	"pathfinder", "pathfinder-msp", "pfseed", "pfinstall", "pfengineer-install",
	"pfsetup-msp", "pfsetup-o365", "pfsetup-google", "pfsetup-apis",
	"AuvikTunnel",
}

func exeName(base string) string {
	if runtime.GOOS == "windows" {
		return base + ".exe"
	}
	return base
}

// CopyToolBundle copies built tools from srcDir into destDir when present.
func CopyToolBundle(srcDir, destDir string) (bool, error) {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return false, err
	}
	var copied bool
	for _, tool := range BundledTools {
		src := filepath.Join(srcDir, exeName(tool))
		st, err := os.Stat(src)
		if err != nil || st.IsDir() {
			continue
		}
		dst := filepath.Join(destDir, exeName(tool))
		need := true
		if stD, err := os.Stat(dst); err == nil {
			need = st.Size() != stD.Size() || st.ModTime().After(stD.ModTime())
		}
		if !need {
			continue
		}
		if err := copyFile(src, dst); err != nil {
			return copied, fmt.Errorf("install %s: %w", tool, err)
		}
		copied = true
	}
	return copied, nil
}

func isPathfinderExe(path string) bool {
	return strings.EqualFold(filepath.Base(path), exeName("pathfinder"))
}

