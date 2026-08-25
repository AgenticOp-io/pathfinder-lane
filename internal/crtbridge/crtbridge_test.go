package crtbridge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/scottpeterman/pathfinderssh/internal/auvik"
)

func TestPatchSSHHostPortRoundTrip(t *testing.T) {
	raw := []byte("S:\"Session Name\"=Core\r\nS:\"Hostname\"=10.1.2.3\r\nD:\"[SSH2] Port\"=00000016\r\nS:\"Username\"=admin\r\n")
	host, port, ok := ReadSSHHostPort(raw)
	if !ok || host != "10.1.2.3" || port != 22 {
		t.Fatalf("read host=%q port=%d ok=%v", host, port, ok)
	}
	patched := PatchSSHHostPort(raw, "127.0.0.1", 52100)
	h2, p2, ok := ReadSSHHostPort(patched)
	if !ok || h2 != "127.0.0.1" || p2 != 52100 {
		t.Fatalf("patched host=%q port=%d", h2, p2)
	}
	if !strings.Contains(string(patched), `S:"Username"=admin`) {
		t.Fatalf("lost username line: %s", patched)
	}
}

func TestCustomerOfRel(t *testing.T) {
	got := CustomerOfRel(`3_Customers/Acme/Core/rtr.ini`, "3_Customers")
	if got != "Acme" {
		t.Fatalf("got %q", got)
	}
	if CustomerOfRel(`Unassigned/lab.ini`, "3_Customers") != "" {
		t.Fatal("unassigned should not map to a customer")
	}
	if CustomerOfRel(`Acme/rtr.ini`, "") != "Acme" {
		t.Fatal("whole Sessions tree uses first folder as customer")
	}
}

func TestResolveAuvikMappedLabel(t *testing.T) {
	tenants := []auvik.Tenant{{ID: "t1", Name: "nanook"}}
	tm := auvik.TenantMap{
		Mappings: map[string]string{"t1": "Nanook Wireless"},
		Domains:  map[string]string{"t1": "nanook"},
	}
	if _, ok := ResolveAuvik("Nanook Wireless", tenants, tm); !ok {
		t.Fatal("Pathfinder tenant-map name should resolve")
	}
	if _, ok := ResolveAuvik("nanook", tenants, tm); !ok {
		t.Fatal("domain prefix should resolve")
	}
	if _, ok := ResolveAuvik("Some Other ISP", tenants, tm); ok {
		t.Fatal("unknown label must not resolve")
	}
}

func TestSeedAuvikTenantsDoesNotOverwrite(t *testing.T) {
	tm := auvik.TenantMap{
		Mappings: map[string]string{"t1": "Nanook Wireless"},
		Domains:  map[string]string{"t1": "nanook"},
	}
	got := SeedAuvikTenants(map[string]string{"Nanook Wireless": "keep-me"}, tm, nil)
	if got["Nanook Wireless"] != "keep-me" {
		t.Fatalf("seed overwrote installer pick: %v", got)
	}
	got = SeedAuvikTenants(nil, tm, nil)
	if got["Nanook Wireless"] != "nanook" {
		t.Fatalf("seed from Pathfinder: %v", got)
	}
}

func TestBackupAndRewrite(t *testing.T) {
	root := t.TempDir()
	sessions := filepath.Join(root, "Sessions", "Customers", "Acme")
	if err := os.MkdirAll(sessions, 0o755); err != nil {
		t.Fatal(err)
	}
	ini := []byte("S:\"Hostname\"=10.9.9.9\nD:\"[SSH2] Port\"=00000016\nS:\"Protocol Name\"=SSH2\n")
	if err := os.WriteFile(filepath.Join(sessions, "edge.ini"), ini, 0o644); err != nil {
		t.Fatal(err)
	}
	got := ListCustomerFolders(filepath.Join(root, "Sessions"), "Customers")
	if len(got) != 1 || got[0] != "Acme" {
		t.Fatalf("folders: %v", got)
	}
	appHome := t.TempDir()
	backup, err := BackupCustomerFolder(filepath.Join(root, "Sessions"), "Customers", appHome)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(backup, "Customers", "Acme", "edge.ini")); err != nil {
		t.Fatalf("backup missing: %v", err)
	}
}
