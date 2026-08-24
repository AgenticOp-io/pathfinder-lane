package pyscripts

import (
	"io"
	"os"
	"path/filepath"
	"strings"
)

const dirName = "python-scripts"

// Dir is the catalog folder for saved Python session scripts.
func Dir(appHome string) string {
	return filepath.Join(appHome, dirName)
}

// List returns .py script paths in the catalog directory.
func List(appHome string) ([]string, error) {
	root := Dir(appHome)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	ents, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".py") {
			continue
		}
		out = append(out, filepath.Join(root, e.Name()))
	}
	return out, nil
}

// ImportCopy saves a script into the catalog under its basename.
func ImportCopy(appHome, srcPath string) (string, error) {
	srcPath = strings.TrimSpace(srcPath)
	if srcPath == "" {
		return "", os.ErrInvalid
	}
	base := filepath.Base(srcPath)
	if !strings.HasSuffix(strings.ToLower(base), ".py") {
		base += ".py"
	}
	dst := filepath.Join(Dir(appHome), base)
	in, err := os.Open(srcPath)
	if err != nil {
		return "", err
	}
	defer in.Close()
	if err := os.MkdirAll(Dir(appHome), 0o755); err != nil {
		return "", err
	}
	out, err := os.Create(dst)
	if err != nil {
		return "", err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return "", err
	}
	return dst, nil
}

// Delete removes a catalog script by basename.
func Delete(appHome, basename string) error {
	basename = strings.TrimSpace(basename)
	if basename == "" {
		return os.ErrInvalid
	}
	return os.Remove(filepath.Join(Dir(appHome), basename))
}
