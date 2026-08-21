// MSP inventory layout: built-in Customers + Unassigned roots.
//
// SecureCRT's "3_Customers/…" trees are migrated once: each real customer
// folder becomes Customers/<name> with sessions flattened into that folder;
// everything else (including 0_OLD_CUSTOMERS and non-customer CRT folders)
// lands as a flat list under Unassigned.
package sessions

import (
	"fmt"
	"strings"
)

// DefaultCustomersRoot / DefaultUnassignedRoot match product.CustomersRoot and
// product.UnassignedRoot without importing product (keeps this package free of
// UI/product deps).
const (
	DefaultCustomersRoot     = "Customers"
	DefaultUnassignedRoot    = "Unassigned"
	LegacyCRTCustomersRoot   = "3_Customers"
)

// EnsureMSPLayout creates the built-in roots and migrates a SecureCRT-shaped
// tree into Customers / Unassigned when needed. Returns whether the tree was
// changed (so the host can save).
func (t *Tree) EnsureMSPLayout() (changed bool, err error) {
	for _, root := range []string{DefaultCustomersRoot, DefaultUnassignedRoot} {
		if _, e := t.FolderAt(root); e != nil {
			if e := t.AddFolder(root); e != nil {
				return changed, e
			}
			changed = true
		}
	}

	if _, e := t.FolderAt(LegacyCRTCustomersRoot); e == nil {
		if e := t.migrateLegacyCRTCustomers(); e != nil {
			return changed, e
		}
		changed = true
	}

	// Any other top-level folder (old CRT site trees) → flatten into Unassigned.
	var extras []string
	for _, f := range t.Folders {
		name := strings.TrimSpace(f.Name)
		if name == DefaultCustomersRoot || name == DefaultUnassignedRoot || name == LegacyCRTCustomersRoot {
			continue
		}
		extras = append(extras, name)
	}
	for _, name := range extras {
		f, e := t.FolderAt(name)
		if e != nil {
			continue
		}
		if e := t.flattenFolderSessionsInto(DefaultUnassignedRoot, *f); e != nil {
			return changed, e
		}
		if e := t.RemoveFolder(name, true); e != nil {
			return changed, e
		}
		changed = true
	}
	return changed, nil
}

func (t *Tree) migrateLegacyCRTCustomers() error {
	legacy, err := t.FolderAt(LegacyCRTCustomersRoot)
	if err != nil {
		return nil
	}
	for _, child := range append([]Folder(nil), legacy.Folders...) {
		name := strings.TrimSpace(child.Name)
		if name == "" {
			continue
		}
		if IsMetaCustomerFolder(name) {
			if err := t.flattenFolderSessionsInto(DefaultUnassignedRoot, child); err != nil {
				return err
			}
			continue
		}
		path, err := t.CreateCustomer(DefaultCustomersRoot, name)
		if err != nil {
			// Already migrated on a previous partial run.
			if !strings.Contains(err.Error(), "already exists") {
				return err
			}
			path = CustomerPath(DefaultCustomersRoot, name)
		}
		if err := t.flattenFolderSessionsInto(path, child); err != nil {
			return err
		}
	}
	// Sessions sitting directly on 3_Customers (rare) → Unassigned.
	if err := t.flattenFolderSessionsInto(DefaultUnassignedRoot, Folder{Sessions: legacy.Sessions}); err != nil {
		return err
	}
	return t.RemoveFolder(LegacyCRTCustomersRoot, true)
}

// flattenFolderSessionsInto copies every session in src (recursively) into dest
// as a flat list. Names that collide are disambiguated with the host.
func (t *Tree) flattenFolderSessionsInto(dest string, src Folder) error {
	if _, err := t.EnsurePath(dest); err != nil {
		return err
	}
	var walk func(Folder)
	walk = func(f Folder) {
		for _, n := range f.Sessions {
			n = n.Normalize()
			if n.Key() == "" && strings.TrimSpace(n.Host) == "" && strings.TrimSpace(n.Name) == "" {
				continue
			}
			_ = t.addDisambiguated(dest, n)
		}
		for _, c := range f.Folders {
			walk(c)
		}
	}
	walk(src)
	return nil
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
		// Still colliding? append a counter.
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
