package vpnprov

import "syscall"

func bindControl(info IfaceInfo) func(network, address string, c syscall.RawConn) error {
	fn := bindToVPN(info)
	if fn == nil {
		return nil
	}
	return func(network, address string, c syscall.RawConn) error {
		_ = c.Control(func(fd uintptr) {
			_ = fn(fd)
		})
		return nil // bind is best-effort; splice still fail-opens
	}
}
