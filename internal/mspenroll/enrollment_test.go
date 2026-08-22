package mspenroll

import (
	"path/filepath"
	"testing"

	"github.com/scottpeterman/pathfinderssh/internal/idp"
)

func TestValidateEnrollment(t *testing.T) {
	if err := Validate(Enrollment{Provider: idp.ProviderLocal}); err != nil {
		t.Fatal(err)
	}
	if err := Validate(Enrollment{Provider: idp.ProviderEntra}); err == nil {
		t.Fatal("want err")
	}
}

func TestEnrollmentRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATHFINDER_MSP_ENROLLMENT", filepath.Join(dir, "msp-enrollment.json"))

	e := Enrollment{Provider: idp.ProviderEntra, ClientID: "client", TenantID: "tenant", Domain: "contoso.com"}
	if err := Save(e); err != nil {
		t.Fatal(err)
	}
	got, ok, err := Load()
	if err != nil || !ok {
		t.Fatalf("load %+v %v", got, err)
	}
	if got.ClientID != "client" {
		t.Fatalf("%+v", got)
	}
}
