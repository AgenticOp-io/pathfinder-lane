package crtbridge

import (
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/auvik"
)

// Match is an Auvik tenant that corresponds to a CRT customer folder.
type Match struct {
	Domain   string
	TenantID string
}

// ResolveAuvik finds the Auvik tenant for an installer-mapped label (domain
// prefix, tenant id, or a Pathfinder tenant-map name). Do not pass a raw CRT
// folder name here — names do not match in the field.
func ResolveAuvik(label string, tenants []auvik.Tenant, tm auvik.TenantMap) (Match, bool) {
	label = strings.TrimSpace(label)
	if label == "" {
		return Match{}, false
	}
	for _, t := range tenants {
		if strings.EqualFold(strings.TrimSpace(t.Name), label) || strings.EqualFold(strings.TrimSpace(t.ID), label) {
			return Match{Domain: strings.TrimSpace(t.Name), TenantID: t.ID}, true
		}
		if mapped := tm.CustomerForTenant(t.ID); mapped != "" && strings.EqualFold(mapped, label) {
			domain := tm.DomainForTenant(t.ID)
			if domain == "" {
				domain = strings.TrimSpace(t.Name)
			}
			if domain != "" {
				return Match{Domain: domain, TenantID: t.ID}, true
			}
		}
		if d := tm.DomainForTenant(t.ID); d != "" && strings.EqualFold(d, label) {
			return Match{Domain: d, TenantID: t.ID}, true
		}
	}
	for tid, name := range tm.Mappings {
		if !strings.EqualFold(strings.TrimSpace(name), label) {
			continue
		}
		if d := tm.DomainForTenant(tid); d != "" {
			return Match{Domain: d, TenantID: tid}, true
		}
	}
	for tid, d := range tm.Domains {
		if strings.EqualFold(strings.TrimSpace(d), label) {
			return Match{Domain: strings.TrimSpace(d), TenantID: tid}, true
		}
	}
	return Match{}, false
}

// SeedAuvikTenants copies Pathfinder tenant-map rows into folder → Auvik
// label bindings when the installer has not mapped that folder yet.
func SeedAuvikTenants(existing map[string]string, tm auvik.TenantMap, tenants []auvik.Tenant) map[string]string {
	out := map[string]string{}
	for k, v := range existing {
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		if k != "" && v != "" {
			out[k] = v
		}
	}
	for tid, folder := range tm.Mappings {
		folder = strings.TrimSpace(folder)
		if folder == "" || out[folder] != "" {
			continue
		}
		name := strings.TrimSpace(tm.DomainForTenant(tid))
		if name == "" {
			for _, t := range tenants {
				if t.ID == tid {
					name = strings.TrimSpace(t.Name)
					break
				}
			}
		}
		if name != "" {
			out[folder] = name
		}
	}
	return out
}
