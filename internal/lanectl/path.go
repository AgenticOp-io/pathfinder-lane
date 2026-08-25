package lanectl

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/crtapp"
)

const (
	pathMarker = "# pathfinder-lane"
	pathExport = `export PATH="$HOME/.local/bin:$PATH"`
)

// UserLocalBin is ~/.local/bin (XDG user executables).
func UserLocalBin() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".local", "bin")
}

// InstallOnPATH puts pflane on the user PATH: Windows user Path (the bin
// directory), Unix ~/.local/bin/pflane plus a shell snippet when needed.
func InstallOnPATH(exe string) (string, error) {
	exe = strings.TrimSpace(exe)
	if exe == "" {
		return "", nil
	}
	abs, err := filepath.Abs(exe)
	if err != nil {
		return "", err
	}
	if isGoRunCache(abs) {
		return "", nil
	}
	if st, err := os.Stat(abs); err != nil || st.IsDir() {
		return "", nil
	}
	return installOnPATH(abs)
}

// PATHStatus is LookPath("pflane") when the current process can find it.
func PATHStatus() string {
	if p, err := lookPathPflane(); err == nil && p != "" {
		return p
	}
	return ""
}

func lookPathPflane() (string, error) {
	return osLookPath(crtapp.ExeName("pflane"))
}

// pathHasDir reports whether dir is already an entry in a PATH string.
func pathHasDir(pathEnv, dir string) bool {
	dir = trimPathEntry(dir)
	if dir == "" {
		return false
	}
	sep := os.PathListSeparator
	for _, p := range strings.Split(pathEnv, string(sep)) {
		if strings.EqualFold(trimPathEntry(p), dir) {
			return true
		}
	}
	return false
}

func trimPathEntry(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	p = filepath.Clean(p)
	return strings.TrimRight(p, `/\`)
}

func pathSnippet() string {
	return "\n" + pathMarker + "\n" + pathExport + "\n"
}

func ensurePathSnippet(rcPath string) error {
	raw, _ := os.ReadFile(rcPath)
	text := string(raw)
	if strings.Contains(text, pathMarker) || strings.Contains(text, `/.local/bin`) {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(rcPath), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(rcPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if text != "" && !strings.HasSuffix(text, "\n") {
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}
	_, err = f.WriteString(pathSnippet())
	return err
}
