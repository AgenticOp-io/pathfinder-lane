package cursorapi

import "testing"

func TestResolveKey(t *testing.T) {
	t.Setenv("CURSOR_API_KEY", "from-env")
	if got := ResolveKey(""); got != "from-env" {
		t.Fatalf("env fallback: %q", got)
	}
	if got := ResolveKey(" explicit "); got != "explicit" {
		t.Fatalf("explicit wins: %q", got)
	}
}
