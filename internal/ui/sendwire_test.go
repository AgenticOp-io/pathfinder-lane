package ui

import "testing"

func TestNormalizeTerminalSend(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"terminal length 0\n", "terminal length 0\r"},
		{"show run\r\n", "show run\r"},
		{"enable\r", "enable\r"},
		{"a\nb\n", "a\rb\r"},
		{"noeol", "noeol"},
	}
	for _, tc := range cases {
		if got := NormalizeTerminalSend(tc.in); got != tc.want {
			t.Fatalf("NormalizeTerminalSend(%q)=%q want %q", tc.in, got, tc.want)
		}
	}
}
