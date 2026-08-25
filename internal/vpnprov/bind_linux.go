//go:build linux

package vpnprov

import "golang.org/x/sys/unix"

func bindToVPN(info IfaceInfo) func(fd uintptr) error {
	name := info.Name
	if name == "" {
		return nil
	}
	return func(fd uintptr) error {
		return unix.BindToDevice(int(fd), name)
	}
}
