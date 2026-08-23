package mspauth

// IntegrationsEnabled is true when the MSP integration stack (Auvik, PSA, vault,
// incidents, Cursor AI) should be available. Cloud sign-in is not required —
// solo and standalone installs configure integrations locally.
func IntegrationsEnabled(_ Enrollment) bool {
	return true
}
