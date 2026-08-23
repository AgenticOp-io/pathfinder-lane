package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/scottpeterman/pathfinderssh/internal/auvik"
	"github.com/scottpeterman/pathfinderssh/internal/mspsync"
	"github.com/scottpeterman/pathfinderssh/internal/sessions"
	"github.com/scottpeterman/pathfinderssh/internal/ui"
)

func cmdSyncAuvik(args []string) error {
	fs := flag.NewFlagSet("sync-auvik", flag.ContinueOnError)
	sessionsPath := fs.String("sessions", "", "sessions.yaml path")
	settingsPath := fs.String("settings", "", "settings.json path")
	prune := fs.Bool("prune", false, "remove stale Auvik sessions missing from inventory")
	dry := fs.Bool("dry-run", false, "sync without writing sessions file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	sp := *sessionsPath
	if sp == "" {
		sp = defaultSessionsPath()
	}
	stPath := *settingsPath
	if stPath == "" {
		stPath = ui.SettingsPath()
	}
	base, err := ui.LoadSettings(stPath)
	if err != nil {
		return err
	}
	tr, err := sessions.LoadFile(sp)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	customers := tr.ListCustomers(sessions.DefaultCustomersRoot)
	resolve := func(name string) string {
		return mspsync.ResolveCustomerName(customers, name)
	}
	agg, err := auvik.SyncAllTenants(ctx, auvik.SyncAllOptions{
		Client:            auvik.New(base.AuvikUsername, base.AuvikAPIKey, base.AuvikBaseURL),
		AppHome:           appHome(),
		Tree:              &tr,
		ImportDefaults: func() auvik.ImportOptions {
			user := strings.TrimSpace(base.MSPInventoryDefUsername)
			if user == "" {
				user = strings.TrimSpace(base.AuvikDefaultUsername)
			}
			cred := strings.TrimSpace(base.MSPInventoryDefCredential)
			if cred == "" {
				cred = strings.TrimSpace(base.AuvikDefaultCredential)
			}
			return auvik.ImportOptions{
				NetworkGearOnly:        true,
				RequireLoginAuthorized: true,
				DefaultUsername:        user,
				DefaultCredential:      cred,
			}
		},
		ResolveCustomer: func(tenant auvik.Tenant) string {
			return auvik.ResolveCustomer(appHome(), tenant.ID, tenant.Name, customers, resolve)
		},
		MoveToAuvikFolder: true,
		UseTunnelDefault:  base.AuvikAutoTunnel,
		PruneStale:        *prune || base.AuvikPruneStale,
	})
	if err != nil {
		return err
	}
	fmt.Println(agg.Summary)
	if agg.Changed && !*dry {
		if err := sessions.SaveFile(sp, tr); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "saved %s\n", sp)
	}
	return nil
}
