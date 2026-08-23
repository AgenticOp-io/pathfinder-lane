package pyscripts

import (
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
