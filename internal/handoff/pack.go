package handoff

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/scottpeterman/pathfinderssh/internal/evidence"
	"github.com/scottpeterman/pathfinderssh/internal/sessions"
	"github.com/scottpeterman/pathfinderssh/internal/ui"
)

// Options configures a customer handoff export zip.
type Options struct {
	Customer string
	AppHome  string
	Tree     sessions.Tree
}

// Build collects non-secret customer package files for MSP handoff.
func Build(opts Options) ([]evidence.File, error) {
	customer := strings.TrimSpace(opts.Customer)
	if customer == "" {
		return nil, fmt.Errorf("customer name required")
	}
	folder := sessions.JoinPath(sessions.DefaultCustomersRoot, customer)
	sub, err := opts.Tree.Subtree(folder)
	if err != nil {
		return nil, err
	}
	yaml, err := sessions.MarshalTree(sub)
	if err != nil {
		return nil, err
	}
	var out []evidence.File
	out = append(out, evidence.File{Name: "sessions.yaml", Content: yaml})

	meta := inventoryMeta(sub, customer)
	metaJSON, _ := json.MarshalIndent(meta, "", "  ")
	out = append(out, evidence.File{Name: "inventory-meta.json", Content: metaJSON})

	mapDir := ui.CustomerMapsDir(opts.AppHome, customer)
	if ents, err := os.ReadDir(mapDir); err == nil {
		for _, e := range ents {
			if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(mapDir, e.Name()))
			if err != nil {
				continue
			}
			out = append(out, evidence.File{Name: "maps/" + e.Name(), Content: data})
		}
	}

	manifest := fmt.Sprintf("PathfinderSSH MSP customer handoff\nCustomer: %s\nExported: %s\nFiles: %d\n",
		customer, time.Now().Format(time.RFC3339), len(out))
	out = append(out, evidence.File{Name: "manifest.txt", Content: []byte(manifest)})
	return out, nil
}

type inventoryEntry struct {
	Folder   string `json:"folder"`
	Name     string `json:"name"`
	Host     string `json:"host,omitempty"`
	Platform string `json:"platform,omitempty"`
	Transport string `json:"transport"`
}

func inventoryMeta(tr sessions.Tree, customer string) []inventoryEntry {
	var out []inventoryEntry
	tr.WalkSessions(func(folder string, n sessions.Node) {
		out = append(out, inventoryEntry{
			Folder:    folder,
			Name:      n.Label(),
			Host:      n.Host,
			Platform:  n.Vendor,
			Transport: string(n.Transport),
		})
	})
	if len(out) == 0 {
		return []inventoryEntry{{Folder: sessions.JoinPath(sessions.DefaultCustomersRoot, customer), Name: "(empty)"}}
	}
	return out
}

// PackName builds the default zip filename.
func PackName(customer string) string {
	safe := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, customer)
	return fmt.Sprintf("handoff-%s-%s.zip", safe, time.Now().Format("20060102-150405"))
}
