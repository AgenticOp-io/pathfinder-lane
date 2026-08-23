package auvik

import (
	"testing"

	"github.com/scottpeterman/pathfinderssh/internal/product"
	"github.com/scottpeterman/pathfinderssh/internal/sessions"
)

func TestSyncTenantTreeCreatesAndUpdatesIP(t *testing.T) {
	tr := sessions.Tree{}
	tenant := Tenant{ID: "t1", Name: "Acme"}
	devs := []Device{{
		ID: "d1", Name: "core-sw", IPs: []string{"10.0.0.1"},
		DeviceType: "switch", LoginStatus: "authorized", TenantID: "t1",
	}}
	opts := SyncOptions{
		ImportOptions:     ImportOptions{NetworkGearOnly: true, RequireLoginAuthorized: true},
		Tenant:            tenant,
		MoveToAuvikFolder: true,
	}
	res := SyncTenantTree(&tr, devs, opts)
	if res.Created != 1 || !res.Changed() {
		t.Fatalf("first sync %+v", res)
	}

	folder := sessions.JoinPath(product.CustomersRoot, "Acme", ImportFolder)
	f, err := tr.FolderAt(folder)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Sessions) != 1 || f.Sessions[0].Host != "10.0.0.1" {
		t.Fatalf("sessions %+v", f.Sessions)
	}

	// Existing session elsewhere with old IP — merge and update authority IP.
	legacy := sessions.JoinPath(product.CustomersRoot, "Acme", "Legacy")
	if _, err := tr.EnsurePath(legacy); err != nil {
		t.Fatal(err)
	}
	old := sessions.Node{
		Name:       "edge-rtr",
		Transport:  sessions.TransportSSH,
		Host:       "192.168.1.50",
		Username:   "admin",
		AuthType:   sessions.AuthPassword,
		Credential: "cust-admin",
	}
	if err := tr.Add(legacy, old); err != nil {
		t.Fatal(err)
	}

	devs2 := []Device{{
		ID: "d2", Name: "edge-rtr", IPs: []string{"10.0.0.9"},
		DeviceType: "router", LoginStatus: "authorized", TenantID: "t1",
	}}
	res2 := SyncTenantTree(&tr, devs2, opts)
	if res2.Merged == 0 || res2.Updated == 0 {
		t.Fatalf("merge sync %+v", res2)
	}

	n, err := sessionAt(&tr, folder, "edge-rtr")
	if err != nil {
		t.Fatal(err)
	}
	if n.Host != "10.0.0.9" {
		t.Fatalf("host=%q want authority IP", n.Host)
	}
	if n.Username != "admin" || n.Credential != "cust-admin" {
		t.Fatalf("credentials overwritten: %+v", n)
	}
	if n.AuvikDeviceID != "d2" {
		t.Fatalf("auvik id %+v", n)
	}
}

func TestShouldTryTunnel(t *testing.T) {
	n := sessions.Node{AuvikDomain: "nanook", Host: "10.0.0.1", AuvikUseTunnel: true}
	if !ShouldTryTunnel(n, errReach(), true, "") {
		t.Fatal("expected tunnel try")
	}
	if ShouldTryTunnel(n, nil, true, "") {
		t.Fatal("nil err")
	}
	n.AuvikDomain = ""
	if ShouldTryTunnel(n, errReach(), true, "") {
		t.Fatal("no domain")
	}
}

func TestShouldUseTunnelFirst(t *testing.T) {
	n := sessions.Node{AuvikDomain: "acme", Host: "10.0.0.1", IntegrationSource: "auvik"}
	if !ShouldUseTunnelFirst(n, "") {
		t.Fatal("want tunnel first for auvik session")
	}
	// Domain alone (e.g. inherited from customer) is enough — no Auvik inventory row required.
	n = sessions.Node{AuvikDomain: "acme", Host: "10.1.10.11"}
	if !ShouldUseTunnelFirst(n, "") {
		t.Fatal("want tunnel first when domain is set on local session")
	}
	n.AuvikDomain = ""
	if ShouldUseTunnelFirst(n, "") {
		t.Fatal("no domain")
	}
}

type reachErr string

func (e reachErr) Error() string { return string(e) }

func errReach() error { return reachErr("10.0.0.1:22 is not reachable: dial tcp: i/o timeout") }
