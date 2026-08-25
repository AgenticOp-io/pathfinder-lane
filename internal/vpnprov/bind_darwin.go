//go:build darwin

package vpnprov

import "golang.org/x/sys/unix"

func bindToVPN(info IfaceInfo) func(fd uintptr) error {
	if info.Index <= 0 {
		return nil
	}
	idx := info.Index
	return func(fd uintptr) error {
		return unix.SetsockoptInt(int(fd), unix.IPPROTO_IP, unix.IP_BOUND_IF, idx)
	}
}
