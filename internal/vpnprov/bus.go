package vpnprov

import (
	"context"
	"strings"
	"sync"

	"github.com/scottpeterman/pathfinderssh/internal/fortivpn"
)

// Bins are optional overrides. Empty fields use each vendor's default path.
type Bins struct {
	FortiBin   string
	FortiTools string
	WireGuard  string
	Zscaler    string
}

// Item is one tunnel the installer can map to a CRT folder.
type Item struct {
	Kind  string
	Name  string
	Label string // value stored in Folder=Label
}

var switchMu sync.Mutex

// ListAll returns FortiClient, WireGuard, and Zscaler ZPA entries that
// are installed on this PC. Missing vendors are omitted, not errors.
func ListAll(ctx context.Context, b Bins) []Item {
	var out []Item
	seen := map[string]bool{}
	add := func(kind, name string) {
		t := Target{Kind: kind, Name: name}
		label := FormatTarget(t)
		if label == "" || seen[strings.ToLower(label)] {
			return
		}
		seen[strings.ToLower(label)] = true
		out = append(out, Item{Kind: kind, Name: name, Label: label})
	}

	forti, err := fortivpn.ListTunnels(ctx, b.FortiBin)
	if err == nil {
		for _, n := range forti {
			add(FortiClient, n)
		}
	}
	for _, n := range listWireGuard(ctx) {
		add(WireGuard, n)
	}
	for _, n := range listZscaler(b.Zscaler) {
		add(Zscaler, n)
	}
	return out
}

// Ensure brings the mapped target up and tears down other full-tunnel
// customer VPNs first (FortiClient, other WireGuard tunnels, ZPA).
// It never disables ZIA. MFA still happens in the vendor UI.
func Ensure(ctx context.Context, b Bins, raw string) error {
	t := ParseTarget(raw)
	if t.Name == "" {
		return nil
	}
	switchMu.Lock()
	defer switchMu.Unlock()

	exclusiveOff(ctx, b, t)
	switch t.Kind {
	case WireGuard:
		return startWireGuard(ctx, b.WireGuard, t.Name)
	case Zscaler:
		return enableZscaler(ctx, b.Zscaler, t.Name)
	default:
		return fortivpn.Ensure(ctx, b.FortiBin, b.FortiTools, t.Name)
	}
}

func exclusiveOff(ctx context.Context, b Bins, keep Target) {
	if keep.Kind != WireGuard {
		stopOtherWireGuard(ctx, "")
	} else {
		stopOtherWireGuard(ctx, keep.Name)
	}
	if keep.Kind != Zscaler {
		_ = disableZPA(ctx, b.Zscaler)
	}
	if keep.Kind != FortiClient {
		_ = fortivpn.DisconnectActive(ctx, b.FortiBin, b.FortiTools)
	}
}
