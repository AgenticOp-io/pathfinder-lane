package vpnprov

import "testing"

func TestParseTarget(t *testing.T) {
	cases := []struct {
		in   string
		kind string
		name string
	}{
		{"aspire-ssl", FortiClient, "aspire-ssl"},
		{"forticlient:aspire-ssl", FortiClient, "aspire-ssl"},
		{"forti:HQ", FortiClient, "HQ"},
		{"wireguard:acme", WireGuard, "acme"},
		{"wg:lab", WireGuard, "lab"},
		{"zscaler:zpa", Zscaler, "zpa"},
		{"zscaler:zpa:partner@x", Zscaler, "zpa:partner@x"},
		{"zpa", Zscaler, "zpa"},
		{"zpa:user@tenant", Zscaler, "zpa:user@tenant"},
		{"", "", ""},
	}
	for _, c := range cases {
		got := ParseTarget(c.in)
		if got.Kind != c.kind || got.Name != c.name {
			t.Fatalf("%q: got %+v want %s/%s", c.in, got, c.kind, c.name)
		}
	}
}

func TestFormatTargetRoundTrip(t *testing.T) {
	if FormatTarget(Target{Kind: WireGuard, Name: "acme"}) != "wireguard:acme" {
		t.Fatal(FormatTarget(Target{Kind: WireGuard, Name: "acme"}))
	}
	if FormatTarget(Target{Kind: FortiClient, Name: "HQ"}) != "HQ" {
		t.Fatal("legacy Forti stays bare")
	}
	if ParseTarget(FormatTarget(Target{Kind: Zscaler, Name: "zpa:u"})).Name != "zpa:u" {
		t.Fatal("zscaler partner")
	}
}
