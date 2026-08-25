package crtbridge

import (
	"path/filepath"
	"strings"
	"unicode"

	"github.com/scottpeterman/pathfinderssh/internal/auvik"
)

// classifySession decides whether a CRT .ini uses the localhost proxy.
//
// mixed / auvik / forticlient never guess names. Auvik and customer VPNs
// (FortiClient, WireGuard, Zscaler) fire only from installer maps.
func ClassifySession(cfg Settings, customer, origHost string, origPort int, rel string, lookupReady bool, tenants []auvik.Tenant, tm auvik.TenantMap, prev Session) Session {
	return classifySession(cfg, customer, origHost, origPort, rel, lookupReady, tenants, tm, prev)
}

func classifySession(cfg Settings, customer, origHost string, origPort int, rel string, lookupReady bool, tenants []auvik.Tenant, tm auvik.TenantMap, prev Session) Session {
	entry := Session{
		OriginalHost: origHost,
		OriginalPort: origPort,
		Customer:     customer,
		DeviceIP:     origHost,
		Mode:         modeDirect,
	}

	if want := cfg.AuvikTenantForSession(rel, customer); want != "" {
		if lookupReady {
			if m, ok := ResolveAuvik(want, tenants, tm); ok {
				entry.UseAuvik = true
				entry.Domain = m.Domain
				entry.DeviceIP = deviceAddr(origHost)
			}
		} else if prev.UseAuvik || prev.Mode == modeAuvik {
			entry.UseAuvik = true
			entry.Domain = prev.Domain
			if prev.DeviceIP != "" {
				entry.DeviceIP = prev.DeviceIP
			}
		}
	}
	if vpn := cfg.VPNTunnelForSession(rel, customer); vpn != "" {
		entry.VPNTunnel = vpn
	}
	if entry.UseAuvik || entry.VPNTunnel != "" {
		entry.Mode = modeProxy
		entry.FrontPort = frontListenPort(rel)
		if entry.DeviceIP == "" {
			entry.DeviceIP = deviceAddr(origHost)
		}
	}
	return entry
}

// IsFortiGateSession reports FortiGate SSH sessions from the CRT relative
// path or session name (FortiGate folders, FG-60F / FG1800F names). FortiAP,
// FortiAnalyzer, FortiManager, and generic "firewall" folders are excluded.
func IsFortiGateSession(rel string) bool {
	s := strings.ToLower(filepath.ToSlash(strings.TrimSpace(rel)))
	if s == "" {
		return false
	}
	base := s
	if i := strings.LastIndex(s, "/"); i >= 0 {
		base = s[i+1:]
	}
	if strings.Contains(s, "fortiap") || strings.Contains(s, "fortianalyzer") || strings.Contains(s, "fortimanager") {
		return false
	}
	if strings.Contains(s, "fortigate") || strings.Contains(s, "fortios") {
		return true
	}
	return fortiGateDeviceName(strings.TrimSuffix(base, ".ini"))
}

func fortiGateDeviceName(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, " ", "")
	for _, prefix := range []string{"fgt", "fg"} {
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		rest := strings.TrimLeft(strings.TrimPrefix(name, prefix), "-_")
		if rest == "" {
			return false
		}
		r := []rune(rest)[0]
		return unicode.IsDigit(r)
	}
	return false
}
