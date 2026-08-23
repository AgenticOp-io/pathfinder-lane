package synclog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppendReadClear(t *testing.T) {
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	Error(home, "auvik", "acme", "HTTP 403", "tenant denied")
	Info(home, "auvik", "", "sync finished", "created 3")
	text, err := ReadTail(home, 64*1024)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "ERROR") || !strings.Contains(text, "acme") {
		t.Fatalf("text=%q", text)
	}
	if hint := AuvikHint("Auvik HTTP 403: permission"); hint == "" {
		t.Fatal("expected 403 hint")
	}
	if err := Clear(home); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(Path(home)); !os.IsNotExist(err) {
		t.Fatalf("expected missing log, err=%v", err)
	}
}
