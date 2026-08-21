//go:build !windows

package appinstall

// TrySingleton is a no-op outside Windows (portable / multi-instance OK).
func TrySingleton() (ok bool, release func()) {
	return true, func() {}
}
