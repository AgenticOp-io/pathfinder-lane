package lanectl

import "testing"

func TestSSHAlias(t *testing.T) {
	if got := SSHAlias("Nanook Wireless", "core.ini"); got != "nanook-wireless-core" {
		t.Fatalf("%q", got)
	}
	if got := SSHAlias("Acme", "FG-60F"); got != "acme-fg-60f" {
		t.Fatalf("%q", got)
	}
}

func TestFolderOfName(t *testing.T) {
	folders := []string{"Acme", "Nanook Wireless"}
	if got := FolderOfName("Acme-core", folders); got != "Acme" {
		t.Fatalf("%q", got)
	}
	if got := FolderOfName("Nanook Wireless/edge", folders); got != "Nanook Wireless" {
		t.Fatalf("%q", got)
	}
	if got := FolderOfName("other", folders); got != "" {
		t.Fatalf("%q", got)
	}
}
