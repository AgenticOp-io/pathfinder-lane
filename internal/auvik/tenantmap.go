package auvik

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/sessions"
)

const tenantMapFile = "auvik-tenant-map.json"

// TenantMap persists Auvik tenant id → Customers/<folder> name overrides
// and tenant id → domainPrefix for AuvikTunnel.
type TenantMap struct {
	Mappings map[string]string `json:"mappings"`
	Domains  map[string]string `json:"domains,omitempty"` // tenantID → domainPrefix (nanook, …)
	// SkipSync tenant IDs that return permission errors (MSP root, etc.).
	SkipSync map[string]string `json:"skip_sync,omitempty"` // tenantID → reason
}

// TenantMapPath is the JSON file location under app home.
func TenantMapPath(appHome string) string {
	return filepath.Join(appHome, tenantMapFile)
}

// LoadTenantMap reads mappings; missing file returns empty map.
func LoadTenantMap(appHome string) (TenantMap, error) {
	path := TenantMapPath(appHome)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return TenantMap{Mappings: map[string]string{}}, nil
		}
		return TenantMap{}, err
	}
	var m TenantMap
	if err := json.Unmarshal(raw, &m); err != nil {
		return TenantMap{}, err
	}
	if m.Mappings == nil {
		m.Mappings = map[string]string{}
	}
	if m.Domains == nil {
		m.Domains = map[string]string{}
	}
	if m.SkipSync == nil {
		m.SkipSync = map[string]string{}
	}
	return m, nil
}

// SaveTenantMap writes mappings to app home.
func SaveTenantMap(appHome string, m TenantMap) error {
	if m.Mappings == nil {
		m.Mappings = map[string]string{}
	}
	if m.Domains == nil {
		m.Domains = map[string]string{}
	}
	if m.SkipSync == nil {
		m.SkipSync = map[string]string{}
	}
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	path := TenantMapPath(appHome)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o600)
}

// CustomerForTenant returns the mapped customer folder or "".
func (m TenantMap) CustomerForTenant(tenantID string) string {
	if m.Mappings == nil {
		return ""
	}
	return strings.TrimSpace(m.Mappings[strings.TrimSpace(tenantID)])
}

// Set records tenant id → customer folder.
func (m *TenantMap) Set(tenantID, customer string) {
	if m.Mappings == nil {
		m.Mappings = map[string]string{}
	}
	tenantID = strings.TrimSpace(tenantID)
	customer = strings.TrimSpace(customer)
	if tenantID == "" || customer == "" {
		return
	}
	m.Mappings[tenantID] = customer
}

// DomainForTenant returns the Auvik domainPrefix for tunnels, or "".
func (m TenantMap) DomainForTenant(tenantID string) string {
	if m.Domains == nil {
		return ""
	}
	return strings.TrimSpace(m.Domains[strings.TrimSpace(tenantID)])
}

// SetDomain records tenant id → Auvik domainPrefix (for AuvikTunnel).
func (m *TenantMap) SetDomain(tenantID, domain string) {
	if m.Domains == nil {
		m.Domains = map[string]string{}
	}
	tenantID = strings.TrimSpace(tenantID)
	domain = strings.TrimSpace(domain)
	if tenantID == "" || domain == "" {
		return
	}
	m.Domains[tenantID] = domain
}

// ShouldSkipSync reports whether this tenant is on the deny/skip list.
func (m TenantMap) ShouldSkipSync(tenantID string) bool {
	if m.SkipSync == nil {
		return false
	}
	_, ok := m.SkipSync[strings.TrimSpace(tenantID)]
	return ok
}

// MarkSkipSync remembers a tenant that must not be synced (e.g. HTTP 403).
func (m *TenantMap) MarkSkipSync(tenantID, reason string) {
	if m.SkipSync == nil {
		m.SkipSync = map[string]string{}
	}
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return
	}
	if strings.TrimSpace(reason) == "" {
		reason = "permission denied"
	}
	m.SkipSync[tenantID] = reason
}

// IsPermissionDenied reports Auvik 403 / no-permission API errors.
func IsPermissionDenied(err error) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "403") ||
		strings.Contains(s, "do not have permission") ||
		strings.Contains(s, "permission to take this action")
}

// ResolveTunnelDomain returns the Auvik client domain for tunnels.
func ResolveTunnelDomain(appHome string, n sessions.Node) string {
	if d := strings.TrimSpace(n.AuvikDomain); d != "" {
		return d
	}
	tid := strings.TrimSpace(n.AuvikTenantID)
	if tid == "" {
		return ""
	}
	m, err := LoadTenantMap(appHome)
	if err != nil {
		return ""
	}
	return m.DomainForTenant(tid)
}

// DomainForCustomer returns the Auvik domainPrefix for a Customers/<name>
// folder using the tenant map (tenant → customer + tenant → domain).
func DomainForCustomer(appHome, customer string) string {
	customer = strings.TrimSpace(customer)
	if customer == "" {
		return ""
	}
	m, err := LoadTenantMap(appHome)
	if err != nil {
		return ""
	}
	for tid, name := range m.Mappings {
		if !strings.EqualFold(strings.TrimSpace(name), customer) {
			continue
		}
		if d := m.DomainForTenant(tid); d != "" {
			return d
		}
	}
	return ""
}

// WithCustomerTunnel stamps AuvikDomain onto a session from its customer folder
// when the node itself has no domain. Local/OOB sessions under an Auvik customer
// then share the same tunnel path as synced Auvik devices.
func WithCustomerTunnel(appHome, folder string, n sessions.Node) sessions.Node {
	out := n
	if ResolveTunnelDomain(appHome, out) != "" {
		return out
	}
	customer := sessions.CustomerOfFolder(folder)
	if d := DomainForCustomer(appHome, customer); d != "" {
		out.AuvikDomain = d
	}
	return out
}

// DomainFromCustomerSessions finds an Auvik domain on any session under
// Customers/<customer>/ (fallback when the tenant map has no domain yet).
func DomainFromCustomerSessions(tr sessions.Tree, customer string) string {
	customer = strings.TrimSpace(customer)
	if customer == "" {
		return ""
	}
	prefix := sessions.JoinPath(sessions.DefaultCustomersRoot, customer)
	var found string
	tr.WalkSessions(func(folder string, n sessions.Node) {
		if found != "" {
			return
		}
		if folder != prefix && !strings.HasPrefix(folder, prefix+"/") &&
			!strings.HasPrefix(folder, prefix+"\\") {
			// JoinPath uses / — also accept case-insensitive customer match.
			if !strings.EqualFold(sessions.CustomerOfFolder(folder), customer) {
				return
			}
		}
		if d := strings.TrimSpace(n.AuvikDomain); d != "" {
			found = d
		}
	})
	return found
}

// EnrichTunnelDomain applies tenant-map then inventory fallbacks so local
// sessions under an Auvik customer inherit the client domain.
func EnrichTunnelDomain(appHome, folder string, n sessions.Node, tr sessions.Tree) sessions.Node {
	out := WithCustomerTunnel(appHome, folder, n)
	if ResolveTunnelDomain(appHome, out) != "" {
		return out
	}
	if d := DomainFromCustomerSessions(tr, sessions.CustomerOfFolder(folder)); d != "" {
		out.AuvikDomain = d
	}
	return out
}

// ResolveCustomer picks mapped folder, else falls back to name resolution.
func ResolveCustomer(appHome, tenantID, tenantName string, customerNames []string, resolveName func(string) string) string {
	m, err := LoadTenantMap(appHome)
	if err == nil {
		if c := m.CustomerForTenant(tenantID); c != "" {
			if resolveName != nil {
				return resolveName(c)
			}
			return c
		}
	}
	name := strings.TrimSpace(tenantName)
	if name == "" {
		name = strings.TrimSpace(tenantID)
	}
	if resolveName != nil {
		return resolveName(name)
	}
	return name
}
