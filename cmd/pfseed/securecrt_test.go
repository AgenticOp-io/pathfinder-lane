package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSplitCRTField(t *testing.T) {
	k, v, ok := splitCRTField(`S:"Hostname"=10.0.0.1`)
	if !ok || k != `S:"Hostname"` || v != "10.0.0.1" {
		t.Fatalf("got %q %q %v", k, v, ok)
	}
}

func TestParseCRTSessionINI(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "3_Customers", "Marshall")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(sub, "core-sw1.ini")
	content := "" +
		`S:"Hostname"=10.1.2.3` + "\n" +
		`S:"Username"=hsgeng` + "\n" +
		`S:"Protocol Name"=SSH2` + "\n" +
		`D:"[SSH2] Port"=00000016` + "\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	rel := filepath.Join("3_Customers", "Marshall", "core-sw1.ini")
	cs, err := parseCRTSessionINI(path, rel)
	if err != nil {
		t.Fatal(err)
	}
	if cs.Host != "10.1.2.3" || cs.User != "hsgeng" || cs.Protocol != "ssh" || cs.Port != 22 {
		t.Fatalf("%+v", cs)
	}
	if cs.Folder != "3_Customers / Marshall" {
		t.Fatalf("folder %q", cs.Folder)
	}
	n := crtToNode(cs)
	if n.Host != "10.1.2.3" || n.Username != "hsgeng" {
		t.Fatalf("node %+v", n)
	}
}
