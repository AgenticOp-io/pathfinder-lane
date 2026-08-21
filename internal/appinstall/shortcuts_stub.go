//go:build !windows

package appinstall

// CreateShortcuts is a no-op outside Windows.
func CreateShortcuts(exe string) error { return nil }

func removeShortcuts() error { return nil }
