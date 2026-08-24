//go:build windows

package auvik

import (
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const swHide = 0

var (
	user32                       = windows.NewLazySystemDLL("user32.dll")
	procEnumWindows              = user32.NewProc("EnumWindows")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	procShowWindow               = user32.NewProc("ShowWindow")
)

// hideSlotWindows hides AuvikTunnel windows under workDir for a short period.
// Auvik's release supervisor often flashes a console on first launch even when
// Start-Process uses -WindowStyle Hidden.
func hideSlotWindows(workDir string, d time.Duration) {
	workDir = strings.TrimSpace(workDir)
	if workDir == "" || d <= 0 {
		return
	}
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		hideAuvikWindowsUnder(workDir)
		time.Sleep(150 * time.Millisecond)
	}
}

func hideAuvikWindowsUnder(workDir string) {
	pids := auvikPidsUnder(workDir)
	if len(pids) == 0 {
		return
	}
	cb := syscall.NewCallback(func(hwnd uintptr, _ uintptr) uintptr {
		var pid uint32
		procGetWindowThreadProcessId.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
		if pids[pid] {
			procShowWindow.Call(hwnd, uintptr(swHide))
		}
		return 1
	})
	procEnumWindows.Call(cb, 0)
}

func auvikPidsUnder(workDir string) map[uint32]bool {
	out := map[uint32]bool{}
	root := strings.ToLower(filepath.Clean(workDir))
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return out
	}
	defer windows.CloseHandle(snap)
	var e windows.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	if err := windows.Process32First(snap, &e); err != nil {
		return out
	}
	for {
		name := windows.UTF16ToString(e.ExeFile[:])
		if strings.EqualFold(name, "AuvikTunnel.exe") {
			if img := processImagePath(e.ProcessID); img != "" {
				if strings.HasPrefix(strings.ToLower(filepath.Clean(img)), root) {
					out[e.ProcessID] = true
				}
			}
		}
		if err := windows.Process32Next(snap, &e); err != nil {
			break
		}
	}
	return out
}

func processImagePath(pid uint32) string {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return ""
	}
	defer windows.CloseHandle(h)
	var buf [windows.MAX_PATH]uint16
	n := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &n); err != nil {
		return ""
	}
	return windows.UTF16ToString(buf[:n])
}
