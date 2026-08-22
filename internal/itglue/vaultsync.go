package itglue

import (
	"fmt"
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/product"
	"github.com/scottpeterman/pathfinderssh/internal/sessions"
	"github.com/scottpeterman/pathfinderssh/internal/vault"
)

// VaultSyncOptions configures IT Glue → vault import.
type VaultSyncOptions struct {
	UpdateExisting bool
}

// SyncPasswordsToVault imports IT Glue passwords into the encrypted vault.
func SyncPasswordsToVault(v *vault.Vault, passwords []Password, opts VaultSyncOptions) (ImportStats, error) {
	st := ImportStats{}
	if v == nil {
		return st, fmt.Errorf("vault not available")
	}
	existing, err := v.All()
	if err != nil {
		return st, err
	}
	byTag := indexByITGlueTag(existing)

	for _, p := range passwords {
		if strings.TrimSpace(p.Password) == "" {
			st.Skipped++
			continue
		}
		tag := ITGlueTag(p.ID)
		name := VaultCredentialName(p)
		if cur, ok := byTag[tag]; ok && opts.UpdateExisting {
			cur.Username = p.Username
			cur.Password = p.Password
			cur.AuthType = "password"
			if cur.Description == "" {
				cur.Description = vaultDescription(p)
			}
			if !hasTag(cur.Tags, tag) {
				cur.Tags = append(cur.Tags, tag, "itglue")
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
			Description: vaultDescription(p),
			Tags:        []string{"itglue", tag},
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

// CredNamesFromVault maps IT Glue password id → vault credential name.
func CredNamesFromVault(v *vault.Vault) (map[string]string, error) {
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
			if strings.HasPrefix(t, "itglue-id:") {
				out[strings.TrimPrefix(t, "itglue-id:")] = c.Name
			}
		}
	}
	return out, nil
}

func vaultDescription(p Password) string {
	parts := []string{"IT Glue password"}
	if p.OrganizationName != "" {
		parts = append(parts, p.OrganizationName)
	}
	if p.CategoryName != "" {
		parts = append(parts, p.CategoryName)
	}
	if p.ResourceURL != "" {
		parts = append(parts, p.ResourceURL)
	}
	return strings.Join(parts, " · ")
}

func indexByITGlueTag(creds []vault.Credential) map[string]vault.Credential {
	out := map[string]vault.Credential{}
	for _, c := range creds {
		for _, t := range c.Tags {
			if strings.HasPrefix(t, "itglue-id:") {
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

// LinkOptions configures session ↔ vault credential linking.
type LinkOptions struct {
	CustomerName string
	OnlyEmpty    bool
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
		if !isUnderFolder(folder, prefix) {
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
		n.ITGluePasswordID = pwID
		if err := t.Replace(folder, n.Label(), n); err != nil {
			st.Skipped++
			return
		}
		st.Linked++
	})
	return st
}

func matchCredential(n sessions.Node, passwords []Password, credNames map[string]string) (credName, pwID string) {
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
		if label != "" && strings.Contains(pn, label) {
			return name, p.ID
		}
		if host != "" && strings.Contains(pn, host) {
			return name, p.ID
		}
	}
	return "", ""
}

func isUnderFolder(folder, prefix string) bool {
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
