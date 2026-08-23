package main

import (
	"fmt"
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/auvik"
	"github.com/scottpeterman/pathfinderssh/internal/jump"
	"github.com/scottpeterman/pathfinderssh/internal/sessions"
	"github.com/scottpeterman/pathfinderssh/internal/ui"
)

func connectStatusLabel(folder string, n sessions.Node, tr sessions.Tree) string {
	home := ui.GetAppHome()
	n = auvik.EnrichTunnelDomain(home, folder, n, tr)
	// Tunnel-first sessions never TCP-probe the private device IP; say so up front
	// so the dialog does not look stuck on "Checking reachability".
	if auvik.ShouldUseTunnelFirst(n, home) {
		domain := auvik.ResolveTunnelDomain(home, n)
		msg := "Opening Auvik tunnel to " + n.Target()
		if domain != "" {
			msg += " (" + domain + ")"
		}
		return msg + " …"
	}
	msg := "Checking reachability of " + n.Target()
	if n.Jump.InUse() {
		msg += fmt.Sprintf("\nJump hop: %s@%s", n.Jump.Username, n.Jump.Host)
	}
	if path := jump.DefaultConfigPath(); path != "" {
		if cfg, ok, err := jump.LoadConfig(path); err == nil && ok {
			if res, err := jump.NewResolver(cfg, nil); err == nil {
				d := jump.Device{Name: n.Name, Addr: n.Host, Platform: n.Vendor}
				dec := res.Resolve(d)
				if !dec.Path.IsDirect() {
					msg += "\nRoute map: " + dec.Path.String()
				}
			}
		}
	}
	if fb := strings.TrimSpace(n.ConsoleFallback); fb != "" {
		msg += "\nConsole fallback: " + fb
	}
	return msg + " …"
}
