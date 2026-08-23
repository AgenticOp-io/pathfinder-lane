package auvik

import (
	"context"
	"fmt"
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/sessions"
)

// SyncAllOptions configures a full-account sync pass.
type SyncAllOptions struct {
	Client            *Client
	AppHome           string
	Tree              *sessions.Tree
	ImportDefaults    func() ImportOptions
	ResolveCustomer   func(tenant Tenant) string
	MoveToAuvikFolder bool
	UseTunnelDefault  bool
	PruneStale        bool
}

// SyncAllAggregate summarizes every tenant in one pass.
type SyncAllAggregate struct {
	Tenants int
	Changed bool
	Stale   int
	Pruned  int
	Skipped int // tenants skipped (permission deny list)
	Summary string
	Errors  []string
}

// SyncAllTenants syncs every Auvik tenant into the session tree.
func SyncAllTenants(ctx context.Context, opts SyncAllOptions) (SyncAllAggregate, error) {
	if opts.Client == nil {
		return SyncAllAggregate{}, fmt.Errorf("auvik client required")
	}
	if opts.Tree == nil {
		return SyncAllAggregate{}, fmt.Errorf("session tree required")
	}
	if err := opts.Client.Verify(ctx); err != nil {
		return SyncAllAggregate{}, err
	}
	tenants, err := opts.Client.ListTenants(ctx)
	if err != nil {
		return SyncAllAggregate{}, err
	}
	var skipMap TenantMap
	if home := strings.TrimSpace(opts.AppHome); home != "" {
		if m, err := LoadTenantMap(home); err == nil {
			skipMap = m
		}
	}
	agg := SyncAllAggregate{Tenants: len(tenants)}
	var parts []string
	imp := opts.ImportDefaults()
	for _, tenant := range tenants {
		if skipMap.ShouldSkipSync(tenant.ID) {
			agg.Skipped++
			continue
		}
		devs, err := opts.Client.ListDevices(ctx, []string{tenant.ID}, 300)
		if err != nil {
			if IsPermissionDenied(err) {
				skipMap.MarkSkipSync(tenant.ID, err.Error())
				if home := strings.TrimSpace(opts.AppHome); home != "" {
					_ = SaveTenantMap(home, skipMap)
				}
				agg.Skipped++
				// Log once via summary note — not every 5 minutes as a hard error.
				parts = append(parts, fmt.Sprintf("%s: skipped (no API permission — will not retry)", tenant.Name))
				continue
			}
			parts = append(parts, tenant.Name+": "+err.Error())
			agg.Errors = append(agg.Errors, tenant.Name+": "+err.Error())
			continue
		}
		customer := tenant.Name
		if opts.ResolveCustomer != nil {
			customer = opts.ResolveCustomer(tenant)
		}
		res := SyncTenantTree(opts.Tree, devs, SyncOptions{
			ImportOptions:     imp,
			Tenant:            tenant,
			CustomerName:      customer,
			MoveToAuvikFolder: opts.MoveToAuvikFolder,
			UseTunnelDefault:  opts.UseTunnelDefault,
		})
		if res.Changed() {
			agg.Changed = true
			parts = append(parts, tenant.Name+": "+res.Summary())
		}
		if len(res.Errors) > 0 {
			msg := tenant.Name + " errors: " + strings.Join(res.Errors, "; ")
			parts = append(parts, msg)
			agg.Errors = append(agg.Errors, msg)
		}
		if strings.TrimSpace(opts.AppHome) != "" {
			if m, err := LoadTenantMap(opts.AppHome); err == nil {
				m.SetDomain(tenant.ID, tenant.Name)
				_ = SaveTenantMap(opts.AppHome, m)
			}
		}
		if opts.PruneStale {
			stale := CollectStale(*opts.Tree, customer, sessions.JoinPath(sessions.DefaultCustomersRoot, customer, ImportFolder), DeviceIDSet(devs))
			agg.Stale += len(stale)
			if len(stale) > 0 {
				pruned := PruneStale(opts.Tree, stale)
				agg.Pruned += pruned
				if pruned > 0 {
					agg.Changed = true
					parts = append(parts, fmt.Sprintf("%s: pruned %d stale Auvik session(s)", tenant.Name, pruned))
				} else if len(stale) > 0 {
					parts = append(parts, fmt.Sprintf("%s: %d stale session(s) in tree (not pruned)", tenant.Name, len(stale)))
				}
			}
		}
	}
	if agg.Skipped > 0 {
		parts = append(parts, fmt.Sprintf("(%d tenant(s) skipped — no API permission)", agg.Skipped))
	}
	if len(parts) == 0 {
		agg.Summary = fmt.Sprintf("no changes across %d client(s)", agg.Tenants)
	} else {
		agg.Summary = strings.Join(parts, "\n")
	}
	return agg, nil
}
