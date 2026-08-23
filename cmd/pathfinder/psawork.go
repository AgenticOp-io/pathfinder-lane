package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/scottpeterman/pathfinderssh/internal/autotask"
	"github.com/scottpeterman/pathfinderssh/internal/connectwise"
	"github.com/scottpeterman/pathfinderssh/internal/evidence"
	"github.com/scottpeterman/pathfinderssh/internal/handoff"
	"github.com/scottpeterman/pathfinderssh/internal/halo"
	"github.com/scottpeterman/pathfinderssh/internal/mapweb"
	"github.com/scottpeterman/pathfinderssh/internal/psaticket"
	"github.com/scottpeterman/pathfinderssh/internal/ui"
	"github.com/scottpeterman/pathfinderssh/internal/workcontext"
)

func (h *host) psaTicketBridge(provider string) psaticket.Bridge {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case workcontext.ProviderConnectWise:
		return connectwise.TicketBridge{Client: connectwise.New(
			h.base.ConnectWiseCompanyID,
			h.base.ConnectWisePublicKey,
			h.base.ConnectWisePrivateKey,
			h.base.ConnectWiseClientID,
			h.base.ConnectWiseBaseURL,
		)}
	case workcontext.ProviderAutotask:
		return autotask.TicketBridge{Client: autotask.New(
			h.base.AutotaskUsername,
			h.base.AutotaskSecret,
			h.base.AutotaskAPIIntegrationCode,
			h.base.AutotaskBaseURL,
		)}
	case workcontext.ProviderHalo:
		return halo.TicketBridge{Client: halo.New(
			h.base.HaloClientID,
			h.base.HaloClientSecret,
			h.base.HaloTenant,
			h.base.HaloBaseURL,
		)}
	default:
		return nil
	}
}

func (h *host) lookupPSATicket(provider, raw string) (psaticket.TicketInfo, error) {
	bridge := h.psaTicketBridge(provider)
	if bridge == nil {
		return psaticket.TicketInfo{}, fmt.Errorf("PSA provider %q not configured", provider)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return bridge.LookupTicket(ctx, raw)
}

func (h *host) exportCustomerHandoff() {
	if !h.mspIntegrationsEnabled() {
		dialog.ShowInformation("Customer handoff", "MSP cloud sign-in required.", h.win)
		return
	}
	picker := ui.NewCustomerFolderPicker(h.mspCustomerNames(), h.opsDeskCustomer())
	body := container.NewVBox(
		widget.NewLabel("Export sessions YAML, topology maps, and inventory metadata\n"+
			"for MSP customer handoff (no vault secrets)."),
		widget.NewLabel("Customer folder:"),
	)
	if len(h.mspCustomerNames()) > 0 {
		body.Add(picker.Select)
	}
	body.Add(picker.New)
	dialog.ShowCustomConfirm("Export customer handoff", "Export…", "Cancel", body, func(ok bool) {
		if !ok {
			return
		}
		customer := picker.Chosen()
		if customer == "" {
			dialog.ShowInformation("Customer handoff", "Choose a customer folder.", h.win)
			return
		}
		files, err := handoff.Build(handoff.Options{
			Customer: customer,
			AppHome:  ui.GetAppHome(),
			Tree:     h.tree.Tree(),
		})
		if err != nil {
			dialog.ShowError(err, h.win)
			return
		}
		name := handoff.PackName(customer)
		save := dialog.NewFileSave(func(uc fyne.URIWriteCloser, err error) {
			if err != nil {
				dialog.ShowError(err, h.win)
				return
			}
			if uc == nil {
				return
			}
			path := uc.URI().Path()
			_ = uc.Close()
			if err := evidence.WriteZip(path, customer, files); err != nil {
				dialog.ShowError(err, h.win)
				return
			}
			dialog.ShowInformation("Customer handoff", fmt.Sprintf("Saved %s (%d files).", path, len(files)), h.win)
		}, h.win)
		save.SetFileName(name)
		save.Show()
	}, h.win)
}

func (h *host) openNOCMapView() {
	customer := strings.TrimSpace(h.opsDeskCustomer())
	if customer == "" {
		picker := ui.NewCustomerFolderPicker(h.mspCustomerNames(), "")
		body := container.NewVBox(
			widget.NewLabel("Open the latest topology map for NOC / wallboard viewing.\n"+
				"Use Reload in the map viewer (or F5) to refresh after crawls."),
			widget.NewLabel("Customer:"),
		)
		if len(h.mspCustomerNames()) > 0 {
			body.Add(picker.Select)
		}
		body.Add(picker.New)
		dialog.ShowCustomConfirm("NOC map view", "Open map", "Cancel", body, func(ok bool) {
			if !ok {
				return
			}
			c := picker.Chosen()
			if c == "" {
				dialog.ShowInformation("NOC map", "Choose a customer.", h.win)
				return
			}
			h.openLatestCustomerMap(c)
		}, h.win)
		return
	}
	h.openLatestCustomerMap(customer)
}

func (h *host) openLatestCustomerMap(customer string) {
	dir := ui.CustomerMapsDir(ui.GetAppHome(), customer)
	path, err := latestJSONInDir(dir)
	if err != nil {
		dialog.ShowError(err, h.win)
		return
	}
	if path == "" {
		dialog.ShowInformation("NOC map",
			fmt.Sprintf("No map JSON in %s.\nRun a crawl or import a topology map first.", dir), h.win)
		return
	}
	h.mapDir = dir
	h.mapCustomer = customer
	h.openMap(mapweb.MapFile{Name: filepath.Base(path), Path: path})
	dialog.ShowInformation("NOC map",
		"Map opened in your browser. Use Reload in the viewer to refresh.\n"+
			"Fullscreen the browser tab for wallboard mode.", h.win)
}

func latestJSONInDir(dir string) (string, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	var best string
	var bestTime time.Time
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
			continue
		}
		full := filepath.Join(dir, e.Name())
		info, err := e.Info()
		if err != nil {
			continue
		}
		if best == "" || info.ModTime().After(bestTime) {
			best = full
			bestTime = info.ModTime()
		}
	}
	return best, nil
}
