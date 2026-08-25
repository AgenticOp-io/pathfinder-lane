//go:build !windows

package lanectl

import (
	"os"
	"path/filepath"
	"strings"
)

func puttyDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".putty", "sessions")
}

func listPutty() []puttyEntry {
	dir := puttyDir()
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []puttyEntry
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		name := e.Name()
		name = strings.ReplaceAll(name, "%20", " ")
		pe := parsePuttyText(name, string(raw))
		pe.Key = e.Name()
		out = append(out, pe)
	}
	return out
}

func applyPutty(e puttyEntry, host string, port int, proxyMethod uint32, proxyCmd string) error {
	dir := puttyDir()
	if dir == "" {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, e.Key)
	raw, _ := os.ReadFile(path)
	return os.WriteFile(path, []byte(patchPuttyText(string(raw), host, port, proxyMethod, proxyCmd)), 0o644)
}
