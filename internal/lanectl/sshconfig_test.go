package lanectl

import (
	"strings"
	"testing"
)

func TestRenderSSHConfigProxy(t *testing.T) {
	got := RenderSSHConfig(`/opt/pflane`, []SSHHost{{
		Alias:  "acme-core",
		Folder: "Acme",
		Host:   "10.1.1.1",
		Port:   22,
		Via:    "proxy",
	}})
	for _, p := range []string{"Host acme-core", "HostName 10.1.1.1", "ProxyCommand", "HostKeyAlias 10.1.1.1", "folder Acme"} {
		if !strings.Contains(got, p) {
			t.Fatalf("missing %q in %s", p, got)
		}
	}
}

func TestRenderSSHConfigAgent(t *testing.T) {
	got := RenderSSHConfig("pflane", []SSHHost{{
		Alias:     "acme-core",
		Folder:    "Acme",
		Host:      "10.1.1.1",
		FrontPort: 52100,
		Via:       "agent",
	}})
	if !strings.Contains(got, "HostName 127.0.0.1") || !strings.Contains(got, "Port 52100") {
		t.Fatal(got)
	}
}
