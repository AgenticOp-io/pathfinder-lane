package vpnprov

import (
	"net"
	"strings"
)

// IfaceInfo is one local adapter we might splice through.
type IfaceInfo struct {
	Name  string
	Index int
	IPv4  net.IP
}

// Dialer returns a TCP dialer bound to the customer VPN adapter when we
// can name it. Fail-open: unknown/unmatched targets use the default route.
func Dialer(target string) *net.Dialer {
	d := &net.Dialer{}
	info, ok := LookupIface(target)
	if !ok {
		return d
	}
	if info.IPv4 != nil {
		d.LocalAddr = &net.TCPAddr{IP: info.IPv4}
	}
	d.Control = bindControl(info)
	return d
}

// LookupIface finds a local interface that matches the mapped VPN target.
func LookupIface(target string) (IfaceInfo, bool) {
	t := ParseTarget(target)
	if t.Name == "" {
		return IfaceInfo{}, false
	}
	return MatchVPNInterface(t, listIfaces())
}

// MatchVPNInterface picks the adapter for a VPN target. Names are matched
// conservatively — we never guess a generic Ethernet/Wi-Fi NIC.
func MatchVPNInterface(t Target, ifaces []IfaceInfo) (IfaceInfo, bool) {
	if t.Name == "" || len(ifaces) == 0 {
		return IfaceInfo{}, false
	}
	var hits []IfaceInfo
	for _, ifc := range ifaces {
		if ifaceMatches(t, ifc) {
			hits = append(hits, ifc)
		}
	}
	if len(hits) == 1 {
		return hits[0], true
	}
	if len(hits) > 1 {
		name := strings.ToLower(strings.TrimSpace(t.Name))
		for _, ifc := range hits {
			if strings.EqualFold(strings.TrimSpace(ifc.Name), t.Name) || strings.EqualFold(ifc.Name, name) {
				return ifc, true
			}
		}
	}
	return IfaceInfo{}, false
}

func ifaceMatches(t Target, ifc IfaceInfo) bool {
	n := strings.ToLower(strings.TrimSpace(ifc.Name))
	if n == "" || strings.Contains(n, "loopback") || n == "lo" {
		return false
	}
	want := strings.ToLower(strings.TrimSpace(t.Name))
	switch t.Kind {
	case WireGuard:
		return n == want || strings.Contains(n, want)
	case Zscaler:
		return strings.Contains(n, "zscaler") || strings.Contains(n, "zpa") || strings.Contains(n, "zsn")
	default: // FortiClient
		return strings.Contains(n, "forti") || strings.Contains(n, "fortissl") || strings.Contains(n, "sslvpn") || strings.Contains(n, "ssl vpn")
	}
}

func listIfaces() []IfaceInfo {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var out []IfaceInfo
	for _, ifc := range ifaces {
		if ifc.Flags&net.FlagUp == 0 || ifc.Flags&net.FlagLoopback != 0 {
			continue
		}
		info := IfaceInfo{Name: ifc.Name, Index: ifc.Index}
		addrs, err := ifc.Addrs()
		if err != nil {
			continue
		}
		for _, a := range addrs {
			var ip net.IP
			switch v := a.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip4 := ip.To4(); ip4 != nil && !ip4.IsLoopback() {
				info.IPv4 = ip4
				break
			}
		}
		if info.IPv4 == nil {
			continue
		}
		out = append(out, info)
	}
	return out
}
