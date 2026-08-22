package buttons

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	f := Defaults()
	if err := Save(path, f); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Buttons) != len(f.Buttons) {
		t.Fatalf("len %d", len(got.Buttons))
	}
	if got.Buttons[0].Send != "terminal length 0\n" {
		t.Fatalf("send %q", got.Buttons[0].Send)
	}
}

func TestScriptFieldRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	f := File{
		Version: 1,
		Buttons: []Button{
			{Label: "Run backup", Script: "Daily Backup"},
			{Label: "ping", Send: "ping 8.8.8.8\n"},
		},
	}
	if err := Save(path, f); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Buttons[0].Script != "Daily Backup" {
		t.Fatalf("script %q", got.Buttons[0].Script)
	}
	if got.Buttons[1].Send != "ping 8.8.8.8\n" {
		t.Fatalf("send %q", got.Buttons[1].Send)
	}
}

func TestLoadMissing(t *testing.T) {
	got, err := Load(filepath.Join(t.TempDir(), "nope.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Buttons) == 0 {
		t.Fatal("expected defaults")
	}
	_ = os.ErrNotExist
}
