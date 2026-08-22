// Package psasync is the PSA/RMM → Customers sync scaffold.
//
// ConnectWise / Autotask / Halo / Ninja adapters plug in here. The engine
// does not ship vendor SDKs; each MSP wires credentials via env or a local
// config file outside the vault.
package psasync

import (
	"context"
	"fmt"
)

// Customer is one MSP customer record from a PSA or RMM.
type Customer struct {
	ExternalID string
	Name       string
	Tags       []string
}

// Source pulls customers from an external system.
type Source interface {
	Name() string
	ListCustomers(ctx context.Context) ([]Customer, error)
}

// Result is what a sync pass did to the local Customers tree.
type Result struct {
	Source   string
	Created  []string
	Existing []string
	Errors   []string
}

// SyncFolderNames ensures each customer name exists under customersRoot
// using createFn(name) → path. createFn should be idempotent (create-or-get).
func SyncFolderNames(customersRoot string, customers []Customer, createFn func(name string) (path string, err error)) Result {
	r := Result{}
	for _, c := range customers {
		name := c.Name
		if name == "" {
			name = c.ExternalID
		}
		if name == "" {
			r.Errors = append(r.Errors, "empty customer name")
			continue
		}
		path, err := createFn(name)
		if err != nil {
			r.Errors = append(r.Errors, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		if path != "" {
			r.Created = append(r.Created, name)
		} else {
			r.Existing = append(r.Existing, name)
		}
	}
	return r
}

// StubSource returns demo customers for UI wiring without a live PSA.
type StubSource struct{}

func (StubSource) Name() string { return "stub" }

func (StubSource) ListCustomers(ctx context.Context) ([]Customer, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	return []Customer{
		{ExternalID: "demo-1", Name: "Demo Customer A", Tags: []string{"demo"}},
		{ExternalID: "demo-2", Name: "Demo Customer B", Tags: []string{"demo"}},
	}, nil
}
