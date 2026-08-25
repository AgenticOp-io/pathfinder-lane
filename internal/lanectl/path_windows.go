//go:build windows

package lanectl

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

func osLookPath(name string) (string, error) {
	return exec.LookPath(name)
}

func installOnPATH(exe string) (string, error) {
	dir := filepath.Dir(exe)
	k, err := registry.OpenKey(registry.CURRENT_USER, `Environment`, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return "", err
	}
	defer k.Close()
	cur, typ, err := k.GetStringValue("Path")
	if err == registry.ErrNotExist {
		cur, typ, err = "", registry.EXPAND_SZ, nil
	}
	if err != nil {
		return "", err
	}
	if pathHasDir(cur, dir) {
		return dir, nil
	}
	next := strings.TrimRight(cur, `;`)
	if next != "" {
		next += `;`
	}
	next += dir
	switch typ {
	case registry.EXPAND_SZ:
		err = k.SetExpandStringValue("Path", next)
	default:
		err = k.SetStringValue("Path", next)
	}
	if err != nil {
		return "", fmt.Errorf("user PATH: %w", err)
	}
	notifyEnvChange()
	return dir, nil
}

func notifyEnvChange() {
	env, err := windows.UTF16PtrFromString("Environment")
	if err != nil {
		return
	}
	user32 := windows.NewLazySystemDLL("user32.dll")
	proc := user32.NewProc("SendMessageTimeoutW")
	const (
		hwndBroadcast  = uintptr(0xffff)
		wmSettingChange = 0x001A
		smtoAbortIfHung = 0x0002
	)
	var dummy uintptr
	_, _, _ = proc.Call(
		hwndBroadcast,
		uintptr(wmSettingChange),
		0,
		uintptr(unsafe.Pointer(env)),
		uintptr(smtoAbortIfHung),
		2000,
		uintptr(unsafe.Pointer(&dummy)),
	)
}
