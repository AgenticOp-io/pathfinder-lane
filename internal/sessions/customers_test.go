package sessions

import "testing"

func TestEnsureMSPLayoutMigratesCRTCustomers(t *testing.T) {
	var tr Tree
	_ = tr.AddFolder("3_Customers/Sylacauga (SUB)/Servers/SSH")
	_ = tr.Add("3_Customers/Sylacauga (SUB)/Servers/SSH", Node{Name: "core", Host: "10.0.0.1", Transport: TransportSSH})
	_ = tr.AddFolder("3_Customers/0_OLD_CUSTOMERS/Hope/SSH")
	_ = tr.Add("3_Customers/0_OLD_CUSTOMERS/Hope/SSH", Node{Name: "old-sw", Host: "10.0.0.2", Transport: TransportSSH})
	_ = tr.AddFolder("Lab Gear")
	_ = tr.Add("Lab Gear", Node{Name: "lab1", Host: "10.0.0.3", Transport: TransportSSH})

	changed, err := tr.EnsureMSPLayout()
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected migration changes")
	}
	if _, err := tr.FolderAt("3_Customers"); err == nil {
		t.Fatal("legacy 3_Customers should be gone")
	}
	if _, err := tr.FolderAt("Customers/Sylacauga (SUB)"); err != nil {
		t.Fatal(err)
	}
	cust, _ := tr.FolderAt("Customers/Sylacauga (SUB)")
	if cust.SessionIndex("core") < 0 && cust.SessionIndex("core (10.0.0.1)") < 0 {
		// Label may be "core"
		found := false
		for _, n := range cust.Sessions {
			if n.Host == "10.0.0.1" {
				found = true
			}
		}
		if !found {
			t.Fatalf("customer sessions = %+v", cust.Sessions)
		}
	}
	un, err := tr.FolderAt("Unassigned")
	if err != nil {
		t.Fatal(err)
	}
	var hosts []string
	for _, n := range un.Sessions {
		hosts = append(hosts, n.Host)
	}
	if !contains(hosts, "10.0.0.2") || !contains(hosts, "10.0.0.3") {
		t.Fatalf("Unassigned hosts = %v", hosts)
	}
	if len(un.Folders) != 0 {
		t.Fatalf("Unassigned should be flat, got folders %+v", un.Folders)
	}
	got := tr.ListCustomers(DefaultCustomersRoot)
	if len(got) != 1 || got[0] != "Sylacauga (SUB)" {
		t.Fatalf("ListCustomers = %v", got)
	}
}

func TestCreateCustomerUnderBuiltinRoot(t *testing.T) {
	var tr Tree
	_, _ = tr.EnsureMSPLayout()
	path, err := tr.CreateCustomer(DefaultCustomersRoot, "Acme Fiber")
	if err != nil {
		t.Fatal(err)
	}
	if path != "Customers/Acme Fiber" {
		t.Fatalf("path = %q", path)
	}
}

func TestCannotMoveBuiltinRoot(t *testing.T) {
	var tr Tree
	_, _ = tr.EnsureMSPLayout()
	if err := tr.MoveFolder("Customers", "Unassigned"); err == nil {
		t.Fatal("expected error moving builtin root")
	}
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}
