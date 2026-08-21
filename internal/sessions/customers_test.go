package sessions

import "testing"

func TestOrganiseCRTImportPreservesCustomerFolders(t *testing.T) {
	var tr Tree
	_ = tr.AddFolder("3_Customers/Sylacauga (SUB)/Servers/SSH")
	_ = tr.Add("3_Customers/Sylacauga (SUB)/Servers/SSH", Node{Name: "core", Host: "10.0.0.1", Transport: TransportSSH})
	_ = tr.AddFolder("3_Customers/0_OLD_CUSTOMERS/Hope/SSH")
	_ = tr.Add("3_Customers/0_OLD_CUSTOMERS/Hope/SSH", Node{Name: "old-sw", Host: "10.0.0.2", Transport: TransportSSH})
	_ = tr.AddFolder("Lab Gear/Rack")
	_ = tr.Add("Lab Gear/Rack", Node{Name: "lab1", Host: "10.0.0.3", Transport: TransportSSH})

	if err := tr.OrganiseCRTImport("3_Customers"); err != nil {
		t.Fatal(err)
	}
	if _, err := tr.FolderAt("3_Customers"); err == nil {
		t.Fatal("legacy customer root should be gone")
	}
	if _, err := tr.FolderAt("Customers/Sylacauga (SUB)/Servers/SSH"); err != nil {
		t.Fatalf("nested customer path missing: %v", err)
	}
	leaf, _ := tr.FolderAt("Customers/Sylacauga (SUB)/Servers/SSH")
	if leaf.SessionIndex("core") < 0 {
		t.Fatalf("session not under nested path: %+v", leaf.Sessions)
	}
	if _, err := tr.FolderAt("Unassigned/0_OLD_CUSTOMERS/Hope/SSH"); err != nil {
		t.Fatalf("meta bucket should keep folders under Unassigned: %v", err)
	}
	if _, err := tr.FolderAt("Unassigned/Lab Gear/Rack"); err != nil {
		t.Fatalf("other CRT folder should keep structure under Unassigned: %v", err)
	}
	got := tr.ListCustomers(DefaultCustomersRoot)
	if len(got) != 1 || got[0] != "Sylacauga (SUB)" {
		t.Fatalf("ListCustomers = %v", got)
	}
}

func TestEnsureMSPLayoutDoesNotFlatten(t *testing.T) {
	var tr Tree
	_ = tr.AddFolder("3_Customers/Acme/SSH")
	_ = tr.Add("3_Customers/Acme/SSH", Node{Name: "x", Host: "1.1.1.1", Transport: TransportSSH})
	changed, err := tr.EnsureMSPLayout()
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected Customers/Unassigned created")
	}
	if _, err := tr.FolderAt("3_Customers/Acme/SSH"); err != nil {
		t.Fatal("EnsureMSPLayout must not migrate CRT trees")
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
