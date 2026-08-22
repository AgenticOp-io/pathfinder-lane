package main

import (
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/sessions"
	"github.com/scottpeterman/pathfinderssh/internal/vault"
	"github.com/scottpeterman/pathfinderssh/internal/workcontext"
)

func (h *host) opsDeskCustomer() string {
	if c := strings.TrimSpace(h.workCtx.CustomerName); c != "" {
		return c
	}
	return strings.TrimSpace(h.opsDeskCustomerName)
}

func (h *host) enterOpsDesk(ctx workcontext.Context) {
	customer := strings.TrimSpace(ctx.CustomerName)
	if customer == "" {
		return
	}
	h.opsDeskCustomerName = customer
	h.mapCustomer = customer
	if h.tree != nil {
		if h.treeFilterBackup == "" {
			h.treeFilterBackup = h.tree.FilterText()
		}
		h.tree.SetFilterText(customer)
	}
	h.refreshVault()
	h.reloadBarButtons()
}

func (h *host) leaveOpsDesk() {
	h.opsDeskCustomerName = ""
	if h.tree != nil && h.treeFilterBackup != "" {
		h.tree.SetFilterText(h.treeFilterBackup)
		h.treeFilterBackup = ""
	} else if h.tree != nil {
		h.tree.ClearFilter()
	}
	h.refreshVault()
	h.reloadBarButtons()
}

func (h *host) reloadBarButtons() {
	if h.shell == nil {
		return
	}
	h.appChrome = nil
	h.buildChrome()
}

func (h *host) vaultCustomerScoped() bool {
	if h.base.VaultBreakGlass {
		return false
	}
	return h.opsDeskCustomer() != ""
}

func (h *host) credentialAllowed(c vault.Credential) bool {
	if !h.vaultCustomerScoped() {
		return true
	}
	tag := sessions.CustomerTag(h.opsDeskCustomer())
	if tag == "" {
		return true
	}
	return c.HasTag(tag) || c.HasTag("break-glass")
}
