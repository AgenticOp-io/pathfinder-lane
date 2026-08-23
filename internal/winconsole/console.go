//go:build !windows

package winconsole

// AttachOrAllocate is a no-op on non-Windows platforms.
func AttachOrAllocate() {}

// Hide is a no-op on non-Windows platforms.
func Hide() {}
