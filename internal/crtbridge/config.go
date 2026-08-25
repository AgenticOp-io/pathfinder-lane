package crtbridge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// AutomationMode selects which last-mile tools the CRT companion runs.
const (
	AutoMixed       = "mixed"       // Auvik and customer VPNs only where a CRT folder is mapped
	AutoFortiClient = "forticlient" // mapped FortiClient / WireGuard / Zscaler; no Auvik
	AutoAuvik       = "auvik"       // Auvik for mapped CRT folders; no customer VPN
)

// Settings is the standalone companion config (~/.pathfinder-crt/settings.json).
type Settings struct {
	Mode         string            `json:"mode"`
	CustomerRoot string            `json:"customer_root,omitempty"`
	CRTConfig    string            `json:"crt_config,omitempty"`
	AuvikUser    string            `json:"auvik_username,omitempty"`
	AuvikKey     string            `json:"auvik_api_key,omitempty"`
	AuvikBase    string            `json:"auvik_base_url,omitempty"`
	TunnelBin    string            `json:"auvik_tunnel_path,omitempty"`
	VPNBin       string            `json:"forticlient_bin,omitempty"`
	VPNTools     string            `json:"forticlient_tools,omitempty"`
	WGBin        string            `json:"wireguard_bin,omitempty"`
	ZSABin       string            `json:"zscaler_bin,omitempty"`
	VPNDefault   string            `json:"forticlient_default_tunnel,omitempty"`
	VPNTunnels   map[string]string `json:"forticlient_tunnels,omitempty"` // customer/folder → VPN target
	AuvikTenants map[string]string `json:"auvik_tenants,omitempty"`       // customer/folder → Auvik tenant/domain
	HostBinds    map[string]string `json:"host_binds,omitempty"`          // session name / alias / host → customer; for PuTTY/OpenSSH that have no folders
}

func SettingsPath(appHome string) string {
	return filepath.Join(appHome, "settings.json")
}

func LoadSettings(appHome string) (Settings, error) {
	raw, err := os.ReadFile(SettingsPath(appHome))
	if err != nil {
		if os.IsNotExist(err) {
			return Settings{Mode: AutoMixed, VPNTunnels: map[string]string{}, AuvikTenants: map[string]string{}, HostBinds: map[string]string{}}, nil
		}
		return Settings{}, err
	}
	var s Settings
	if err := json.Unmarshal(raw, &s); err != nil {
		return Settings{}, err
	}
	s.Normalize()
	return s, nil
}

func SaveSettings(appHome string, s Settings) error {
	s.Normalize()
	if err := os.MkdirAll(appHome, 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(SettingsPath(appHome), raw, 0o600)
}

func (s *Settings) Normalize() {
	switch strings.ToLower(strings.TrimSpace(s.Mode)) {
	case AutoFortiClient, "forti", "vpn":
		s.Mode = AutoFortiClient
	case AutoAuvik:
		s.Mode = AutoAuvik
	default:
		s.Mode = AutoMixed
	}
	if s.VPNTunnels == nil {
		s.VPNTunnels = map[string]string{}
	}
	if s.AuvikTenants == nil {
		s.AuvikTenants = map[string]string{}
	}
	if s.HostBinds == nil {
		s.HostBinds = map[string]string{}
	}
}

func (s Settings) AuvikEnabled() bool {
	return s.Mode == AutoMixed || s.Mode == AutoAuvik
}

func (s Settings) FortiEnabled() bool {
	return s.Mode == AutoMixed || s.Mode == AutoFortiClient
}

// VPNEnabled is FortiClient, WireGuard, and Zscaler ZPA (same modes).
func (s Settings) VPNEnabled() bool {
	return s.FortiEnabled()
}

// HostFolder returns the customer bound to a session name, alias, or host.
// Names are never guessed; only explicit HostBinds count.
func (s Settings) HostFolder(keys ...string) string {
	if len(s.HostBinds) == 0 {
		return ""
	}
	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		if v := strings.TrimSpace(s.HostBinds[k]); v != "" {
			return v
		}
		for bk, bv := range s.HostBinds {
			if strings.EqualFold(strings.TrimSpace(bk), k) && strings.TrimSpace(bv) != "" {
				return strings.TrimSpace(bv)
			}
		}
	}
	return ""
}

// VPNTunnelFor returns the FortiClient tunnel mapped to a CRT customer folder.
func (s Settings) VPNTunnelFor(customer string) string {
	return s.VPNTunnelForSession("", customer)
}

// VPNTunnelForSession returns the tunnel for a session path. The longest
// mapped CRT folder that contains the session wins. Unmapped sessions stay
// on standard SSH unless VPNDefault is set. Folder names are never guessed
// as tunnel names.
func (s Settings) VPNTunnelForSession(rel, customer string) string {
	if !s.VPNEnabled() {
		return ""
	}
	return mappedFolderValue(rel, customer, s.VPNTunnels, s.VPNDefault)
}

// AuvikTenantForSession returns the Auvik tenant/domain mapped to this CRT
// path. Unmapped sessions never inherit a tenant from a similar folder name.
func (s Settings) AuvikTenantForSession(rel, customer string) string {
	if !s.AuvikEnabled() {
		return ""
	}
	return mappedFolderValue(rel, customer, s.AuvikTenants, "")
}

// mappedFolderValue returns the longest mapped CRT folder that covers the
// session. Folder names are never treated as values.
func mappedFolderValue(rel, customer string, m map[string]string, fallback string) string {
	bestKey, best := "", ""
	for folder, val := range m {
		folder, val = strings.TrimSpace(folder), strings.TrimSpace(val)
		if folder == "" || val == "" {
			continue
		}
		if !folderCovers(rel, customer, folder) {
			continue
		}
		if len(normFolderPath(folder)) >= len(normFolderPath(bestKey)) {
			bestKey = folder
			best = val
		}
	}
	if best != "" {
		return best
	}
	return strings.TrimSpace(fallback)
}

func normFolderPath(p string) string {
	p = filepath.ToSlash(strings.TrimSpace(p))
	p = strings.Trim(p, "/")
	return strings.ToLower(p)
}

func folderCovers(rel, customer, mapped string) bool {
	mapped = normFolderPath(mapped)
	if mapped == "" {
		return false
	}
	customer = strings.ToLower(strings.TrimSpace(customer))
	if customer != "" && mapped == customer {
		return true
	}
	rel = normFolderPath(rel)
	if rel == "" {
		return false
	}
	relDir := rel
	if strings.HasSuffix(rel, ".ini") {
		if i := strings.LastIndex(rel, "/"); i >= 0 {
			relDir = rel[:i]
		}
	}
	if relDir == mapped || strings.HasPrefix(relDir, mapped+"/") {
		return true
	}
	if strings.Contains(relDir, "/"+mapped+"/") || strings.HasSuffix(relDir, "/"+mapped) {
		return true
	}
	return false
}

// ParseTunnelLines reads "Folder=Tunnel" lines from the installer.
func ParseTunnelLines(text string) map[string]string {
	out := map[string]string{}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			k, v, ok = strings.Cut(line, "\t")
		}
		if !ok {
			continue
		}
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		if k != "" && v != "" {
			out[k] = v
		}
	}
	return out
}

func FormatTunnelLines(m map[string]string) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteByte('=')
		b.WriteString(m[k])
		b.WriteByte('\n')
	}
	return b.String()
}

// SuggestFolder picks a CRT folder whose name matches a FortiClient tunnel.
func SuggestFolder(tunnel string, folders []string) string {
	tunnel = strings.TrimSpace(tunnel)
	if tunnel == "" {
		return ""
	}
	for _, f := range folders {
		if strings.EqualFold(f, tunnel) {
			return f
		}
	}
	low := strings.ToLower(tunnel)
	var best string
	for _, f := range folders {
		fl := strings.ToLower(f)
		if len(fl) < 3 {
			continue
		}
		if strings.Contains(low, fl) || strings.Contains(fl, low) {
			if best == "" || len(f) > len(best) {
				best = f
			}
		}
	}
	return best
}
