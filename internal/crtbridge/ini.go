package crtbridge

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// ReadSSHHostPort returns Hostname and SSH2 port from a CRT session ini.
func ReadSSHHostPort(raw []byte) (host string, port int, ok bool) {
	port = 22
	sc := bufio.NewScanner(bytes.NewReader(raw))
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 1024*1024)
	var haveHost bool
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		key, val, parsed := splitINI(line)
		if !parsed {
			continue
		}
		switch key {
		case `S:"Hostname"`:
			host = val
			haveHost = host != ""
		case `D:"[SSH2] Port"`, `D:"Port"`:
			if p, err := strconv.ParseInt(val, 16, 32); err == nil && p > 0 {
				port = int(p)
			} else if p, err := strconv.Atoi(val); err == nil && p > 0 {
				port = p
			}
		}
	}
	return host, port, haveHost
}

// PatchSSHHostPort rewrites Hostname and SSH2 port, leaving every other line.
func PatchSSHHostPort(raw []byte, host string, port int) []byte {
	if port <= 0 {
		port = 22
	}
	hostLine := `S:"Hostname"=` + host
	portLine := fmt.Sprintf(`D:"[SSH2] Port"=%08x`, port)
	var out bytes.Buffer
	sc := bufio.NewScanner(bytes.NewReader(raw))
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 1024*1024)
	var wroteHost, wrotePort bool
	for sc.Scan() {
		line := sc.Text()
		key, _, parsed := splitINI(strings.TrimSpace(line))
		switch {
		case parsed && key == `S:"Hostname"`:
			out.WriteString(hostLine)
			wroteHost = true
		case parsed && (key == `D:"[SSH2] Port"` || key == `D:"Port"`):
			out.WriteString(portLine)
			wrotePort = true
		default:
			out.WriteString(line)
		}
		out.WriteByte('\n')
	}
	if !wroteHost {
		out.WriteString(hostLine)
		out.WriteByte('\n')
	}
	if !wrotePort {
		out.WriteString(portLine)
		out.WriteByte('\n')
	}
	return out.Bytes()
}

func splitINI(line string) (key, val string, ok bool) {
	eq := strings.IndexByte(line, '=')
	if eq < 0 {
		return "", "", false
	}
	return line[:eq], line[eq+1:], true
}
