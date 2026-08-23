package docvault

import (
	"fmt"
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/product"
	"github.com/scottpeterman/pathfinderssh/internal/sessions"
	"github.com/scottpeterman/pathfinderssh/internal/vault"
)

// Password is a normalized documentation-platform password record.
type Password struct {
	ID               string
	Name             string
	Username         string
	Password         string
	OrganizationName string
	URL              string
	ResourceURL      string
	CategoryName     string
}

// ImportStats counts vault import operations.
type ImportStats struct {
	Added    int
	Updated  int
	Skipped  int
	Failed   int
	Errors   []string
}

// LinkStats counts session linking operations.
type LinkStats struct {
	Linked  int
	Skipped int
}

// VaultSyncOptions configures import into the encrypted vault.
type VaultSyncOptions struct {
	SourceTag      string // e.g. "itglue", "hudu"
	IDTagPrefix    string // e.g. "itglue-id:", "hudu-id:"
	UpdateExisting bool
	CustomerName   string // tags creds with customer:<name> for scoped ops desk
}

// SyncPasswordsToVault imports passwords into the encrypted vault.
func SyncPasswordsToVault(v *vault.Vault, passwords []Password, opts VaultSyncOptions) (ImportStats, error) {
	st := ImportStats{}
	if v == nil {
		return st, fmt.Errorf("vault not available")
	}
	opts.SourceTag = strings.TrimSpace(opts.SourceTag)
	opts.IDTagPrefix = strings.TrimSpace(opts.IDTagPrefix)
	if opts.IDTagPrefix == "" {
		opts.IDTagPrefix = opts.SourceTag + "-id:"
	}
	existing, err := v.All()
	if err != nil {
		return st, err
	}
	byTag := indexByIDTag(existing, opts.IDTagPrefix)

	for _, p := range passwords {
		if strings.TrimSpace(p.Password) == "" {
			st.Skipped++
			continue
		}
		tag := opts.IDTagPrefix + p.ID
		name := CredentialName(p)
		custTag := sessions.CustomerTag(opts.CustomerName)
		if cur, ok := byTag[tag]; ok && opts.UpdateExisting {
			cur.Username = p.Username
			cur.Password = p.Password
			cur.AuthType = "password"
			if cur.Description == "" {
				cur.Description = description(p)
			}
			if !hasTag(cur.Tags, tag) {
				cur.Tags = append(cur.Tags, tag, opts.SourceTag)
			}
			if custTag != "" && !hasTag(cur.Tags, custTag) {
				cur.Tags = append(cur.Tags, custTag)
			}
			if err := v.Update(cur); err != nil {
				st.Failed++
				st.Errors = append(st.Errors, err.Error())
				continue
			}
			st.Updated++
			continue
		}
		if _, err := v.Get(name); err == nil {
			st.Skipped++
			continue
		}
		c := vault.Credential{
			Name:        name,
			Username:    p.Username,
			AuthType:    "password",
			Password:    p.Password,
			Description: description(p),
			Tags:        []string{opts.SourceTag, tag},
		}
		if custTag != "" {
			c.Tags = append(c.Tags, custTag)
		}
		if _, err := v.Add(c); err != nil {
			if err == vault.ErrDuplicateName {
				st.Skipped++
				continue
			}
			st.Failed++
			st.Errors = append(st.Errors, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		st.Added++
	}
	return st, nil
}

// CredentialName builds a stable vault entry name from a password record.
func CredentialName(p Password) string {
	name := strings.TrimSpace(p.Name)
	if name == "" {
		name = "password-" + p.ID
	}
	org := strings.TrimSpace(p.OrganizationName)
	if org != "" && !strings.Contains(strings.ToLower(name), strings.ToLower(org)) {
		return org + " · " + name
	}
	return name
}

func description(p Password) string {
	parts := []string{strings.TrimSpace(p.OrganizationName) + " password"}
	if p.CategoryName != "" {
		parts = append(parts, p.CategoryName)
	}
	if p.ResourceURL != "" {
		parts = append(parts, p.ResourceURL)
	}
	return strings.Join(parts, " · ")
}

func indexByIDTag(creds []vault.Credential, prefix string) map[string]vault.Credential {
	out := map[string]vault.Credential{}
	for _, c := range creds {
		for _, t := range c.Tags {
			if strings.HasPrefix(t, prefix) {
				out[t] = c
				break
			}
		}
	}
	return out
}

func hasTag(tags []string, want string) bool {
	for _, t := range tags {
		if t == want {
			return true
		}
	}
	return false
}

// CredNamesFromVault maps external password id → vault credential name.
func CredNamesFromVault(v *vault.Vault, idTagPrefix string) (map[string]string, error) {
	out := map[string]string{}
	if v == nil {
		return out, fmt.Errorf("vault not available")
	}
	creds, err := v.All()
	if err != nil {
		return nil, err
	}
	for _, c := range creds {
		for _, t := range c.Tags {
			if strings.HasPrefix(t, idTagPrefix) {
				out[strings.TrimPrefix(t, idTagPrefix)] = c.Name
			}
		}
	}
	return out, nil
}

// LinkOptions configures session ↔ vault linking.
type LinkOptions struct {
	CustomerName string
	OnlyEmpty    bool
	IDField      string // session field to store password id — uses ExternalPasswordID
}

// LinkSessions attaches vault credentials to SSH sessions under a customer.
func LinkSessions(t *sessions.Tree, passwords []Password, credNames map[string]string, opts LinkOptions) LinkStats {
	st := LinkStats{}
	if t == nil {
		return st
	}
	customer := strings.TrimSpace(opts.CustomerName)
	if customer == "" {
		return st
	}
	prefix := sessions.CustomerPath(product.CustomersRoot, customer)

	t.WalkSessions(func(folder string, n sessions.Node) {
		if !isUnderCustomerFolder(folder, prefix) {
			return
		}
		if opts.OnlyEmpty && strings.TrimSpace(n.Credential) != "" {
			st.Skipped++
			return
		}
		credName, pwID := matchCredential(n, passwords, credNames)
		if credName == "" {
			st.Skipped++
			return
		}
		n = n.Normalize()
		n.Credential = credName
		n.AuthType = sessions.AuthPassword
		if strings.TrimSpace(n.Username) == "" {
			for _, p := range passwords {
				if p.ID == pwID && p.Username != "" {
					n.Username = p.Username
					break
				}
			}
		}
		n.ExternalPasswordID = pwID
		if err := t.Replace(folder, n.Label(), n); err != nil {
			st.Skipped++
			return
		}
		st.Linked++
	})
	return st
}

func matchCredential(n sessions.Node, passwords []Password, credNames map[string]string) (credName, pwID string) {
	if id := strings.TrimSpace(n.ExternalPasswordID); id != "" {
		if name := credNames[id]; name != "" {
			return name, id
		}
	}
	if id := strings.TrimSpace(n.ITGluePasswordID); id != "" {
		if name := credNames[id]; name != "" {
			return name, id
		}
	}
	host := strings.ToLower(strings.TrimSpace(n.Host))
	label := strings.ToLower(strings.TrimSpace(n.Normalize().Label()))
	for _, p := range passwords {
		name := credNames[p.ID]
		if name == "" {
			continue
		}
		pn := strings.ToLower(p.Name)
		url := strings.ToLower(p.URL)
		res := strings.ToLower(p.ResourceURL)
		if host != "" && (strings.Contains(url, host) || strings.Contains(res, host)) {
			return name, p.ID
		}
		if label != "" && (strings.Contains(pn, label) || strings.Contains(label, pn)) {
			return name, p.ID
		}
		if host != "" && strings.Contains(pn, host) {
			return name, p.ID
		}
	}
	return "", ""
}

func isUnderCustomerFolder(folder, prefix string) bool {
	folder = strings.TrimSpace(folder)
	prefix = strings.TrimSpace(prefix)
	if folder == "" || prefix == "" {
		return false
	}
	if folder == prefix {
		return true
	}
	return strings.HasPrefix(folder, prefix+sessions.PathSep)
}
