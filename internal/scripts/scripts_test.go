package scripts

import (
	"context"
	"strings"
	"testing"
	"time"
)

type memSender struct {
	active []string
	all    []string
}

func (m *memSender) SendActive(text string) { m.active = append(m.active, text) }
func (m *memSender) SendAll(text string)    { m.all = append(m.all, text) }

func TestRunActiveSendsAndDelays(t *testing.T) {
	m := &memSender{}
	sc := Script{
		Name:  "t",
		Scope: "active",
		Steps: []Step{
			{Send: "one\n", DelayMs: 5},
			{Send: "two\n"},
		},
	}
	start := time.Now()
	if err := Run(context.Background(), sc, m); err != nil {
		t.Fatal(err)
	}
	if time.Since(start) < 5*time.Millisecond {
		t.Fatal("expected delay")
	}
	if got := strings.Join(m.active, "|"); got != "one\n|two\n" {
		t.Fatalf("active=%q", got)
	}
	if len(m.all) != 0 {
		t.Fatalf("all=%v", m.all)
	}
}

func TestRunCancel(t *testing.T) {
	m := &memSender{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := Run(ctx, Script{Name: "x", Steps: []Step{{Send: "a\n", DelayMs: 1000}}}, m)
	if err == nil {
		t.Fatal("expected cancel error")
	}
}
