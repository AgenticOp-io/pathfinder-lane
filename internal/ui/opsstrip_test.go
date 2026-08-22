package ui

import "testing"

func TestParseQuickConnect(t *testing.T) {
	cases := []struct {
		in             string
		host, user     string
		port           int
	}{
		{"router.lab", "router.lab", "", 0},
		{"admin@router.lab", "router.lab", "admin", 0},
		{"admin@router.lab:2222", "router.lab", "admin", 2222},
		{"10.0.0.1:22", "10.0.0.1", "", 22},
	}
	for _, c := range cases {
		h, u, p := ParseQuickConnect(c.in)
		if h != c.host || u != c.user || p != c.port {
			t.Fatalf("%q → %q %q %d; want %q %q %d", c.in, h, u, p, c.host, c.user, c.port)
		}
	}
}
