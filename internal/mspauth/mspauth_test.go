package mspauth

import (
	"encoding/base64"
	"path/filepath"
	"testing"
	"time"

	"github.com/scottpeterman/pathfinderssh/internal/idp"
)

func TestValidateEnrollment(t *testing.T) {
	if err := ValidateEnrollment(Enrollment{Provider: ProviderLocal}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateEnrollment(Enrollment{Provider: ProviderEntra}); err == nil {
		t.Fatal("want err")
	}
	if err := ValidateEnrollment(Enrollment{Provider: ProviderEntra, ClientID: "c", TenantID: "t"}); err != nil {
		t.Fatal(err)
	}
}

func TestEnrollmentRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATHFINDER_MSP_ENROLLMENT", filepath.Join(dir, "msp-enrollment.json"))

	e := Enrollment{Provider: ProviderEntra, ClientID: "client", TenantID: "tenant", Domain: "contoso.com"}
	if err := SaveEnrollment(e); err != nil {
		t.Fatal(err)
	}
	got, ok, err := LoadEnrollment()
	if err != nil || !ok {
		t.Fatalf("load %+v %v", got, err)
	}
	if got.ClientID != "client" {
		t.Fatalf("%+v", got)
	}
}

func TestParseIDTokenClaims(t *testing.T) {
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"user1","email":"u@contoso.com","name":"User"}`))
	raw := "hdr." + payload + ".sig"
	c, err := idp.ParseIDTokenClaims(raw)
	if err != nil || c.Subject != "user1" {
		t.Fatalf("%+v %v", c, err)
	}
}

func TestLoginRequired(t *testing.T) {
	enroll := Enrollment{Provider: ProviderEntra, ClientID: "c", TenantID: "t"}
	if !LoginRequired(enroll, UserSession{}, false) {
		t.Fatal("cloud enroll requires login")
	}
}

func TestSessionValid(t *testing.T) {
	enroll := Enrollment{Provider: ProviderEntra, TenantID: "t", ClientID: "c"}
	now := time.Now()
	sess := UserSession{Provider: ProviderEntra, Subject: "sub", Email: "a@contoso.com", AuthenticatedAt: now}
	if !SessionValid(enroll, sess, now) {
		t.Fatal("expected valid")
	}
}
