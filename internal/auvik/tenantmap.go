package auvik

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

const tenantMapFile = "auvik-tenant-map.json"

// TenantMap persists Auvik tenant id → Customers/<folder> name overrides.
type TenantMap struct {
	Mappings map[string]string `json:"mappings"`
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
	return m, nil
}

// SaveTenantMap writes mappings to app home.
func SaveTenantMap(appHome string, m TenantMap) error {
	if m.Mappings == nil {
		m.Mappings = map[string]string{}
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
