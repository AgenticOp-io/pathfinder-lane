//go:build windows

package winconsole

import (
	"syscall"
)

var (
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	procAttachConsole    = kernel32.NewProc("AttachConsole")
	procAllocConsole     = kernel32.NewProc("AllocConsole")
	procFreeConsole      = kernel32.NewProc("FreeConsole")
	user32               = syscall.NewLazyDLL("user32.dll")
	procGetConsoleWindow = user32.NewProc("GetConsoleWindow")
	procShowWindow       = user32.NewProc("ShowWindow")
)

const (
	attachParentProcess = ^uint32(0) // ATTACH_PARENT_PROCESS
	swHide                = 0
)

// AttachOrAllocate attaches to the parent console or allocates one for CLI output.
func AttachOrAllocate() {
	if r1, _, _ := procAttachConsole.Call(uintptr(attachParentProcess)); r1 != 0 {
		return
	}
	procAllocConsole.Call()
}

// Hide removes or hides the console window (graphical install without a console).
func Hide() {
	if r1, _, _ := procFreeConsole.Call(); r1 != 0 {
		return
	}
	hwnd, _, _ := procGetConsoleWindow.Call()
	if hwnd != 0 {
		procShowWindow.Call(hwnd, uintptr(swHide))
	}
}

// HasConsole reports whether this process already has a console window.
func HasConsole() bool {
	hwnd, _, _ := procGetConsoleWindow.Call()
	return hwnd != 0
}
