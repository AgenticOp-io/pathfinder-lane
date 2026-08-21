package appinstall

import (
	"path/filepath"
	"runtime"
	"strings"
)

// NormalizePath makes exe paths comparable across \\?\ prefixes, slash style,
// and Windows case. EvalSymlinks is best-effort.
func NormalizePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	p = strings.TrimPrefix(p, `\\?\`)
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	if resolved, err := filepath.EvalSymlinks(p); err == nil {
		p = resolved
		p = strings.TrimPrefix(p, `\\?\`)
	}
	p = filepath.Clean(p)
	if runtime.GOOS == "windows" {
		p = strings.ToLower(p)
	}
	return p
}

// SameFile reports whether two paths name the same executable location.
func SameFile(a, b string) bool {
	na, nb := NormalizePath(a), NormalizePath(b)
	if na == "" || nb == "" {
		return false
	}
	return na == nb
}

// CurrentExe is os.Executable normalized for comparison.
func CurrentExe() (string, error) {
	exe, err := osExecutable()
	if err != nil {
		return "", err
	}
	return NormalizePath(exe), nil
}
