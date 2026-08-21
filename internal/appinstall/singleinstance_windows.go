//go:build windows

package appinstall

import (
	"github.com/scottpeterman/pathfinderssh/internal/product"
	"golang.org/x/sys/windows"
)

// TrySingleton returns false when another PathfinderSSH MSP instance already
// holds the named mutex. release closes the handle when this process owns it.
func TrySingleton() (ok bool, release func()) {
	name, err := windows.UTF16PtrFromString(product.SingletonMutex)
	if err != nil {
		return true, func() {}
	}
	handle, err := windows.CreateMutex(nil, false, name)
	if handle == 0 {
		return true, func() {}
	}
	if err == windows.ERROR_ALREADY_EXISTS {
		_ = windows.CloseHandle(handle)
		return false, func() {}
	}
	return true, func() {
		_ = windows.CloseHandle(handle)
	}
}
