package invsync

import (
	"testing"

	"github.com/scottpeterman/pathfinderssh/internal/sessions"
)

func TestSyncCustomerTreeCrossSourceMergeByIP(t *testing.T) {
	tr := sessions.Tree{}
	root := sessions.DefaultCustomersRoot
	if _, err := tr.CreateCustomer(root, "Contoso Ltd"); err != nil {
		t.Fatal(err)
	}
	auvikFolder := sessions.JoinPath(root, "Contoso Ltd", "Auvik")
	if err := tr.Add(auvikFolder, sessions.Node{
		Name:              "Core Switch",
		Transport:         sessions.TransportSSH,
		Host:              "10.0.0.1",
		AuvikDeviceID:     "auvik-1",
		IntegrationSource: SourceAuvik,
		ExternalDeviceID:  "auvik-1",
	}); err != nil {
		t.Fatal(err)
	}

	res := SyncCustomerTree(&tr, []Device{{
		ID:   "domotz-99",
		Name: "Core Switch",
		Host: "10.0.0.1",
	}}, Options{
		CustomerName:   "Contoso",
		ImportFolder:   "Domotz",
		IntegrationSrc: SourceDomotz,
	})
	if res.Merged == 0 && res.Updated == 0 {
		t.Fatalf("expected merge by IP, got %s", res.Summary())
	}
	if res.Created != 0 {
		t.Fatalf("should not create duplicate session: %s", res.Summary())
	}
}

func TestSyncCustomerTreeScopedDeviceID(t *testing.T) {
	tr := sessions.Tree{}
	root := sessions.DefaultCustomersRoot
	if _, err := tr.CreateCustomer(root, "Fabrikam"); err != nil {
		t.Fatal(err)
	}
	ninjaFolder := sessions.JoinPath(root, "Fabrikam", "Ninja")
	if err := tr.Add(ninjaFolder, sessions.Node{
		Name:              "Host",
		Transport:         sessions.TransportSSH,
		Host:              "10.1.1.1",
		IntegrationSource: SourceNinja,
		ExternalDeviceID:  "42",
	}); err != nil {
		t.Fatal(err)
	}

	res := SyncCustomerTree(&tr, []Device{{
		ID:   "42",
		Name: "Other Host",
		Host: "10.2.2.2",
	}}, Options{
		CustomerName:   "Fabrikam",
		ImportFolder:   "Domotz",
		IntegrationSrc: SourceDomotz,
	})
	if res.Created != 1 {
		t.Fatalf("scoped ids should not collide: %s", res.Summary())
	}
}
