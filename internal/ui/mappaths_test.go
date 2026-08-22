package ui

import (
	"path/filepath"
	"testing"
)

func TestCustomerMapsDir(t *testing.T) {
	got := CustomerMapsDir(`/home/u/.pathfinderssh`, `Acme Fiber`)
	want := filepath.Join(`/home/u/.pathfinderssh`, `maps`, `Acme Fiber`)
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestInferCustomerFromMapsPath(t *testing.T) {
	p := filepath.Join(`C:`, `Users`, `david`, `.pathfinderssh`, `maps`, `Acme`, `crawl-2026-08-21.json`)
	if got := InferCustomerFromMapsPath(p); got != `Acme` {
		t.Fatalf("got %q", got)
	}
	if got := InferCustomerFromMapsPath(filepath.Join(`C:`, `temp`, `map.json`)); got != `` {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestSanitizePathSegment(t *testing.T) {
	if got := SanitizePathSegment(`a/b:c`); got != `a_b_c` {
		t.Fatalf("got %q", got)
	}
}
