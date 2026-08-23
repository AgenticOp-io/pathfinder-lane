package auvik

import (
	"context"
	"fmt"
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/sessions"
)

// SyncAllOptions configures a full-account sync pass.
type SyncAllOptions struct {
	Client           *Client
	AppHome          string
	Tree             *sessions.Tree
	ImportDefaults   func() ImportOptions
	ResolveCustomer  func(tenant Tenant) string
	MoveToAuvikFolder bool
	UseTunnelDefault  bool
	PruneStale        bool
}

// SyncAllAggregate summarizes every tenant in one pass.
type SyncAllAggregate struct {
	Tenants   int
	Changed   bool
	Stale     int
	Pruned    int
	Summary   string
	Errors    []string
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
	agg := SyncAllAggregate{Tenants: len(tenants)}
	var parts []string
	imp := opts.ImportDefaults()
	for _, tenant := range tenants {
		devs, err := opts.Client.ListDevices(ctx, []string{tenant.ID}, 300)
		if err != nil {
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
	if len(parts) == 0 {
		agg.Summary = fmt.Sprintf("no changes across %d client(s)", agg.Tenants)
	} else {
		agg.Summary = strings.Join(parts, "\n")
	}
	return agg, nil
}
