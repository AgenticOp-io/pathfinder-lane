package ui

import (
	"fmt"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const unassignedCustomer = "(no customer)"

// regroupCustomerTabs reorders docked tabs so terminals are grouped by
// customer with a non-session header tab above each group. Non-terminal tabs
// stay after the terminal groups. No-op while tiled.
//
// DocTabs has no native section API; header TabItems are markers whose
// selection jumps to the first real tab in that group. Closing a header
// asks before terminating every terminal in that customer group.
func (s *Shell) regroupCustomerTabs() {
	if s == nil || s.tabs == nil || s.tiled {
		return
	}
	if s.regrouping {
		return
	}
	s.regrouping = true
	defer func() { s.regrouping = false }()

	selected := s.tabs.Selected()
	wantInst := s.instanceFor(selected)

	type group struct {
		name string
		tabs []*container.TabItem
	}
	byCust := map[string]*group{}
	var order []string
	var others []*container.TabItem

	for _, inst := range s.instances() {
		if inst == nil || inst.closed.Load() || inst.tab == nil || inst.win != nil {
			continue
		}
		if inst.mount.Kind != KindTerminal {
			others = append(others, inst.tab)
			continue
		}
		name := terminalCustomerName(inst)
		g := byCust[name]
		if g == nil {
			g = &group{name: name}
			byCust[name] = g
			order = append(order, name)
		}
		g.tabs = append(g.tabs, inst.tab)
	}

	sort.SliceStable(order, func(i, j int) bool {
		a, b := order[i], order[j]
		if a == unassignedCustomer {
			return false
		}
		if b == unassignedCustomer {
			return true
		}
		return strings.ToLower(a) < strings.ToLower(b)
	})
	for _, name := range order {
		g := byCust[name]
		sort.SliceStable(g.tabs, func(i, j int) bool {
			ia, ja := s.instanceFor(g.tabs[i]), s.instanceFor(g.tabs[j])
			if ia == nil || ja == nil {
				return false
			}
			return strings.ToLower(ia.Title()) < strings.ToLower(ja.Title())
		})
	}

	// Drop previous header tabs; their content is disposable.
	s.customerHeaders = map[*container.TabItem]string{}

	items := make([]*container.TabItem, 0, len(order)*2+len(others))
	showHeaders := len(order) > 0
	for _, name := range order {
		g := byCust[name]
		if showHeaders {
			label := " " + name + " "
			hdr := container.NewTabItemWithIcon(label, theme.FolderIcon(), widget.NewLabel(""))
			s.customerHeaders[hdr] = name
			items = append(items, hdr)
		}
		items = append(items, g.tabs...)
	}
	items = append(items, others...)

	s.tabs.Items = items
	s.tabs.Refresh()

	// Restore selection on a real instance tab when possible.
	if wantInst != nil && wantInst.tab != nil && !wantInst.closed.Load() {
		s.tabs.Select(wantInst.tab)
		return
	}
	if len(items) == 0 {
		return
	}
	// Prefer first non-header tab.
	for _, it := range items {
		if _, isHdr := s.customerHeaders[it]; !isHdr {
			s.tabs.Select(it)
			return
		}
	}
	s.tabs.Select(items[0])
}

func terminalCustomerName(inst *Instance) string {
	if inst == nil {
		return unassignedCustomer
	}
	meta, ok := inst.mount.Applet.(TerminalMeta)
	if !ok {
		return unassignedCustomer
	}
	name := strings.TrimSpace(meta.CustomerName())
	if name == "" {
		return unassignedCustomer
	}
	return name
}

func (s *Shell) isCustomerHeader(t *container.TabItem) bool {
	if s == nil || t == nil {
		return false
	}
	_, ok := s.customerHeaders[t]
	return ok
}

// confirmCloseCustomerGroup asks before ending every open terminal for a
// customer (the X on that customer's folder header in the tab strip).
func (s *Shell) confirmCloseCustomerGroup(customer string) {
	if s == nil {
		return
	}
	customer = strings.TrimSpace(customer)
	if customer == "" {
		return
	}
	toClose := s.terminalsForCustomer(customer)
	if len(toClose) == 0 {
		return
	}
	label := customer
	if label == unassignedCustomer {
		label = "sessions with no customer"
	}
	title := "Close all terminal sessions?"
	msg := fmt.Sprintf("Do you want to close all terminal sessions for %s?\n\n%d session(s) will be terminated.", label, len(toClose))
	dialog.ShowConfirm(title, msg, func(ok bool) {
		if !ok {
			return
		}
		for _, inst := range toClose {
			if inst != nil && !inst.closed.Load() {
				inst.Close()
			}
		}
	}, s.win)
}

// terminalsForCustomer returns every open terminal (docked or detached) for
// the given customer group name, including the unassigned bucket.
func (s *Shell) terminalsForCustomer(customer string) []*Instance {
	var out []*Instance
	for _, inst := range s.instances() {
		if inst == nil || inst.closed.Load() || inst.mount.Kind != KindTerminal {
			continue
		}
		if terminalCustomerName(inst) == customer {
			out = append(out, inst)
		}
	}
	return out
}

// selectFirstInCustomer selects the first docked terminal tab for customer.
func (s *Shell) selectFirstInCustomer(customer string) {
	if s == nil {
		return
	}
	for _, inst := range s.instances() {
		if inst == nil || inst.tab == nil || inst.win != nil || inst.mount.Kind != KindTerminal {
			continue
		}
		if terminalCustomerName(inst) == customer {
			s.tabs.Select(inst.tab)
			return
		}
	}
}

// dockedTerminalsByCustomer returns docked terminal instances grouped and
// sorted for tile layout (same order as the tab strip).
func (s *Shell) dockedTerminalsByCustomer() []*Instance {
	type pair struct {
		cust string
		inst *Instance
	}
	var list []pair
	for _, inst := range s.instances() {
		if inst == nil || inst.closed.Load() || inst.mount.Kind != KindTerminal || inst.win != nil {
			continue
		}
		list = append(list, pair{cust: terminalCustomerName(inst), inst: inst})
	}
	sort.SliceStable(list, func(i, j int) bool {
		a, b := list[i], list[j]
		if a.cust != b.cust {
			if a.cust == unassignedCustomer {
				return false
			}
			if b.cust == unassignedCustomer {
				return true
			}
			if strings.ToLower(a.cust) != strings.ToLower(b.cust) {
				return strings.ToLower(a.cust) < strings.ToLower(b.cust)
			}
		}
		return strings.ToLower(a.inst.Title()) < strings.ToLower(b.inst.Title())
	})
	out := make([]*Instance, len(list))
	for i, p := range list {
		out[i] = p.inst
	}
	return out
}

// terminalTabTitle is the session label under a customer header (no prefix).
// In tile mode the customer is included so panes stay identifiable.
func terminalTabTitle(inst *Instance, includeCustomer bool) string {
	if inst == nil {
		return ""
	}
	title := inst.Title()
	if !includeCustomer {
		return title
	}
	cust := terminalCustomerName(inst)
	if cust == "" || cust == unassignedCustomer {
		return title
	}
	return cust + " · " + title
}

// Ensure fyne.Resource use stays local to this file's icon call.
var _ fyne.Resource = theme.FolderIcon()
