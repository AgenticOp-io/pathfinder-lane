package vpnprov

import (
	"net"
	"strings"
	"testing"
)

func TestMatchVPNInterfaceWireGuard(t *testing.T) {
	ifaces := []IfaceInfo{
		{Name: "Ethernet", IPv4: net.ParseIP("10.0.0.5").To4()},
		{Name: "acme", IPv4: net.ParseIP("10.10.10.2").To4()},
	}
	got, ok := MatchVPNInterface(Target{Kind: WireGuard, Name: "acme"}, ifaces)
	if !ok || got.Name != "acme" {
		t.Fatalf("%v %v", got, ok)
	}
}

func TestMatchVPNInterfaceDoesNotGuessEthernet(t *testing.T) {
	ifaces := []IfaceInfo{
		{Name: "Ethernet", IPv4: net.ParseIP("10.1.1.8").To4()},
		{Name: "Wi-Fi", IPv4: net.ParseIP("192.168.1.10").To4()},
	}
	if _, ok := MatchVPNInterface(Target{Kind: WireGuard, Name: "acme"}, ifaces); ok {
		t.Fatal("must not bind a generic NIC")
	}
}

func TestMatchVPNInterfaceForti(t *testing.T) {
	ifaces := []IfaceInfo{
		{Name: "Fortinet SSL VPN Adapter", IPv4: net.ParseIP("10.200.1.5").To4()},
	}
	got, ok := MatchVPNInterface(Target{Kind: FortiClient, Name: "marshall-ssl"}, ifaces)
	if !ok || !strings.Contains(strings.ToLower(got.Name), "forti") {
		t.Fatalf("%v %v", got, ok)
	}
}
