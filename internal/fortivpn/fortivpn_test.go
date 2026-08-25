package fortivpn

import "testing"

func TestParseListOfficial(t *testing.T) {
	raw := "SSL:\r\n machine\r\n sslvpn test\r\n\r\nIPSEC:\r\n ipsec test\n"
	got := ParseList(raw)
	if len(got) != 3 || got[0] != "machine" || got[1] != "sslvpn test" || got[2] != "ipsec test" {
		t.Fatalf("%q", got)
	}
	got = ParseList("ssl: Aspire VPN\nipsec: Marshall\n")
	if len(got) != 2 || got[0] != "Aspire VPN" || got[1] != "Marshall" {
		t.Fatalf("one-line: %q", got)
	}
}

func TestParseStatus(t *testing.T) {
	raw := "machine :: Disconnected\r\nsslvpn test :: Connected\r\nipsec :: Disconnected\n"
	got := ParseStatus(raw)
	if len(got) != 3 || got[1].Name != "sslvpn test" || got[1].State != "Connected" {
		t.Fatalf("%+v", got)
	}
}

func TestPlanSwitchTunnels(t *testing.T) {
	lines := []StatusLine{
		{Name: "Acme", State: "Connected"},
		{Name: "Other", State: "Disconnected"},
	}
	up, drop := Plan(lines, "Acme")
	if !up || drop {
		t.Fatalf("already on Acme: up=%v drop=%v", up, drop)
	}
	up, drop = Plan(lines, "Other")
	if up || !drop {
		t.Fatalf("must drop Acme before Other: up=%v drop=%v", up, drop)
	}
	up, drop = Plan([]StatusLine{{Name: "Other", State: "Disconnected"}}, "Other")
	if up || drop {
		t.Fatalf("idle, just connect: up=%v drop=%v", up, drop)
	}
}
