package ui

import "testing"

func TestEmbeddedLogoPresent(t *testing.T) {
	SetLogoPath("") // clear any override
	res := Logo()
	if res == nil {
		t.Fatal("expected embedded AgenticOps logo")
	}
	if len(res.Content()) < 1000 {
		t.Fatalf("logo too small: %d bytes", len(res.Content()))
	}
	icon := AppIcon()
	if icon == nil {
		t.Fatal("expected embedded app icon")
	}
}
