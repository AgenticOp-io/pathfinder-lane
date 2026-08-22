package invsync

import (
	"fmt"
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/mspsync"
	"github.com/scottpeterman/pathfinderssh/internal/product"
	"github.com/scottpeterman/pathfinderssh/internal/sessions"
)

// Options configures a generic inventory sync into Customers/<client>/<folder>/.
type Options struct {
	CustomerName   string
	ImportFolder   string // e.g. "Domotz", "Ninja"
	IntegrationSrc string // IntegrationSource on session nodes
	DefaultUser    string
	DefaultCred    string
}

// Result summarizes one sync pass.
type Result struct {
	Created  int
	Updated  int
	Merged   int
	Moved    int
	Skipped  int
	NoIP     int
	Errors   []string
}

func (r Result) Summary() string {
	return fmt.Sprintf("created %d, updated %d, merged %d, moved %d, skipped %d, no IP %d",
		r.Created, r.Updated, r.Merged, r.Moved, r.Skipped, r.NoIP)
}

type sessionLoc struct {
	folder string
	label  string
}

type customerIndex struct {
	byExtID map[string]sessionLoc
	byHost  map[string]sessionLoc
	byName  map[string]sessionLoc
}

// SyncCustomerTree merges devices into Customers/<client>/<ImportFolder>/.
func SyncCustomerTree(t *sessions.Tree, devices []Device, opts Options) Result {
	res := Result{}
	opts.CustomerName = strings.TrimSpace(opts.CustomerName)
	opts.ImportFolder = strings.TrimSpace(opts.ImportFolder)
	opts.IntegrationSrc = strings.TrimSpace(opts.IntegrationSrc)
	if opts.CustomerName == "" {
		res.Errors = append(res.Errors, "customer name required")
		return res
	}
	root := product.CustomersRoot
	opts.CustomerName = mspsync.ResolveCustomerName(t.ListCustomers(root), opts.CustomerName)
	if opts.ImportFolder == "" {
		res.Errors = append(res.Errors, "import folder required")
		return res
	}

	if _, err := t.CreateCustomer(root, opts.CustomerName); err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			res.Errors = append(res.Errors, err.Error())
			return res
		}
	}
	targetFolder := sessions.JoinPath(root, opts.CustomerName, opts.ImportFolder)
	if _, err := t.EnsurePath(targetFolder); err != nil {
		res.Errors = append(res.Errors, err.Error())
		return res
	}

	idx := indexCustomerSessions(*t, root, opts.CustomerName)

	for _, d := range devices {
		host := strings.TrimSpace(d.Host)
		if host == "" {
			res.NoIP++
			continue
		}
		want := nodeFromDevice(d, opts)
		loc, how := findMatch(idx, d, want, opts.IntegrationSrc)
		if loc.folder == "" {
			if err := addWithDisambiguation(t, targetFolder, want); err != nil {
				res.Errors = append(res.Errors, want.Label()+": "+err.Error())
				continue
			}
			res.Created++
			registerIndex(&idx, targetFolder, want)
			continue
		}
		existing, err := sessionAt(t, loc.folder, loc.label)
		if err != nil {
			res.Errors = append(res.Errors, want.Label()+": "+err.Error())
			continue
		}
		merged, changed := mergeAuthority(existing, want, opts)
		if how == "ip" || how == "name" {
			res.Merged++
		}
		if changed {
			res.Updated++
		}
		target := loc.folder
		if !folderEqual(target, targetFolder) {
			if err := t.Move(loc.folder, loc.label, targetFolder); err != nil {
				res.Errors = append(res.Errors, want.Label()+": move: "+err.Error())
			} else {
				res.Moved++
				target = targetFolder
				loc.label = merged.Label()
			}
		}
		if err := t.Replace(target, loc.label, merged); err != nil {
			res.Errors = append(res.Errors, merged.Label()+": "+err.Error())
			continue
		}
		registerIndex(&idx, target, merged)
	}
	return res
}

func nodeFromDevice(d Device, opts Options) sessions.Node {
	name := strings.TrimSpace(d.Name)
	if name == "" {
		name = d.Host
	}
	n := sessions.Node{
		Name:              name,
		Transport:         sessions.TransportSSH,
		Host:              strings.TrimSpace(d.Host),
		Port:              sessions.TransportSSH.DefaultPort(),
		AuthType:          sessions.AuthAgent,
		HostKeyPolicy:     sessions.HostKeyTOFU,
		Vendor:            strings.TrimSpace(d.Vendor),
		DeviceType:        strings.TrimSpace(d.DeviceType),
		IntegrationSource: opts.IntegrationSrc,
		ExternalDeviceID:  strings.TrimSpace(d.ID),
	}
	if u := strings.TrimSpace(opts.DefaultUser); u != "" {
		n.Username = u
	}
	if c := strings.TrimSpace(opts.DefaultCred); c != "" {
		n.Credential = c
		n.AuthType = sessions.AuthPassword
	}
	return n.Normalize()
}

func mergeAuthority(existing, from sessions.Node, opts Options) (sessions.Node, bool) {
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
	if from.ExternalDeviceID != "" && out.ExternalDeviceID != from.ExternalDeviceID {
		out.ExternalDeviceID = from.ExternalDeviceID
		changed = true
	}
	if from.IntegrationSource != "" && out.IntegrationSource != from.IntegrationSource {
		out.IntegrationSource = from.IntegrationSource
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
	if opts.DefaultUser != "" && strings.TrimSpace(out.Username) == "" {
		out.Username = strings.TrimSpace(opts.DefaultUser)
		changed = true
	}
	if opts.DefaultCred != "" && strings.TrimSpace(out.Credential) == "" {
		out.Credential = strings.TrimSpace(opts.DefaultCred)
		out.AuthType = sessions.AuthPassword
		changed = true
	}
	return out, changed
}

func indexCustomerSessions(t sessions.Tree, root, customer string) customerIndex {
	idx := customerIndex{
		byExtID: map[string]sessionLoc{},
		byHost:  map[string]sessionLoc{},
		byName:  map[string]sessionLoc{},
	}
	prefix := sessions.CustomerPath(root, customer)
	t.WalkSessions(func(folder string, n sessions.Node) {
		if !isUnderFolder(folder, prefix) {
			return
		}
		n = n.Normalize()
		loc := sessionLoc{folder: folder, label: n.Label()}
		if idKey := deviceIDKey(n); idKey != "" {
			idx.byExtID[idKey] = loc
		}
		if h := strings.ToLower(strings.TrimSpace(n.Host)); h != "" {
			idx.byHost[h] = loc
		}
		if name := strings.ToLower(strings.TrimSpace(n.Label())); name != "" {
			idx.byName[name] = loc
		}
	})
	return idx
}

func findMatch(idx customerIndex, d Device, want sessions.Node, source string) (sessionLoc, string) {
	if key := mspsync.DeviceIDKey(source, d.ID); key != "" {
		if loc, ok := idx.byExtID[key]; ok {
			return loc, "id"
		}
	}
	if h := strings.ToLower(strings.TrimSpace(want.Host)); h != "" {
		if loc, ok := idx.byHost[h]; ok {
			return loc, "ip"
		}
	}
	if name := strings.ToLower(strings.TrimSpace(want.Label())); name != "" {
		if loc, ok := idx.byName[name]; ok {
			return loc, "name"
		}
	}
	return sessionLoc{}, ""
}

func registerIndex(idx *customerIndex, folder string, n sessions.Node) {
	n = n.Normalize()
	loc := sessionLoc{folder: folder, label: n.Label()}
	if idKey := deviceIDKey(n); idKey != "" {
		idx.byExtID[idKey] = loc
	}
	if h := strings.ToLower(strings.TrimSpace(n.Host)); h != "" {
		idx.byHost[h] = loc
	}
	if name := strings.ToLower(strings.TrimSpace(n.Label())); name != "" {
		idx.byName[name] = loc
	}
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

func folderEqual(a, b string) bool {
	return strings.TrimSpace(a) == strings.TrimSpace(b)
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

func deviceIDKey(n sessions.Node) string {
	n = n.Normalize()
	if id := strings.TrimSpace(n.ExternalDeviceID); id != "" {
		return mspsync.DeviceIDKey(n.IntegrationSource, id)
	}
	if id := strings.TrimSpace(n.AuvikDeviceID); id != "" {
		return mspsync.DeviceIDKey(SourceAuvik, id)
	}
	return ""
}
