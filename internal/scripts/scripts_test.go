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

type memExpect struct {
	memSender
	saw []string
	err error
}

func (m *memExpect) WaitForPattern(_ context.Context, pattern string, regex bool, _ time.Duration) error {
	m.saw = append(m.saw, pattern)
	_ = regex
	return m.err
}

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

func TestRunWaitFor(t *testing.T) {
	m := &memExpect{}
	sc := Script{
		Name: "w",
		Steps: []Step{
			{Send: "show ver\n", WaitFor: "#", TimeoutMs: 1000, DelayMs: 1},
		},
	}
	if err := Run(context.Background(), sc, m); err != nil {
		t.Fatal(err)
	}
	if len(m.saw) != 1 || m.saw[0] != "#" {
		t.Fatalf("saw=%v", m.saw)
	}
	if got := strings.Join(m.active, ""); got != "show ver\n" {
		t.Fatalf("active=%q", got)
	}
}

func TestRunWaitRequiresExpecter(t *testing.T) {
	m := &memSender{}
	err := Run(context.Background(), Script{
		Name:  "w",
		Steps: []Step{{WaitFor: "#"}},
	}, m)
	if err == nil {
		t.Fatal("expected error without Expecter")
	}
}

func TestMatchBuffer(t *testing.T) {
	ok, err := MatchBuffer("Router#", "#", false)
	if err != nil || !ok {
		t.Fatalf("literal: ok=%v err=%v", ok, err)
	}
	ok, err = MatchBuffer("lab-r1>", `[>#]\s*$`, true)
	if err != nil || !ok {
		t.Fatalf("regex: ok=%v err=%v", ok, err)
	}
	ok, err = MatchBuffer("nope", "#", false)
	if err != nil || ok {
		t.Fatalf("miss: ok=%v err=%v", ok, err)
	}
}
