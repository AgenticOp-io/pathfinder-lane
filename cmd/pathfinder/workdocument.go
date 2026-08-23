package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/scottpeterman/pathfinderssh/internal/capturepack"
	"github.com/scottpeterman/pathfinderssh/internal/evidence"
	"github.com/scottpeterman/pathfinderssh/internal/incidentbridge"
	"github.com/scottpeterman/pathfinderssh/internal/opsgenie"
	"github.com/scottpeterman/pathfinderssh/internal/pagerduty"
	"github.com/scottpeterman/pathfinderssh/internal/psaticket"
	"github.com/scottpeterman/pathfinderssh/internal/ui"
	"github.com/scottpeterman/pathfinderssh/internal/workcontext"
)

func (h *host) initWorkContext() {
	h.workCtxPath = workcontext.Path(ui.GetAppHome())
	ctx, err := workcontext.Load(h.workCtxPath)
	if err != nil {
		log.Printf("[workcontext] load: %v", err)
	}
	h.workCtx = ctx
	if h.workContextLabel == nil {
		h.workContextLabel = widget.NewLabel("")
		h.workContextLabel.Importance = widget.MediumImportance
	}
	h.refreshWorkContextLabel()
	if h.workCtx.Active() && h.workCtx.CustomerName != "" {
		h.enterOpsDesk(h.workCtx)
	}
}

func (h *host) refreshWorkContextLabel() {
	if h.workContextLabel != nil {
		h.workContextLabel.SetText(h.workCtx.DisplayLabel())
	}
	if h.appChrome != nil {
		h.appChrome.SetWorkContext(h.workCtx.DisplayLabel())
	}
}

func (h *host) saveWorkContext() {
	if err := workcontext.Save(h.workCtxPath, h.workCtx); err != nil {
		log.Printf("[workcontext] save: %v", err)
	}
	h.refreshWorkContextLabel()
}

func (h *host) incidentBridge(provider string) incidentbridge.Bridge {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case workcontext.ProviderOpsgenie:
		return opsgenie.Bridge{Client: opsgenie.New(h.base.OpsgenieAPIKey, h.base.OpsgenieBaseURL)}
	default:
		return pagerduty.Bridge{Client: pagerduty.New(h.base.PagerDutyAPIKey, h.base.PagerDutyBaseURL)}
	}
}

func (h *host) bindWorkContext() {
	if !h.mspIntegrationsEnabled() {
		dialog.ShowInformation("Work context", "MSP cloud sign-in required.", h.win)
		return
	}
	ui.ShowBindWorkContextDialog(h.win, ui.WorkContextBindOptions{
		CustomerNames: h.mspCustomerNames(),
		OnBind: func(provider, incidentRaw, customer, title, notes string) error {
			incidentID := workcontext.NormalizeIncidentID(incidentRaw)
			if workcontext.IsPSAProvider(provider) {
				info, err := h.lookupPSATicket(provider, incidentRaw)
				if err != nil {
					return err
				}
				if incidentID == "" {
					incidentID = info.ID
				}
				if customer == "" {
					customer = h.resolveMSPCustomer(info.CustomerName)
				}
				if title == "" {
					title = info.Title
				}
			}
			if incidentID == "" {
				return fmt.Errorf("incident or ticket id required")
			}
			h.workCtx = workcontext.Bind(provider, incidentID, customer, title, notes)
			h.saveWorkContext()
			h.enterOpsDesk(h.workCtx)
			return nil
		},
	})
}

func (h *host) clearWorkContext() {
	h.leaveOpsDesk()
	h.workCtx = workcontext.Context{}
	if err := workcontext.Clear(h.workCtxPath); err != nil {
		dialog.ShowError(err, h.win)
		return
	}
	h.refreshWorkContextLabel()
	dialog.ShowInformation("Work context", "Active incident cleared.", h.win)
}

func (h *host) documentWorkToIncident() {
	if !h.mspIntegrationsEnabled() {
		dialog.ShowInformation("Document work", "MSP cloud sign-in required.", h.win)
		return
	}
	defaultInc := h.workCtx.IncidentID
	if defaultInc == "" {
		defaultInc = h.workCtx.IncidentURL
	}
	provider := h.workCtx.Provider
	if provider == "" {
		provider = workcontext.ProviderPagerDuty
	}
	ui.ShowDocumentWorkDialog(h.win, ui.DocumentWorkOptions{
		DefaultIncident: defaultInc,
		Provider:        provider,
		OnDocument:      h.postWorkDocumentation,
	})
}

func (h *host) postWorkDocumentation(incidentRaw, engineerNote string, allTabs, includeMap, includeConfigs bool) (string, error) {
	incidentID := workcontext.NormalizeIncidentID(incidentRaw)
	if incidentID == "" {
		incidentID = strings.TrimSpace(incidentRaw)
	}
	if incidentID == "" {
		return "", fmt.Errorf("incident id or URL required")
	}
	provider := h.workCtx.Provider
	if provider == "" {
		provider = workcontext.ProviderPagerDuty
	}
	if workcontext.IsPSAProvider(provider) {
		return h.postPSAWorkDocumentation(incidentID, engineerNote, allTabs, includeMap, includeConfigs, provider)
	}
	scrollbacks, tabs := h.collectEvidenceFiles(allTabs)
	files, err := capturepack.Collect(capturepack.Options{
		IncidentID:     incidentID,
		Customer:       h.opsDeskCustomer(),
		AppHome:        ui.GetAppHome(),
		StorePath:      h.lastCapture.Params.StorePath,
		LinkedHosts:    h.workCtx.LinkedHosts,
		Scrollbacks:    scrollbacks,
		IncludeMap:     includeMap,
		IncludeConfigs: includeConfigs,
	})
	if err != nil {
		return "", err
	}
	if len(files) == 0 {
		return "", fmt.Errorf("nothing to document — open a terminal or enable map/config capture")
	}
	summary := workcontext.BuildSummary(workcontext.SummaryInput{
		Context:      h.workCtx,
		OpenTabs:     tabs,
		EngineerNote: engineerNote,
	})
	zipBytes, err := evidence.BuildZipBytes(incidentID, files)
	if err != nil {
		return "", err
	}
	name := capturepack.PackName(incidentID)
	localPath := filepath.Join(ui.GetLogsDir(), name)
	if err := evidence.WriteZip(localPath, incidentID, files); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	bridge := h.incidentBridge(provider)
	if err := bridge.Verify(ctx); err != nil {
		return "", fmt.Errorf("%s: %w", provider, err)
	}
	req := incidentbridge.DocumentRequest{
		IncidentID: incidentID,
		Summary:    summary + fmt.Sprintf("\nLocal capture pack: %s\n", localPath),
		FileName:   name,
		FileBytes:  zipBytes,
	}
	if err := incidentbridge.PostDocument(ctx, bridge, req); err != nil {
		return "", err
	}
	if !h.workCtx.Active() || h.workCtx.IncidentID != incidentID {
		h.workCtx = workcontext.Bind(provider, incidentID, h.workCtx.CustomerName, h.workCtx.Title, engineerNote)
	} else {
		h.workCtx.IncidentID = incidentID
		h.workCtx.Provider = provider
	}
	if note := strings.TrimSpace(engineerNote); note != "" {
		h.workCtx.EngineerNotes = note
	}
	h.saveWorkContext()
	return fmt.Sprintf("Posted note to %s alert/incident %s.\nCapture pack: %s", provider, incidentID, localPath), nil
}

func (h *host) postPSAWorkDocumentation(ticketID, engineerNote string, allTabs, includeMap, includeConfigs bool, provider string) (string, error) {
	scrollbacks, tabs := h.collectEvidenceFiles(allTabs)
	files, err := capturepack.Collect(capturepack.Options{
		IncidentID:     ticketID,
		Customer:       h.opsDeskCustomer(),
		AppHome:        ui.GetAppHome(),
		StorePath:      h.lastCapture.Params.StorePath,
		LinkedHosts:    h.workCtx.LinkedHosts,
		Scrollbacks:    scrollbacks,
		IncludeMap:     includeMap,
		IncludeConfigs: includeConfigs,
	})
	if err != nil {
		return "", err
	}
	if len(files) == 0 {
		return "", fmt.Errorf("nothing to document — open a terminal or enable map/config capture")
	}
	summary := workcontext.BuildSummary(workcontext.SummaryInput{
		Context:      h.workCtx,
		OpenTabs:     tabs,
		EngineerNote: engineerNote,
	})
	zipBytes, err := evidence.BuildZipBytes(ticketID, files)
	if err != nil {
		return "", err
	}
	name := capturepack.PackName(ticketID)
	localPath := filepath.Join(ui.GetLogsDir(), name)
	if err := evidence.WriteZip(localPath, ticketID, files); err != nil {
		return "", err
	}
	bridge := h.psaTicketBridge(provider)
	if bridge == nil {
		return "", fmt.Errorf("PSA provider %q not configured", provider)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	req := psaticket.DocumentRequest{
		TicketID:  ticketID,
		Summary:   summary + fmt.Sprintf("\nLocal capture pack: %s\n", localPath),
		FileName:  name,
		FileBytes: zipBytes,
	}
	if err := psaticket.PostDocument(ctx, bridge, req); err != nil {
		return "", err
	}
	if !h.workCtx.Active() || h.workCtx.IncidentID != ticketID {
		h.workCtx = workcontext.Bind(provider, ticketID, h.workCtx.CustomerName, h.workCtx.Title, engineerNote)
	} else {
		h.workCtx.IncidentID = ticketID
		h.workCtx.Provider = provider
	}
	if note := strings.TrimSpace(engineerNote); note != "" {
		h.workCtx.EngineerNotes = note
	}
	h.saveWorkContext()
	return fmt.Sprintf("Posted note to %s ticket %s.\nCapture pack: %s", provider, ticketID, localPath), nil
}

func (h *host) collectEvidenceFiles(allTabs bool) ([]evidence.File, []workcontext.TabInfo) {
	var files []evidence.File
	var tabs []workcontext.TabInfo
	if allTabs {
		for _, inst := range h.shell.Instances() {
			if inst == nil {
				continue
			}
			ta, ok := inst.Applet().(*termApplet)
			if !ok || ta.sess == nil {
				continue
			}
			title := inst.Title()
			text := ta.sess.ScrollbackText()
			if strings.TrimSpace(text) == "" {
				continue
			}
			host := ta.host
			if host != "" {
				h.workCtx.RecordHost(host)
			}
			files = append(files, evidence.File{Name: title + ".txt", Content: []byte(text)})
			tabs = append(tabs, workcontext.TabInfo{Title: title, Host: host})
		}
		return files, tabs
	}
	inst := h.shell.Current()
	if inst == nil {
		return files, tabs
	}
	ta, ok := inst.Applet().(*termApplet)
	if !ok || ta.sess == nil {
		return files, tabs
	}
	title := inst.Title()
	text := ta.sess.ScrollbackText()
	if strings.TrimSpace(text) != "" {
		host := ta.host
		if host != "" {
			h.workCtx.RecordHost(host)
		}
		files = append(files, evidence.File{Name: title + ".txt", Content: []byte(text)})
		tabs = append(tabs, workcontext.TabInfo{Title: title, Host: host})
	}
	return files, tabs
}
