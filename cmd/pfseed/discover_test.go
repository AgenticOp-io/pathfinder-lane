package main

import (
	"reflect"
	"testing"
)

func TestParseSSHConfig(t *testing.T) {
	text := `
Host *
    StrictHostKeyChecking ask

Host core-sw1 core-sw1.lab
    HostName 10.0.0.1
    User admin
    Port 22

Host github.com
    User git

Host jump
    HostName jump.lab.example
    Port 2222
`
	got := parseSSHConfig(text)
	want := []Candidate{
		{Name: "core-sw1", Host: "10.0.0.1", Port: 22, User: "admin"},
		{Name: "core-sw1.lab", Host: "10.0.0.1", Port: 22, User: "admin"},
		{Name: "github.com", Host: "github.com", Port: 22, User: "git"},
		{Name: "jump", Host: "jump.lab.example", Port: 2222},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %#v\nwant %#v", got, want)
	}
}

func TestSkipHost(t *testing.T) {
	if !skipHost("github.com") || !skipHost("ssh.dev.azure.com") || skipHost("core-sw1") {
		t.Fatal("skipHost mismatch")
	}
}

func TestSplitHostPort(t *testing.T) {
	h, p, ok := splitHostPort("10.0.0.1:2222")
	if !ok || h != "10.0.0.1" || p != 2222 {
		t.Fatalf("got %q %d %v", h, p, ok)
	}
	if _, _, ok := splitHostPort("10.0.0.1"); ok {
		t.Fatal("bare host should not split")
	}
}
