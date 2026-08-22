package crawler

import "testing"

func TestParseMikroTikNeighbors(t *testing.T) {
	in := `
Flags: D - dynamic
 0 identity=sw1 address=10.0.0.2 interface=ether1 platform="Cisco" mac-address=AA:BB:CC:DD:EE:FF

 1 identity=ap1 address=10.0.0.3 interface=ether2 platform=MikroTik
`
	got := parseMikroTikNeighbors(in)
	if len(got) != 2 {
		t.Fatalf("len=%d %#v", len(got), got)
	}
	if got[0].RemoteDevice != "sw1" || got[0].RemoteIP != "10.0.0.2" {
		t.Fatalf("%+v", got[0])
	}
}

func TestGuestNeighbor(t *testing.T) {
	n := guestNeighbor("web1", "10.1.2.3", "linux", "qemu:100")
	if n.Protocol != "guest" || n.RemoteIP != "10.1.2.3" {
		t.Fatalf("%+v", n)
	}
}
