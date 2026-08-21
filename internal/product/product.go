// Package product is the PathfinderSSH MSP product identity.
package product

const (
	// Name is the full product name shown in the window and About box.
	Name = "PathfinderSSH MSP"

	// InstallDir is the LocalAppData folder name (Windows) / app bundle leaf.
	InstallDir = "PathfinderSSH-MSP"

	// ShortcutBase is the .lnk filename without extension.
	ShortcutBase = "PathfinderSSH MSP"

	// SingletonMutex is the Windows named mutex that keeps one instance.
	SingletonMutex = "Local\\PathfinderSSH-MSP-singleton"

	// CustomersRoot holds one subfolder per customer (MSP-managed, not CRT).
	CustomersRoot = "Customers"

	// UnassignedRoot is a flat list of connections that are not under a customer
	// (typical SecureCRT imports that were never filed by customer).
	UnassignedRoot = "Unassigned"

	// LegacyCRTCustomersRoot is the old SecureCRT import folder. Migrated once
	// into Customers / Unassigned and then removed from the live tree.
	LegacyCRTCustomersRoot = "3_Customers"
)

// BuiltinRoots is the global connection-pane folder list built into the product.
// The tree is always seeded with these; operators do not create them by hand.
var BuiltinRoots = []string{CustomersRoot, UnassignedRoot}
