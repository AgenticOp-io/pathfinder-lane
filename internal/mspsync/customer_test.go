package mspsync

import "testing"

func TestResolveCustomerName(t *testing.T) {
	existing := []string{"Contoso Ltd", "Northwind Traders", "Fabrikam Inc"}
	if got := ResolveCustomerName(existing, "Contoso Ltd"); got != "Contoso Ltd" {
		t.Fatalf("exact: got %q", got)
	}
	if got := ResolveCustomerName(existing, "contoso ltd"); got != "Contoso Ltd" {
		t.Fatalf("normalized: got %q", got)
	}
	if got := ResolveCustomerName(existing, "Contoso"); got != "Contoso Ltd" {
		t.Fatalf("prefix: got %q", got)
	}
	if got := ResolveCustomerName(existing, "Brand New MSP"); got != "Brand New MSP" {
		t.Fatalf("new: got %q", got)
	}
}

func TestMatchScore(t *testing.T) {
	if MatchScore("Fabrikam Inc", "Fabrikam") < 80 {
		t.Fatalf("expected high score for Fabrikam/Fabrikam Inc")
	}
	if MatchScore("Acme", "Totally Different") != 0 {
		t.Fatalf("expected zero for unrelated names")
	}
}
