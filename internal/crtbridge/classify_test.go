package crtbridge

import (
	"testing"

	"github.com/scottpeterman/pathfinderssh/internal/auvik"
)

func TestVPNTunnelForModes(t *testing.T) {
	mixed := Settings{Mode: AutoMixed, VPNTunnels: map[string]string{"Acme": "acme-vpn"}}
	mixed.Normalize()
	if got := mixed.VPNTunnelFor("Acme"); got != "acme-vpn" {
		t.Fatalf("mapped mixed: %q", got)
	}
	if got := mixed.VPNTunnelFor("Other"); got != "" {
		t.Fatalf("mixed must not guess folder name: %q", got)
	}
	mixed.VPNDefault = "corp"
	if got := mixed.VPNTunnelFor("Other"); got != "corp" {
		t.Fatalf("mixed default: %q", got)
	}

	forti := Settings{Mode: AutoFortiClient}
	forti.Normalize()
	if got := forti.VPNTunnelFor("Acme"); got != "" {
		t.Fatalf("forticlient must not guess folder as tunnel: %q", got)
	}
	forti.VPNTunnels = map[string]string{"Acme": "acme-ssl"}
	if got := forti.VPNTunnelFor("Acme"); got != "acme-ssl" {
		t.Fatalf("explicit map wins: %q", got)
	}

	auvikOnly := Settings{Mode: AutoAuvik, VPNTunnels: map[string]string{"Acme": "ignored"}}
	auvikOnly.Normalize()
	if got := auvikOnly.VPNTunnelFor("Acme"); got != "" {
		t.Fatalf("auvik mode has no FortiClient: %q", got)
	}
}

func TestVPNTunnelForSessionFolderCover(t *testing.T) {
	s := Settings{Mode: AutoFortiClient, VPNTunnels: map[string]string{
		"Aspire":          "aspire-ssl",
		"Aspire/FortiGate": "aspire-fg",
	}}
	s.Normalize()
	relSwitch := `3_Customers/Aspire/Arista/Arista1.ini`
	if got := s.VPNTunnelForSession(relSwitch, "Aspire"); got != "aspire-ssl" {
		t.Fatalf("everything under Aspire: %q", got)
	}
	relFG := `3_Customers/Aspire/FortiGate/SSH/FG-60F.ini`
	if got := s.VPNTunnelForSession(relFG, "Aspire"); got != "aspire-fg" {
		t.Fatalf("longer folder wins: %q", got)
	}
	if got := s.VPNTunnelForSession(`3_Customers/WSTEL/edge.ini`, "WSTEL"); got != "" {
		t.Fatalf("unmapped stays empty: %q", got)
	}

	mixedWG := Settings{Mode: AutoMixed, VPNTunnels: map[string]string{"Acme": "wireguard:acme"}}
	mixedWG.Normalize()
	if got := mixedWG.VPNTunnelFor("Acme"); got != "wireguard:acme" {
		t.Fatalf("wireguard map: %q", got)
	}
}

func TestAuvikTenantForSessionFolderCover(t *testing.T) {
	s := Settings{Mode: AutoMixed, AuvikTenants: map[string]string{
		"Aspire":     "aspire-msp",
		"Aspire/Lab": "aspire-lab",
	}}
	s.Normalize()
	if got := s.AuvikTenantForSession(`3_Customers/Aspire/core.ini`, "Aspire"); got != "aspire-msp" {
		t.Fatalf("folder: %q", got)
	}
	if got := s.AuvikTenantForSession(`3_Customers/Aspire/Lab/sw.ini`, "Aspire"); got != "aspire-lab" {
		t.Fatalf("nested: %q", got)
	}
	if got := s.AuvikTenantForSession(`3_Customers/WSTEL/edge.ini`, "WSTEL"); got != "" {
		t.Fatalf("unmapped: %q", got)
	}
}

func TestSuggestFolder(t *testing.T) {
	folders := []string{"Aspire", "Marshall", "WSTEL"}
	if got := SuggestFolder("Aspire SSL", folders); got != "Aspire" {
		t.Fatalf("suggest: %q", got)
	}
	if got := SuggestFolder("nope", folders); got != "" {
		t.Fatalf("no match: %q", got)
	}
}

func TestClassifyAutomationModes(t *testing.T) {
	tenants := []auvik.Tenant{{ID: "t1", Name: "nanook"}}
	tm := auvik.TenantMap{
		Mappings: map[string]string{"t1": "Nanook Wireless"},
		Domains:  map[string]string{"t1": "nanook"},
	}
	relAuvik := `3_Customers/Nanook Wireless/core.ini`
	relOther := `3_Customers/Other ISP/edge.ini`

	mixed := Settings{Mode: AutoMixed}
	mixed.Normalize()
	s := classifySession(mixed, "Nanook Wireless", "10.1.1.1", 22, relAuvik, true, tenants, tm, Session{})
	if s.UseAuvik || s.Proxied() {
		t.Fatalf("mixed must not guess CRT folder == Auvik tenant: %+v", s)
	}
	mixed.AuvikTenants = map[string]string{"Nanook Wireless": "nanook"}
	s = classifySession(mixed, "Nanook Wireless", "10.1.1.1", 22, relAuvik, true, tenants, tm, Session{})
	if !s.UseAuvik || s.VPNTunnel != "" || !s.Proxied() || s.Domain != "nanook" {
		t.Fatalf("mixed mapped Auvik: %+v", s)
	}
	s = classifySession(mixed, "nanook", "10.1.1.1", 22, `3_Customers/nanook/core.ini`, true, tenants, auvik.TenantMap{}, Session{})
	if s.UseAuvik {
		t.Fatalf("same name as tenant still needs a map: %+v", s)
	}
	s = classifySession(mixed, "Other ISP", "10.2.2.2", 22, relOther, true, tenants, tm, Session{})
	if s.Proxied() {
		t.Fatalf("mixed must leave unknown customers on standard SSH: %+v", s)
	}

	mixed.VPNTunnels = map[string]string{"Other ISP": "other-vpn"}
	s = classifySession(mixed, "Other ISP", "10.2.2.2", 22, relOther, true, tenants, tm, Session{})
	if s.UseAuvik || s.VPNTunnel != "other-vpn" || !s.Proxied() {
		t.Fatalf("mixed FortiClient map: %+v", s)
	}

	forti := Settings{Mode: AutoFortiClient}
	forti.Normalize()
	s = classifySession(forti, "Nanook Wireless", "10.1.1.1", 22, relAuvik, true, tenants, tm, Session{})
	if s.UseAuvik {
		t.Fatal("forticlient mode must not use Auvik")
	}
	if s.Proxied() || s.VPNTunnel != "" {
		t.Fatalf("forticlient must not guess unmapped folders: %+v", s)
	}
	forti.VPNTunnels = map[string]string{"Marshall": "marshall-ssl"}
	relForti := `3_Customers/Marshall/New VPN/1_FortiGate/SSH/FG-60F.ini`
	s = classifySession(forti, "Marshall", "10.3.3.3", 22, relForti, true, tenants, tm, Session{})
	if s.UseAuvik || s.VPNTunnel != "marshall-ssl" || !s.Proxied() {
		t.Fatalf("mapped folder covers FortiGate: %+v", s)
	}
	relSwitch := `3_Customers/Marshall/Arista/core.ini`
	s = classifySession(forti, "Marshall", "10.3.3.4", 22, relSwitch, true, tenants, tm, Session{})
	if s.VPNTunnel != "marshall-ssl" || !s.Proxied() {
		t.Fatalf("mapped folder covers everything under it: %+v", s)
	}
	relAP := `3_Customers/8thFire/FortiNet/FortiAP/FortiAP1.ini`
	s = classifySession(forti, "8thFire", "10.4.4.4", 22, relAP, true, tenants, tm, Session{})
	if s.Proxied() {
		t.Fatalf("unmapped customer stays direct: %+v", s)
	}

	auvikOnly := Settings{Mode: AutoAuvik, VPNTunnels: map[string]string{"Nanook Wireless": "should-ignore"}, AuvikTenants: map[string]string{"Nanook Wireless": "nanook"}}
	auvikOnly.Normalize()
	s = classifySession(auvikOnly, "Nanook Wireless", "10.1.1.1", 22, relAuvik, true, tenants, tm, Session{})
	if !s.UseAuvik || s.VPNTunnel != "" {
		t.Fatalf("auvik mode ignores FortiClient map: %+v", s)
	}
	s = classifySession(auvikOnly, "Other ISP", "10.2.2.2", 22, relOther, true, tenants, tm, Session{})
	if s.Proxied() {
		t.Fatalf("auvik-only unknown customer stays direct: %+v", s)
	}
}

func TestIsFortiGateSession(t *testing.T) {
	yes := []string{
		`3_Customers/Marshall/New VPN/1_FortiGate/SSH/FG-60F.ini`,
		`3_Customers/WSTEL/FortiGate/SSH/FG1800F.ini`,
		`3_Customers/8thFire/FortiNet/FortiGate/core.ini`,
		`Customers/Acme/fortios/edge.ini`,
		`FGT-100F.ini`,
	}
	no := []string{
		`3_Customers/Aspire/Arista/Arista1.ini`,
		`3_Customers/8thFire/FortiNet/FortiAP/FortiAP1.ini`,
		`3_Customers/Marshall/New VPN/2_Servers/VMs/SSH/FortiAnalyzer.ini`,
		`3_Customers/0_OLD_CUSTOMERS/Duke/Juniper/Firewall/SRX - cfw.ini`,
		`3_Customers/Acme/FortiManager/fmg.ini`,
		`core.ini`,
	}
	for _, rel := range yes {
		if !IsFortiGateSession(rel) {
			t.Fatalf("want FortiGate: %s", rel)
		}
	}
	for _, rel := range no {
		if IsFortiGateSession(rel) {
			t.Fatalf("did not want FortiGate: %s", rel)
		}
	}
}

func TestParseTunnelLines(t *testing.T) {
	m := ParseTunnelLines("# comment\nAcme=acme-vpn\nOther ISP\tother-ssl\n")
	if m["Acme"] != "acme-vpn" || m["Other ISP"] != "other-ssl" {
		t.Fatalf("%v", m)
	}
}

func TestLegacyAuvikModeNormalizes(t *testing.T) {
	s := Session{Mode: modeAuvik, OriginalHost: "10.0.0.1"}
	s.normalize()
	if s.Mode != modeProxy || !s.UseAuvik || !s.Proxied() {
		t.Fatalf("%+v", s)
	}
}
