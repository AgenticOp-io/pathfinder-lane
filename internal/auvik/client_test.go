package auvik

import "testing"

func TestResolveBaseURL(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", defaultBase},
		{"us2", "https://auvikapi.us2.my.auvik.com"},
		{"https://auvikapi.us1.my.auvik.com", "https://auvikapi.us1.my.auvik.com"},
		{"https://auvikapi.us2.my.auvik.com/", "https://auvikapi.us2.my.auvik.com"},
		{"https://auvikapi.eu1.my.auvik.com/v1", "https://auvikapi.eu1.my.auvik.com"},
		{"https://us1.my.auvik.com", "https://auvikapi.us1.my.auvik.com"},
		{"https://us2.my.auvik.com/dashboard", "https://auvikapi.us2.my.auvik.com"},
		{"http://auvikapi.ca1.my.auvik.com", "https://auvikapi.ca1.my.auvik.com"},
		{"https://hyperionsolutionsgroup.us2.my.auvik.com/#msp/root/clients", "https://auvikapi.us2.my.auvik.com"},
		{"https://acme.us3.my.auvik.com/foo?x=1", "https://auvikapi.us3.my.auvik.com"},
	}
	for _, tc := range cases {
		got := ResolveBaseURL(tc.in)
		if got != tc.want {
			t.Errorf("ResolveBaseURL(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestFormatHTTPErrorHTML(t *testing.T) {
	msg := formatHTTPError(404, "https://auvikapi.us1.my.auvik.com/v1/tenants", []byte("<!DOCTYPE html><html><title>Not Found</title></html>"))
	if !containsAll(msg, "HTTP 404", "auvikapi") {
		t.Fatalf("expected helpful 404 message, got %q", msg)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !contains(s, p) {
			return false
		}
	}
	return true
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		(func() bool {
			for i := 0; i+len(sub) <= len(s); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		})())
}
