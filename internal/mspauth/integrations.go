package mspauth

// IntegrationsEnabled is true when the org enrolled with cloud MSP sign-in
// (Microsoft 365 or Google). Solo/local installs hide integration settings
// and import actions to keep the UI simple.
func IntegrationsEnabled(enroll Enrollment) bool {
	return enroll.Provider.RequiresCloudLogin()
}
