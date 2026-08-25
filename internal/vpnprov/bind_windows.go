//go:build windows

package vpnprov

import "golang.org/x/sys/windows"

const ipUnicastIF = 31 // IPPROTO_IP / IP_UNICAST_IF

func bindToVPN(info IfaceInfo) func(fd uintptr) error {
	if info.Index <= 0 {
		return nil
	}
	v := htonl(uint32(info.Index))
	return func(fd uintptr) error {
		return windows.SetsockoptInt(windows.Handle(fd), windows.IPPROTO_IP, ipUnicastIF, int(v))
	}
}

func htonl(v uint32) uint32 {
	return v<<24 | v>>24 | (v&0xff00)<<8 | (v&0xff0000)>>8
}
