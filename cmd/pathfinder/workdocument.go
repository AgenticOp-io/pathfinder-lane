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

	"github.com/scottpeterman/pathfinderssh/internal/evidence"
	"github.com/scottpeterman/pathfinderssh/internal/incidentbridge"
	"github.com/scottpeterman/pathfinderssh/internal/pagerduty"
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

func (h *host) pagerDutyBridge() incidentbridge.Bridge {
	return pagerduty.Bridge{Client: pagerduty.New(h.base.PagerDutyAPIKey, h.base.PagerDutyBaseURL)}
}

func (h *host) bindWorkContext() {
	if !h.mspIntegrationsEnabled() {
		dialog.ShowInformation("Work context", "MSP cloud sign-in required.", h.win)
		return
	}
	ui.ShowBindWorkContextDialog(h.win, ui.WorkContextBindOptions{
		CustomerNames: h.mspCustomerNames(),
		OnBind: func(incidentRaw, customer, title, notes string) error {
			h.workCtx = workcontext.Bind(workcontext.ProviderPagerDuty, incidentRaw, customer, title, notes)
			h.saveWorkContext()
			return nil
		},
	})
}

func (h *host) clearWorkContext() {
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
	ui.ShowDocumentWorkDialog(h.win, ui.DocumentWorkOptions{
		DefaultIncident: defaultInc,
		OnDocument:      h.postWorkDocumentation,
	})
}

func (h *host) postWorkDocumentation(incidentRaw, engineerNote string, allTabs bool) (string, error) {
	incidentID := workcontext.NormalizeIncidentID(incidentRaw)
	if incidentID == "" {
		return "", fmt.Errorf("incident id or URL required")
	}
	files, tabs := h.collectEvidenceFiles(allTabs)
	if len(files) == 0 {
		return "", fmt.Errorf("no scrollback to document — open a terminal first")
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
	name := fmt.Sprintf("pathfinder-evidence-%s-%s.zip", sanitizeFilePart(incidentID), time.Now().Format("20060102-150405"))
	localPath := filepath.Join(ui.GetLogsDir(), name)
	if err := evidence.WriteZip(localPath, incidentID, files); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	bridge := h.pagerDutyBridge()
	if err := bridge.Verify(ctx); err != nil {
		return "", fmt.Errorf("pagerduty: %w", err)
	}
	req := incidentbridge.DocumentRequest{
		IncidentID: incidentID,
		Summary:    summary + fmt.Sprintf("\nLocal evidence zip: %s\n", localPath),
		FileName:   name,
		FileBytes:  zipBytes,
	}
	if err := incidentbridge.PostDocument(ctx, bridge, req); err != nil {
		return "", err
	}
	if !h.workCtx.Active() || h.workCtx.IncidentID != incidentID {
		h.workCtx = workcontext.Bind(workcontext.ProviderPagerDuty, incidentID, h.workCtx.CustomerName, h.workCtx.Title, engineerNote)
	} else {
		h.workCtx.IncidentID = incidentID
	}
	if note := strings.TrimSpace(engineerNote); note != "" {
		h.workCtx.EngineerNotes = note
	}
	h.saveWorkContext()
	return fmt.Sprintf("Posted note to PagerDuty incident %s.\nEvidence zip: %s", incidentID, localPath), nil
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

func sanitizeFilePart(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, s)
}
