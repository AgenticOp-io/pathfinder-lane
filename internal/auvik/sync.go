package auvik

import (
	"fmt"
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/mspsync"
	"github.com/scottpeterman/pathfinderssh/internal/product"
	"github.com/scottpeterman/pathfinderssh/internal/sessions"
)

// SyncOptions configures an authority sync into the session tree.
type SyncOptions struct {
	ImportOptions
	Tenant            Tenant
	CustomerName      string // folder under Customers; defaults to Tenant.Name
	MoveToAuvikFolder bool   // relocate matched sessions into Customers/<client>/Auvik
	UseTunnelDefault  bool   // set AuvikUseTunnel on new/linked nodes
}

func (o SyncOptions) withDefaults() SyncOptions {
	o.ImportOptions = o.ImportOptions.withDefaults()
	if strings.TrimSpace(o.CustomerName) == "" {
		o.CustomerName = strings.TrimSpace(o.Tenant.Name)
	}
	if o.CustomerName == "" {
		o.CustomerName = o.Tenant.ID
	}
	return o
}

// SyncResult summarizes one tenant sync pass.
type SyncResult struct {
	Created  int
	Updated  int
	Merged   int
	Moved    int
	Skipped  int
	NoIP     int
	Errors   []string
}

func (s SyncResult) Summary() string {
	return fmt.Sprintf("created %d, updated %d, merged %d, moved %d, skipped %d, no IP %d",
		s.Created, s.Updated, s.Merged, s.Moved, s.Skipped, s.NoIP)
}

func (s SyncResult) Changed() bool {
	return s.Created+s.Updated+s.Merged+s.Moved > 0
}

type sessionLoc struct {
	folder string
	label  string
}

type customerIndex struct {
	byAuvikID map[string]sessionLoc
	byHost    map[string]sessionLoc
	byName    map[string]sessionLoc
}

// SyncTenantTree merges Auvik inventory into Customers/<client>/Auvik/.
//
// Existing sessions under the same customer are matched by Auvik device id,
// management IP, or display name. Auvik updates Host when addresses change;
// username, credentials, and terminal settings on matched nodes are preserved.
func SyncTenantTree(t *sessions.Tree, devices []Device, opts SyncOptions) SyncResult {
	opts = opts.withDefaults()
	res := SyncResult{}

	root := product.CustomersRoot
	customer := opts.CustomerName
	if customer == "" {
		res.Errors = append(res.Errors, "customer name required")
		return res
	}
	customer = mspsync.ResolveCustomerName(t.ListCustomers(root), customer)

	if _, err := t.CreateCustomer(root, customer); err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			res.Errors = append(res.Errors, err.Error())
			return res
		}
	}

	auvikFolder := sessions.JoinPath(root, customer, ImportFolder)
	if _, err := t.EnsurePath(auvikFolder); err != nil {
		res.Errors = append(res.Errors, err.Error())
		return res
	}

	idx := indexCustomerSessions(*t, root, customer)

	for _, d := range devices {
		if !opts.WantDevice(d) {
			res.Skipped++
			continue
		}
		host := d.PrimaryIP()
		if host == "" {
			res.NoIP++
			continue
		}

		want := nodeFromDevice(d, opts)
		loc, how := findMatch(idx, d, want)
		if loc.folder == "" {
			if err := addWithDisambiguation(t, auvikFolder, want); err != nil {
				res.Errors = append(res.Errors, want.Label()+": "+err.Error())
				continue
			}
			res.Created++
			registerIndex(&idx, auvikFolder, want)
			continue
		}

		existing, err := sessionAt(t, loc.folder, loc.label)
		if err != nil {
			res.Errors = append(res.Errors, want.Label()+": "+err.Error())
			continue
		}

		merged, changed := mergeAuvikAuthority(existing, want, opts.ImportOptions)
		if how == "ip" || how == "name" {
			res.Merged++
		}
		if changed {
			res.Updated++
		}

		targetFolder := loc.folder
		if opts.MoveToAuvikFolder && !folderEqual(targetFolder, auvikFolder) {
			if err := t.Move(loc.folder, loc.label, auvikFolder); err != nil {
				res.Errors = append(res.Errors, want.Label()+": move: "+err.Error())
			} else {
				res.Moved++
				targetFolder = auvikFolder
				loc.label = merged.Label()
			}
		}

		if err := t.Replace(targetFolder, loc.label, merged); err != nil {
			res.Errors = append(res.Errors, merged.Label()+": "+err.Error())
			continue
		}
		registerIndex(&idx, targetFolder, merged)
	}

	return res
}

func nodeFromDevice(d Device, opts SyncOptions) sessions.Node {
	nodes, _ := SessionNodes([]Device{d}, opts.ImportOptions)
	if len(nodes) == 0 {
		return sessions.Node{}
	}
	n := nodes[0]
	n.AuvikDeviceID = strings.TrimSpace(d.ID)
	n.AuvikTenantID = strings.TrimSpace(d.TenantID)
	n.AuvikDomain = strings.TrimSpace(opts.Tenant.Name)
	n.IntegrationSource = mspsync.SourceAuvik
	n.ExternalDeviceID = strings.TrimSpace(d.ID)
	if opts.UseTunnelDefault {
		n.AuvikUseTunnel = true
	}
	n.Vendor = strings.TrimSpace(d.Vendor)
	n.DeviceType = strings.TrimSpace(d.DeviceType)
	return n.Normalize()
}

func mergeAuvikAuthority(existing, from sessions.Node, imp ImportOptions) (sessions.Node, bool) {
	out := existing.Normalize()
	changed := false

	if from.Host != "" && !strings.EqualFold(strings.TrimSpace(out.Host), strings.TrimSpace(from.Host)) {
		out.Host = from.Host
		changed = true
	}
	if strings.TrimSpace(out.Name) == "" && strings.TrimSpace(from.Name) != "" {
		out.Name = from.Name
		changed = true
	}
	if from.AuvikDeviceID != "" && out.AuvikDeviceID != from.AuvikDeviceID {
		out.AuvikDeviceID = from.AuvikDeviceID
		changed = true
	}
	if from.ExternalDeviceID != "" && out.ExternalDeviceID != from.ExternalDeviceID {
		out.ExternalDeviceID = from.ExternalDeviceID
		changed = true
	}
	if from.IntegrationSource != "" && out.IntegrationSource != from.IntegrationSource {
		out.IntegrationSource = from.IntegrationSource
		changed = true
	}
	if from.AuvikTenantID != "" && out.AuvikTenantID != from.AuvikTenantID {
		out.AuvikTenantID = from.AuvikTenantID
		changed = true
	}
	if from.AuvikDomain != "" && out.AuvikDomain != from.AuvikDomain {
		out.AuvikDomain = from.AuvikDomain
		changed = true
	}
	if from.Vendor != "" && out.Vendor != from.Vendor {
		out.Vendor = from.Vendor
		changed = true
	}
	if from.DeviceType != "" && out.DeviceType != from.DeviceType {
		out.DeviceType = from.DeviceType
		changed = true
	}
	if from.AuvikUseTunnel && !out.AuvikUseTunnel {
		out.AuvikUseTunnel = true
		changed = true
	}

	if imp.DefaultUsername != "" && strings.TrimSpace(out.Username) == "" {
		out.Username = strings.TrimSpace(imp.DefaultUsername)
		changed = true
	}
	if imp.DefaultCredential != "" && strings.TrimSpace(out.Credential) == "" {
		out.Credential = strings.TrimSpace(imp.DefaultCredential)
		out.AuthType = sessions.AuthPassword
		changed = true
	}

	return out.Normalize(), changed
}

func indexCustomerSessions(t sessions.Tree, root, customer string) customerIndex {
	idx := customerIndex{
		byAuvikID: map[string]sessionLoc{},
		byHost:    map[string]sessionLoc{},
		byName:    map[string]sessionLoc{},
	}
	prefix := sessions.CustomerPath(root, customer)
	t.WalkSessions(func(folder string, n sessions.Node) {
		if !isUnderFolder(folder, prefix) {
			return
		}
		registerIndex(&idx, folder, n)
	})
	return idx
}

func registerIndex(idx *customerIndex, folder string, n sessions.Node) {
	loc := sessionLoc{folder: folder, label: n.Label()}
	if id := strings.TrimSpace(n.AuvikDeviceID); id != "" {
		idx.byAuvikID[mspsync.DeviceIDKey(mspsync.SourceAuvik, id)] = loc
	}
	if id := strings.TrimSpace(n.ExternalDeviceID); id != "" {
		src := strings.TrimSpace(n.IntegrationSource)
		if src == "" {
			src = mspsync.SourceAuvik
		}
		idx.byAuvikID[mspsync.DeviceIDKey(src, id)] = loc
	}
	if h := strings.ToLower(strings.TrimSpace(n.Host)); h != "" {
		idx.byHost[h] = loc
	}
	if name := strings.ToLower(strings.TrimSpace(n.Name)); name != "" {
		idx.byName[name] = loc
	}
	if label := strings.ToLower(n.Label()); label != "" {
		idx.byName[label] = loc
	}
}

func findMatch(idx customerIndex, d Device, want sessions.Node) (sessionLoc, string) {
	if id := strings.TrimSpace(d.ID); id != "" {
		key := mspsync.DeviceIDKey(mspsync.SourceAuvik, id)
		if loc, ok := idx.byAuvikID[key]; ok {
			return loc, "id"
		}
	}
	for _, ip := range d.IPs {
		ip = strings.ToLower(strings.TrimSpace(ip))
		if ip == "" {
			continue
		}
		if loc, ok := idx.byHost[ip]; ok {
			return loc, "ip"
		}
	}
	for _, key := range []string{
		strings.ToLower(strings.TrimSpace(d.Name)),
		strings.ToLower(want.Label()),
	} {
		if key == "" {
			continue
		}
		if loc, ok := idx.byName[key]; ok {
			return loc, "name"
		}
	}
	return sessionLoc{}, ""
}

func sessionAt(t *sessions.Tree, folder, label string) (sessions.Node, error) {
	f, err := t.FolderAt(folder)
	if err != nil {
		return sessions.Node{}, err
	}
	j := f.SessionIndex(label)
	if j < 0 {
		return sessions.Node{}, fmt.Errorf("no session %q in %q", label, folder)
	}
	return f.Sessions[j], nil
}

func addWithDisambiguation(t *sessions.Tree, folder string, n sessions.Node) error {
	if err := t.Add(folder, n); err != nil {
		if strings.Contains(err.Error(), "already has") {
			n.Name = n.Label() + " (" + n.Host + ")"
			return t.Add(folder, n)
		}
		return err
	}
	return nil
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

func folderEqual(a, b string) bool {
	return sessions.JoinPath(sessions.SplitPath(a)...) == sessions.JoinPath(sessions.SplitPath(b)...)
}
