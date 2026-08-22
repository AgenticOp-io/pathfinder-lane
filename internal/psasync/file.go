package psasync

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileEnvelope is the on-disk PSA customer list.
type FileEnvelope struct {
	Source    string     `json:"source"`
	Customers []Customer `json:"customers"`
}

// FileSource reads customers from a JSON file (see DefaultFileName).
type FileSource struct {
	Path string
}

// DefaultFileName under app home.
const DefaultFileName = "psa-customers.json"

// DefaultPath joins appHome with DefaultFileName.
func DefaultPath(appHome string) string {
	return filepath.Join(appHome, DefaultFileName)
}

func (f FileSource) Name() string {
	base := filepath.Base(f.Path)
	if base == "" || base == "." {
		return "file"
	}
	return "file:" + base
}

func (f FileSource) ListCustomers(ctx context.Context) ([]Customer, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	raw, err := os.ReadFile(f.Path)
	if err != nil {
		return nil, err
	}
	var env FileEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("psa json: %w", err)
	}
	out := make([]Customer, 0, len(env.Customers))
	for _, c := range env.Customers {
		c.Name = strings.TrimSpace(c.Name)
		c.ExternalID = strings.TrimSpace(c.ExternalID)
		if c.Name == "" && c.ExternalID == "" {
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

// WriteExample writes a starter JSON file when missing.
func WriteExample(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	env := FileEnvelope{
		Source: "file",
		Customers: []Customer{
			{ExternalID: "EXAMPLE-1", Name: "Example Customer", Tags: []string{"sample"}},
		},
	}
	data, err := json.MarshalIndent(env, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
