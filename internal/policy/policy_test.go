package policy

import (
	"testing"
	"time"
)

func TestAllowReadOnly(t *testing.T) {
	p := Policy{ReadOnly: true}
	ok, reason := p.Allow(time.Now())
	if ok || reason == "" {
		t.Fatalf("want blocked, got %v %q", ok, reason)
	}
}

func TestChangeWindowDay(t *testing.T) {
	p := Policy{ChangeWindowStart: "09:00", ChangeWindowEnd: "17:00"}
	noon := time.Date(2026, 1, 1, 12, 0, 0, 0, time.Local)
	ok, _ := p.Allow(noon)
	if !ok {
		t.Fatal("noon should be allowed")
	}
	night := time.Date(2026, 1, 1, 20, 0, 0, 0, time.Local)
	ok, _ = p.Allow(night)
	if ok {
		t.Fatal("20:00 should be blocked")
	}
}

func TestChangeWindowOvernight(t *testing.T) {
	p := Policy{ChangeWindowStart: "22:00", ChangeWindowEnd: "06:00"}
	ok, _ := p.Allow(time.Date(2026, 1, 1, 23, 0, 0, 0, time.Local))
	if !ok {
		t.Fatal("23:00 in overnight window")
	}
	ok, _ = p.Allow(time.Date(2026, 1, 1, 10, 0, 0, 0, time.Local))
	if ok {
		t.Fatal("10:00 outside overnight window")
	}
}
