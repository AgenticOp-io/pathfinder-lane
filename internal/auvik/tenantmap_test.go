package auvik

import (
	"path/filepath"
	"testing"

	"github.com/scottpeterman/pathfinderssh/internal/sessions"
)

func TestTenantMapDomains(t *testing.T) {
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	m := TenantMap{Mappings: map[string]string{}, Domains: map[string]string{}}
	m.Set("t1", "Acme")
	m.SetDomain("t1", "nanook")
	if err := SaveTenantMap(home, m); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadTenantMap(home)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.DomainForTenant("t1") != "nanook" {
		t.Fatalf("domain=%q", loaded.DomainForTenant("t1"))
	}
	n := sessions.Node{AuvikTenantID: "t1", Host: "10.0.0.1", AuvikUseTunnel: true}
	if ResolveTunnelDomain(home, n) != "nanook" {
		t.Fatalf("resolve=%q", ResolveTunnelDomain(home, n))
	}
	local := sessions.Node{Name: "oob", Host: "10.1.10.11"}
	enriched := EnrichTunnelDomain(home, "Customers/Acme/OOB", local, sessions.Tree{})
	if ResolveTunnelDomain(home, enriched) != "nanook" {
		t.Fatalf("customer enrich=%q want nanook", ResolveTunnelDomain(home, enriched))
	}
	if !ShouldUseTunnelFirst(enriched, home) {
		t.Fatal("local session under Auvik customer should tunnel first")
	}
}
