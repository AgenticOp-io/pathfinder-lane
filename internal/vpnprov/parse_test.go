package vpnprov

import "testing"

func TestParseWGServices(t *testing.T) {
	raw := "SERVICE_NAME: BrokerInfrastructure\r\nSERVICE_NAME: WireGuardTunnel$acme\r\nSERVICE_NAME: WireGuardTunnel$lab\r\n"
	got := ParseWGServices(raw)
	if len(got) != 2 || got[0] != "acme" || got[1] != "lab" {
		t.Fatalf("%v", got)
	}
}

func TestParseWGConfs(t *testing.T) {
	got := ParseWGConfs([]string{"acme.conf.dpapi", "lab.conf", "readme.txt"})
	if len(got) != 2 {
		t.Fatalf("%v", got)
	}
}

func TestParseZscalerName(t *testing.T) {
	svc, p := ParseZscalerName("zpa")
	if svc != "zpa" || p != "" {
		t.Fatalf("%s %s", svc, p)
	}
	svc, p = ParseZscalerName("zpa:partner@tenant")
	if svc != "zpa" || p != "partner@tenant" {
		t.Fatalf("%s %s", svc, p)
	}
	svc, p = ParseZscalerName("just-user")
	if svc != "zpa" || p != "just-user" {
		t.Fatalf("bare partner: %s %s", svc, p)
	}
}

func TestZPAEnabled(t *testing.T) {
	if !zpaEnabled(`{"enabled":true}`) {
		t.Fatal("json true")
	}
	if zpaEnabled(`{"enabled":false,"state":"disabled"}`) {
		t.Fatal("disabled")
	}
	if zpaEnabled(`{"enabled": false}`) {
		t.Fatal("enabled false")
	}
}
