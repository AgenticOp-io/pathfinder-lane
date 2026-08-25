//go:build !linux && !darwin && !windows

package vpnprov

func bindToVPN(info IfaceInfo) func(fd uintptr) error {
	_ = info
	return nil
}
