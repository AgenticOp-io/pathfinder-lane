package lanectl

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPathHasDir(t *testing.T) {
	dir := `C:\Users\me\AppData\Local\PathfinderCRT-Bridge\bin`
	path := `C:\Windows\system32;C:\Users\me\AppData\Local\PathfinderCRT-Bridge\bin;C:\Go\bin`
	if runtime.GOOS == "windows" {
		if !pathHasDir(path, dir) {
			t.Fatal("expected hit")
		}
		if pathHasDir(path, `C:\missing`) {
			t.Fatal("missing dir")
		}
		return
	}
	if !pathHasDir("/usr/bin:/home/me/.local/bin", "/home/me/.local/bin") {
		t.Fatal("unix hit")
	}
}

func TestEnsurePathSnippetIdempotent(t *testing.T) {
	dir := t.TempDir()
	rc := filepath.Join(dir, ".zprofile")
	if err := ensurePathSnippet(rc); err != nil {
		t.Fatal(err)
	}
	if err := ensurePathSnippet(rc); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(rc)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(raw), pathMarker); n != 1 {
		t.Fatalf("marker count %d: %s", n, raw)
	}
}

func TestParseProxyJump(t *testing.T) {
	u, h, p := parseProxyJump("admin@10.9.9.9:2222")
	if u != "admin" || h != "10.9.9.9" || p != 2222 {
		t.Fatalf("%s %s %d", u, h, p)
	}
	u, h, p = parseProxyJump("bastion.example")
	if u != "" || h != "bastion.example" || p != 22 {
		t.Fatalf("%s %s %d", u, h, p)
	}
}

func TestParseSSHConfigProxyJump(t *testing.T) {
	got := parseSSHConfigHosts("Host core\n  HostName 10.1.1.1\n  ProxyJump jump\n")
	if len(got) != 1 || got[0].JumpHost != "jump" {
		t.Fatalf("%+v", got)
	}
}

func TestRenderSSHConfigJump(t *testing.T) {
	got := RenderSSHConfig("/opt/pflane", []SSHHost{{
		Alias:    "acme-core",
		Folder:   "Acme",
		Host:     "10.1.1.1",
		Port:     22,
		Via:      "proxy",
		JumpHost: "10.9.9.9",
		JumpUser: "jump",
	}})
	for _, p := range []string{"ProxyJump", "HostName 10.9.9.9", "ProxyCommand", "Host acme-core", "HostName 10.1.1.1"} {
		if !strings.Contains(got, p) {
			t.Fatalf("missing %q in %s", p, got)
		}
	}
	if strings.Contains(got, "HostName 10.1.1.1\n  Port 22\n  ProxyCommand") {
		t.Fatalf("device must not also have ProxyCommand:\n%s", got)
	}
	if !strings.Contains(got, "ProxyJump") {
		t.Fatal(got)
	}
}
