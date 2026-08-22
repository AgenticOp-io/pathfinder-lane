package workcontext

import (
	"strings"
	"testing"
)

func TestNormalizeIncidentID(t *testing.T) {
	tests := []struct {
		raw, want string
	}{
		{"", ""},
		{"Q0ABCDEF", "Q0ABCDEF"},
		{"https://subdomain.pagerduty.com/incidents/Q0ABCDEF", "Q0ABCDEF"},
		{"https://api.pagerduty.com/incidents/Q0XYZ?foo=1", "Q0XYZ"},
	}
	for _, tc := range tests {
		got := NormalizeIncidentID(tc.raw)
		if got != tc.want {
			t.Errorf("NormalizeIncidentID(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}
}

func TestRecordHostDedupes(t *testing.T) {
	c := Context{}
	c.RecordHost("10.0.0.1")
	c.RecordHost("10.0.0.1")
	c.RecordHost("10.0.0.2")
	if len(c.LinkedHosts) != 2 {
		t.Fatalf("linked hosts = %v, want 2", c.LinkedHosts)
	}
}

func TestBuildSummaryIncludesIncidentAndTabs(t *testing.T) {
	s := BuildSummary(SummaryInput{
		Context: Context{
			IncidentID:   "Q0TEST",
			CustomerName: "Contoso",
			Title:        "Router down",
		},
		OpenTabs: []TabInfo{
			{Title: "edge-r1", Host: "10.1.1.1"},
		},
		EngineerNote: "Replaced uplink cable.",
	})
	if !strings.Contains(s, "Q0TEST") {
		t.Fatalf("summary missing incident: %s", s)
	}
	if !strings.Contains(s, "Contoso") {
		t.Fatalf("summary missing customer: %s", s)
	}
	if !strings.Contains(s, "edge-r1") {
		t.Fatalf("summary missing tab: %s", s)
	}
	if !strings.Contains(s, "Replaced uplink cable") {
		t.Fatalf("summary missing engineer note: %s", s)
	}
}
