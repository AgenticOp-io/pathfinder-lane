package crtbridge

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/crtimport"
	"github.com/scottpeterman/pathfinderssh/internal/sessions"
)

// SessionsDir is Config/Sessions.
func SessionsDir(crtConfig string) string {
	crtConfig = strings.TrimSpace(crtConfig)
	if crtConfig == "" {
		crtConfig = crtimport.DefaultConfig()
	}
	if crtConfig == "" {
		return ""
	}
	if strings.EqualFold(filepath.Base(crtConfig), "Sessions") {
		return crtConfig
	}
	return filepath.Join(crtConfig, "Sessions")
}

// DetectCustomerRoot picks the SecureCRT top-level folder that looks like the
// customer list (Customers, 3_Customers, or a name containing "customer").
func DetectCustomerRoot(sessionsDir string) string {
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		return ""
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.EqualFold(name, "Default") {
			continue
		}
		names = append(names, name)
	}
	for _, name := range names {
		if strings.EqualFold(name, sessions.DefaultCustomersRoot) ||
			strings.EqualFold(name, sessions.LegacyCRTCustomersRoot) {
			return name
		}
	}
	for _, name := range names {
		if strings.Contains(strings.ToLower(name), "customer") {
			return name
		}
	}
	if len(names) == 1 {
		return names[0]
	}
	return ""
}

// ListCustomerFolders returns immediate child folders under the CRT customer root
// (the folders you map to a FortiClient VPN).
func ListCustomerFolders(sessionsDir, customerRoot string) []string {
	sessionsDir = strings.TrimSpace(sessionsDir)
	customerRoot = strings.TrimSpace(customerRoot)
	if sessionsDir == "" {
		return nil
	}
	root := sessionsDir
	if customerRoot != "" {
		root = filepath.Join(sessionsDir, customerRoot)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if strings.HasPrefix(name, "__") || strings.EqualFold(name, "Default") {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// CustomerOfRel returns the customer name for a session path under Sessions.
func CustomerOfRel(rel, customerRoot string) string {
	rel = relKey(rel)
	parts := strings.Split(rel, "/")
	if len(parts) == 0 {
		return ""
	}
	customerRoot = strings.TrimSpace(customerRoot)
	if customerRoot != "" {
		if !strings.EqualFold(parts[0], customerRoot) {
			return ""
		}
		if len(parts) < 3 {
			// file sitting on the customer root, not under a customer
			return ""
		}
		return parts[1]
	}
	if len(parts) < 2 {
		return ""
	}
	return parts[0]
}

// BackupCustomerFolder copies the main customer folder (or all of Sessions)
// into appHome/crt-backup/<stamp>/ before any rewrite.
func BackupCustomerFolder(sessionsDir, customerRoot, appHome string) (backupDir string, err error) {
	sessionsDir = strings.TrimSpace(sessionsDir)
	if sessionsDir == "" {
		return "", fmt.Errorf("SecureCRT Sessions folder not found")
	}
	src := sessionsDir
	if customerRoot != "" {
		cand := filepath.Join(sessionsDir, customerRoot)
		if st, err := os.Stat(cand); err == nil && st.IsDir() {
			src = cand
		}
	}
	backupDir = filepath.Join(appHome, "crt-backup", nowStamp())
	if err := copyDir(src, filepath.Join(backupDir, filepath.Base(src))); err != nil {
		return "", fmt.Errorf("backup SecureCRT folder: %w", err)
	}
	return backupDir, nil
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}
