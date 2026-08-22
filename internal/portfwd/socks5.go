package portfwd

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"

	"golang.org/x/crypto/ssh"
)

// Minimal SOCKS5 CONNECT (no auth) for Dynamic forwards.
func serveSOCKS5(conn net.Conn, client *ssh.Client) {
	buf := make([]byte, 512)
	if _, err := io.ReadFull(conn, buf[:2]); err != nil {
		return
	}
	if buf[0] != 0x05 {
		return
	}
	nMethods := int(buf[1])
	if _, err := io.ReadFull(conn, buf[:nMethods]); err != nil {
		return
	}
	// no-auth
	if _, err := conn.Write([]byte{0x05, 0x00}); err != nil {
		return
	}
	if _, err := io.ReadFull(conn, buf[:4]); err != nil {
		return
	}
	if buf[0] != 0x05 || buf[1] != 0x01 { // CONNECT
		_ = socksFail(conn, 0x07)
		return
	}
	var host string
	var port uint16
	switch buf[3] {
	case 0x01: // IPv4
		if _, err := io.ReadFull(conn, buf[:4]); err != nil {
			return
		}
		host = net.IP(buf[:4]).String()
	case 0x03: // domain
		if _, err := io.ReadFull(conn, buf[:1]); err != nil {
			return
		}
		l := int(buf[0])
		if _, err := io.ReadFull(conn, buf[:l]); err != nil {
			return
		}
		host = string(buf[:l])
	case 0x04: // IPv6
		if _, err := io.ReadFull(conn, buf[:16]); err != nil {
			return
		}
		host = net.IP(buf[:16]).String()
	default:
		_ = socksFail(conn, 0x08)
		return
	}
	if _, err := io.ReadFull(conn, buf[:2]); err != nil {
		return
	}
	port = binary.BigEndian.Uint16(buf[:2])
	target := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	remote, err := client.Dial("tcp", target)
	if err != nil {
		_ = socksFail(conn, 0x05)
		return
	}
	defer remote.Close()
	// success: VER REP RSV ATYP BND.ADDR BND.PORT (0.0.0.0:0)
	if _, err := conn.Write([]byte{0x05, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0}); err != nil {
		return
	}
	pipe(conn, remote)
}

func socksFail(conn net.Conn, rep byte) error {
	_, err := conn.Write([]byte{0x05, rep, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
	return err
}
