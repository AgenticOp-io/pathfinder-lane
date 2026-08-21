// Package recent records recently opened sessions for the session-tree strip.
package recent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const FileName = "recent.json"
const Max = 20

// Entry is one recently activated session.
type Entry struct {
	Folder string    `json:"folder"`
	Name   string    `json:"name"`
	Host   string    `json:"host,omitempty"`
	At     time.Time `json:"at"`
}

type file struct {
	Version int     `json:"version"`
	Entries []Entry `json:"entries"`
}

func Path(appHome string) string {
	return filepath.Join(appHome, FileName)
}

func Load(path string) ([]Entry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var f file
	if err := json.Unmarshal(raw, &f); err != nil {
		return nil, err
	}
	return f.Entries, nil
}

// Touch puts folder/name at the front, deduped, capped at Max.
func Touch(path, folder, name, host string) ([]Entry, error) {
	cur, err := Load(path)
	if err != nil {
		return nil, err
	}
	next := []Entry{{Folder: folder, Name: name, Host: host, At: time.Now().UTC()}}
	for _, e := range cur {
		if e.Folder == folder && e.Name == name {
			continue
		}
		next = append(next, e)
		if len(next) >= Max {
			break
		}
	}
	f := file{Version: 1, Entries: next}
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return nil, err
	}
	return next, nil
}
