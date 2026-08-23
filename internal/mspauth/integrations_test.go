package mspauth

import (
	"testing"

	"github.com/scottpeterman/pathfinderssh/internal/idp"
)

func TestIntegrationsEnabledSolo(t *testing.T) {
	if !IntegrationsEnabled(Enrollment{Provider: idp.ProviderLocal}) {
		t.Fatal("solo should enable MSP integrations")
	}
}

func TestIntegrationsEnabledCloud(t *testing.T) {
	if !IntegrationsEnabled(Enrollment{Provider: idp.ProviderEntra}) {
		t.Fatal("cloud should enable MSP integrations")
	}
}
