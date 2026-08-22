package mspauth

import (
	"path/filepath"
	"testing"

	"github.com/scottpeterman/pathfinderssh/internal/idp"
)

func TestParseSetupMode(t *testing.T) {
	p, ok := ParseSetupMode("solo")
	if !ok || p != ProviderLocal {
		t.Fatal()
	}
	p, ok = ParseSetupMode("o365")
	if !ok || p != ProviderEntra {
		t.Fatal()
	}
	_, ok = ParseSetupMode("")
	if ok {
		t.Fatal()
	}
}

func TestHeadlessSetup(t *testing.T) {
	if !HeadlessSetup("solo") {
		t.Fatal()
	}
	if HeadlessSetup("o365") {
		t.Fatal()
	}
}

func TestSaveSoloSetup(t *testing.T) {
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	t.Setenv("PATHFINDER_MSP_ENROLLMENT", filepath.Join(dir, "msp-enrollment.json"))

	if err := SaveSoloSetup(home); err != nil {
		t.Fatal(err)
	}
	e, found, err := LoadEnrollment()
	if err != nil || !found || e.Provider != idp.ProviderLocal {
		t.Fatalf("%+v %v %v", e, found, err)
	}
	sess, ok, err := LoadUserSession(home)
	if err != nil || !ok || sess.Provider != ProviderLocal {
		t.Fatalf("%+v %v %v", sess, ok, err)
	}
}
