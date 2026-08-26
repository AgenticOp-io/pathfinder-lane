package lanectl

import (
	"strings"
	"testing"

	"github.com/scottpeterman/pathfinderssh/internal/crtbridge"
)

func TestHostBindMapsSession(t *testing.T) {
	cfg := crtbridge.Settings{
		Mode:       crtbridge.AutoFortiClient,
		VPNTunnels: map[string]string{"Acme": "wireguard:acme"},
		HostBinds:  map[string]string{"fw-01": "Acme"},
	}
	h := Host{Alias: "fw-01", Name: "fw-01", Host: "10.1.1.1"}
	h.Folder = cfg.HostFolder(h.Alias, h.Name, h.Host)
	if !Mapped(cfg, h) {
		t.Fatalf("bind should map fw-01 under Acme, folder=%q", h.Folder)
	}
}

func TestVPNForSession(t *testing.T) {
	cfg := crtbridge.Settings{
		Mode:       crtbridge.AutoFortiClient,
		VPNTunnels: map[string]string{"Acme": "wireguard:acme"},
		HostBinds:  map[string]string{"fw-01": "Acme"},
	}
	if got := vpnForSession(cfg, "Customers/Acme", "core", "10.1.1.1"); got != "wireguard:acme" {
		t.Fatalf("folder cover: %q", got)
	}
	if got := vpnForSession(cfg, "", "fw-01", "10.1.1.1"); got != "wireguard:acme" {
		t.Fatalf("host bind: %q", got)
	}
	if got := vpnForSession(cfg, "Unassigned", "other", "10.2.2.2"); got != "" {
		t.Fatalf("unmapped must stay empty, got %q", got)
	}
}

func TestMapped(t *testing.T) {
	cfg := crtbridge.Settings{
		Mode:       crtbridge.AutoFortiClient,
		VPNTunnels: map[string]string{"Acme": "wireguard:acme"},
	}
	h := Host{Folder: "Acme", Name: "core", Host: "10.1.1.1", Rel: "Acme/core.ini"}
	if !Mapped(cfg, h) {
		t.Fatal("expected mapped")
	}
	if Mapped(cfg, Host{Folder: "Other", Name: "x", Host: "10.2.2.2", Rel: "Other/x.ini"}) {
		t.Fatal("other folder must stay unmapped")
	}
}

func TestParseSSHConfigHosts(t *testing.T) {
	got := parseSSHConfigHosts("Host edge\n  HostName 10.9.9.9\n  Port 2222\n  User admin\n")
	if len(got) != 1 || got[0].Host != "10.9.9.9" || got[0].Port != 2222 || got[0].User != "admin" {
		t.Fatalf("%+v", got)
	}
}

func TestPatchPuttyText(t *testing.T) {
	raw := "HostName=10.1.1.1\nPortNumber=22\nProtocol=ssh\n"
	got := patchPuttyText(raw, "10.1.1.1", 22, 5, `lane proxy -folder Acme -host %host -port %port`)
	if !strings.Contains(got, "ProxyMethod=5") || !strings.Contains(got, "lane proxy") {
		t.Fatal(got)
	}
	if !strings.Contains(got, "HostName=10.1.1.1") {
		t.Fatal(got)
	}
}

func TestLaneProxyCommand(t *testing.T) {
	got := LaneProxyCommand(`/opt/lane`, "Acme", "%h", "%p")
	if !strings.Contains(got, "/opt/lane proxy -folder Acme -host %h -port %p") {
		t.Fatal(got)
	}
}
