//go:build !windows

package lanectl

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func osLookPath(name string) (string, error) {
	return exec.LookPath(name)
}

func installOnPATH(exe string) (string, error) {
	dir := UserLocalBin()
	if dir == "" {
		return "", nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	dest := filepath.Join(dir, "lane")
	if same, _ := samePath(exe, dest); same {
		_ = ensureUnixPathRC()
		return dest, nil
	}
	_ = os.Remove(dest)
	if err := os.Symlink(exe, dest); err != nil {
		if copyErr := copyExe(exe, dest); copyErr != nil {
			return "", err
		}
	}
	if err := ensureUnixPathRC(); err != nil {
		return dest, err
	}
	return dest, nil
}

func samePath(a, b string) (bool, error) {
	if a == b {
		return true, nil
	}
	ra, err := filepath.EvalSymlinks(a)
	if err != nil {
		ra = a
	}
	rb, err := filepath.EvalSymlinks(b)
	if err != nil {
		return false, nil
	}
	return ra == rb, nil
}

func ensureUnixPathRC() error {
	if pathHasDir(os.Getenv("PATH"), UserLocalBin()) {
		return nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	rc := filepath.Join(home, ".profile")
	if runtime.GOOS == "darwin" {
		rc = filepath.Join(home, ".zprofile")
	}
	return ensurePathSnippet(rc)
}
