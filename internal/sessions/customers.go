// MSP inventory layout: built-in Customers + Unassigned roots.
//
// SecureCRT import is organised by a wizard: the operator picks which CRT
// top-level folder is the customer list. That folder's children become
// Customers/<name>/ with nested folders preserved. Everything else moves under
// Unassigned/, also keeping folder structure.
package sessions

import (
	"fmt"
	"sort"
	"strings"
)

// DefaultCustomersRoot / DefaultUnassignedRoot match product.CustomersRoot and
// product.UnassignedRoot without importing product (keeps this package free of
// UI/product deps).
const (
	DefaultCustomersRoot   = "Customers"
	DefaultUnassignedRoot  = "Unassigned"
	LegacyCRTCustomersRoot = "3_Customers"
)

// EnsureMSPLayout creates the built-in roots when missing. It does not rewrite
// SecureCRT trees — that is OrganiseCRTImport after the import wizard.
func (t *Tree) EnsureMSPLayout() (changed bool, err error) {
	for _, root := range []string{DefaultCustomersRoot, DefaultUnassignedRoot} {
		if _, e := t.FolderAt(root); e != nil {
			if e := t.AddFolder(root); e != nil {
				return changed, e
			}
			changed = true
		}
	}
	return changed, nil
}

// ResetMSPInventory clears the tree to empty Customers + Unassigned roots.
func (t *Tree) ResetMSPInventory() error {
	t.Folders = nil
	if err := t.AddFolder(DefaultCustomersRoot); err != nil {
		return err
	}
	return t.AddFolder(DefaultUnassignedRoot)
}

// TopLevelFolderNames returns sorted names of immediate child folders.
func (t Tree) TopLevelFolderNames() []string {
	out := make([]string, 0, len(t.Folders))
	for _, f := range t.Folders {
		name := strings.TrimSpace(f.Name)
		if name == "" {
			continue
		}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// TopLevelNamesFromFolders is TopLevelFolderNames for a bare import list.
func TopLevelNamesFromFolders(folders []Folder) []string {
	var tr Tree
	tr.Folders = append([]Folder(nil), folders...)
	return tr.TopLevelFolderNames()
}

// OrganiseCRTImport moves a freshly imported SecureCRT tree into MSP roots.
//
// customerRoot is a top-level folder name (e.g. "3_Customers"). Its immediate
// child folders become Customers/<child>/ with nested structure kept. Meta
// buckets (0_OLD_…) and sessions sitting on the customer root go to Unassigned.
// Every other top-level folder moves under Unassigned/<name>/ (structure kept).
//
// Pass customerRoot "" to send the entire import under Unassigned.
func (t *Tree) OrganiseCRTImport(customerRoot string) error {
	if _, err := t.EnsureMSPLayout(); err != nil {
		return err
	}
	customerRoot = strings.TrimSpace(customerRoot)

	if customerRoot != "" {
		if err := t.adoptCustomerRoot(customerRoot); err != nil {
			return err
		}
	}

	var extras []string
	for _, f := range t.Folders {
		name := strings.TrimSpace(f.Name)
		switch name {
		case DefaultCustomersRoot, DefaultUnassignedRoot, "":
			continue
		default:
			extras = append(extras, name)
		}
	}
	for _, name := range extras {
		if err := t.MoveFolder(name, DefaultUnassignedRoot); err != nil {
			return fmt.Errorf("move %q into Unassigned: %w", name, err)
		}
	}
	return nil
}

func (t *Tree) adoptCustomerRoot(customerRoot string) error {
	src, err := t.FolderAt(customerRoot)
	if err != nil {
		return fmt.Errorf("customer list folder %q not found in import", customerRoot)
	}

	// Snapshot children — MoveFolder mutates the parent.
	children := append([]Folder(nil), src.Folders...)
	directSessions := append([]Node(nil), src.Sessions...)

	for _, child := range children {
		name := strings.TrimSpace(child.Name)
		if name == "" {
			continue
		}
		from := JoinPath(customerRoot, name)
		if IsMetaCustomerFolder(name) {
			if err := t.MoveFolder(from, DefaultUnassignedRoot); err != nil {
				return err
			}
			continue
		}
		if err := t.MoveFolder(from, DefaultCustomersRoot); err != nil {
			// Name already under Customers — merge by moving into a disambiguated folder.
			if strings.Contains(err.Error(), "already exists") {
				alt := name + " (imported)"
				// Rename leaf in place then move.
				if f, e := t.FolderAt(from); e == nil {
					f.Name = alt
					from = JoinPath(customerRoot, alt)
				}
				if err := t.MoveFolder(from, DefaultCustomersRoot); err != nil {
					return err
				}
				continue
			}
			return err
		}
	}

	for _, n := range directSessions {
		_ = t.addDisambiguated(DefaultUnassignedRoot, n)
	}
	// Clear sessions on the now-empty CRT root, then remove it.
	if f, err := t.FolderAt(customerRoot); err == nil {
		f.Sessions = nil
		f.Folders = nil
	}
	return t.RemoveFolder(customerRoot, true)
}

func (t *Tree) addDisambiguated(folder string, n Node) error {
	n = n.Normalize()
	want := n.Label()
	if f, err := t.FolderAt(folder); err == nil && f.SessionIndex(want) >= 0 {
		if host := strings.TrimSpace(n.Host); host != "" {
			n.Name = want + " (" + host + ")"
		} else {
			n.Name = want + " (imported)"
		}
		if f.SessionIndex(n.Label()) >= 0 {
			n.Name = n.Label() + " (2)"
		}
	}
	return t.Add(folder, n)
}

// EnsureCustomersRoot creates the customers root folder when missing.
func (t *Tree) EnsureCustomersRoot(root string) error {
	root = strings.TrimSpace(root)
	if root == "" {
		root = DefaultCustomersRoot
	}
	if _, err := t.FolderAt(root); err == nil {
		return nil
	}
	return t.AddFolder(root)
}

// ListCustomers returns immediate child folder names under the customers root.
func (t Tree) ListCustomers(root string) []string {
	root = strings.TrimSpace(root)
	if root == "" {
		root = DefaultCustomersRoot
	}
	f, err := t.FolderAt(root)
	if err != nil {
		return nil
	}
	out := make([]string, 0, len(f.Folders))
	for _, c := range f.Folders {
		name := strings.TrimSpace(c.Name)
		if name != "" {
			out = append(out, name)
		}
	}
	return out
}

// CustomerPath is the folder path for a customer under root.
func CustomerPath(root, customer string) string {
	root = strings.TrimSpace(root)
	if root == "" {
		root = DefaultCustomersRoot
	}
	return JoinPath(root, strings.TrimSpace(customer))
}

// CreateCustomer adds a new customer folder under root.
func (t *Tree) CreateCustomer(root, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", fmt.Errorf("customer name is required")
	}
	if strings.Contains(name, PathSep) || strings.Contains(name, " / ") {
		return "", fmt.Errorf("customer name must be a single folder name")
	}
	if IsMetaCustomerFolder(name) {
		return "", fmt.Errorf("customer name %q looks like a legacy CRT bucket, pick another", name)
	}
	if err := t.EnsureCustomersRoot(root); err != nil {
		return "", err
	}
	path := CustomerPath(root, name)
	if _, err := t.FolderAt(path); err == nil {
		return "", fmt.Errorf("a customer called %q already exists", name)
	}
	if err := t.AddFolder(path); err != nil {
		return "", err
	}
	return path, nil
}

// IsMetaCustomerFolder reports SecureCRT organisational buckets (0_OLD_CUSTOMERS).
func IsMetaCustomerFolder(name string) bool {
	name = strings.TrimSpace(name)
	if len(name) < 3 {
		return false
	}
	return name[0] >= '0' && name[0] <= '9' && name[1] == '_'
}

// MoveFolder relocates a folder (and its contents) under a new parent path.
// toParent may be "" to move to the tree root. The folder cannot be moved into
// itself or one of its descendants. Builtin roots cannot be moved.
func (t *Tree) MoveFolder(fromPath, toParent string) error {
	fromParts := SplitPath(fromPath)
	if len(fromParts) == 0 {
		return fmt.Errorf("folder path is required")
	}
	leaf := fromParts[len(fromParts)-1]
	if len(fromParts) == 1 && (leaf == DefaultCustomersRoot || leaf == DefaultUnassignedRoot) {
		return fmt.Errorf("cannot move the built-in %q folder", leaf)
	}

	toParent = strings.TrimSpace(toParent)
	toParts := SplitPath(toParent)
	fromJoined := JoinPath(fromParts...)

	if toParent != "" {
		toJoined := JoinPath(toParts...)
		if toJoined == fromJoined || strings.HasPrefix(toJoined+PathSep, fromJoined+PathSep) {
			return fmt.Errorf("cannot move a folder into itself")
		}
	}

	oldParentPath := ""
	if len(fromParts) > 1 {
		oldParentPath = JoinPath(fromParts[:len(fromParts)-1]...)
	}
	if JoinPath(SplitPath(oldParentPath)...) == JoinPath(SplitPath(toParent)...) {
		return nil
	}

	wantLeaf := strings.ToLower(leaf)

	if toParent == "" {
		if t.FolderIndex(leaf) >= 0 {
			return fmt.Errorf("a folder called %q already exists at the root", leaf)
		}
	} else {
		dest, err := t.EnsurePath(toParent)
		if err != nil {
			return err
		}
		for _, c := range dest.Folders {
			if strings.ToLower(strings.TrimSpace(c.Name)) == wantLeaf {
				return fmt.Errorf("a folder called %q already exists under %q", leaf, toParent)
			}
		}
	}

	var moved Folder
	if len(fromParts) == 1 {
		i := t.FolderIndex(fromParts[0])
		if i < 0 {
			return fmt.Errorf("no folder called %q", fromPath)
		}
		moved = t.Folders[i]
		t.Folders = append(t.Folders[:i], t.Folders[i+1:]...)
	} else {
		oldParent, err := t.FolderAt(oldParentPath)
		if err != nil {
			return fmt.Errorf("no folder called %q", fromPath)
		}
		idx := -1
		for i, c := range oldParent.Folders {
			if strings.ToLower(strings.TrimSpace(c.Name)) == wantLeaf {
				idx = i
				break
			}
		}
		if idx < 0 {
			return fmt.Errorf("no folder called %q", fromPath)
		}
		moved = oldParent.Folders[idx]
		oldParent.Folders = append(oldParent.Folders[:idx], oldParent.Folders[idx+1:]...)
	}

	if toParent == "" {
		t.Folders = append(t.Folders, moved)
		return nil
	}
	dest, err := t.FolderAt(toParent)
	if err != nil {
		t.Folders = append(t.Folders, moved)
		return err
	}
	dest.Folders = append(dest.Folders, moved)
	return nil
}
