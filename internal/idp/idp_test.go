package idp

import "testing"

func TestProviderNormalize(t *testing.T) {
	if Provider("ENTRA").Normalize() != ProviderEntra {
		t.Fatal()
	}
	if Provider("").Normalize() != ProviderLocal {
		t.Fatal()
	}
}

func TestValidateLoginConfig(t *testing.T) {
	if err := ValidateLoginConfig(LoginConfig{Provider: ProviderLocal}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateLoginConfig(LoginConfig{Provider: ProviderEntra}); err == nil {
		t.Fatal("want err")
	}
	if err := ValidateLoginConfig(LoginConfig{Provider: ProviderEntra, ClientID: "c", TenantID: "t"}); err != nil {
		t.Fatal(err)
	}
}

func TestParseIDTokenClaims(t *testing.T) {
	payload := "eyJzdWIiOiJ1c2VyMSIsImVtYWlsIjoidUBjb250b3NvLmNvbSJ9"
	raw := "hdr." + payload + ".sig"
	c, err := ParseIDTokenClaims(raw)
	if err != nil || c.Subject != "user1" || c.Email != "u@contoso.com" {
		t.Fatalf("%+v %v", c, err)
	}
}
