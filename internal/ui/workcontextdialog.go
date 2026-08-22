package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/scottpeterman/pathfinderssh/internal/mspsync"
	"github.com/scottpeterman/pathfinderssh/internal/workcontext"
)

// WorkContextBindOptions configures bind/clear incident work context.
type WorkContextBindOptions struct {
	CustomerNames []string
	OnBind        func(provider, incidentRaw, customer, title, notes string) error
	OnClear       func() error
}

// ShowBindWorkContextDialog binds an incident id or URL to the engineer session.
func ShowBindWorkContextDialog(w fyne.Window, opts WorkContextBindOptions) {
	if w == nil || opts.OnBind == nil {
		return
	}
	provider := widget.NewSelect([]string{workcontext.ProviderPagerDuty, workcontext.ProviderOpsgenie}, nil)
	provider.SetSelected(workcontext.ProviderPagerDuty)
	incident := widget.NewEntry()
	incident.SetPlaceHolder("Incident id or URL (PagerDuty / Opsgenie)")
	picker := NewCustomerFolderPicker(opts.CustomerNames, "")
	title := widget.NewEntry()
	title.SetPlaceHolder("Short title (optional)")
	notes := widget.NewEntry()
	notes.SetPlaceHolder("Engineer notes (optional)")
	body := container.NewVBox(
		widget.NewLabel("Bind active work context for ops desk.\n"+
			"Filters the session tree to the customer and scopes macros/vault."),
		widget.NewLabel("Provider:"),
		provider,
		widget.NewLabel("Incident id or URL:"),
		incident,
		widget.NewLabel("Customer folder:"),
	)
	if len(opts.CustomerNames) > 0 {
		body.Add(picker.Select)
	}
	body.Add(picker.New)
	body.Add(widget.NewLabel("Title:"))
	body.Add(title)
	body.Add(widget.NewLabel("Notes:"))
	body.Add(notes)

	dialog.ShowCustomConfirm("Bind incident", "Bind", "Cancel", body, func(ok bool) {
		if !ok {
			return
		}
		raw := incident.Text
		if raw == "" {
			dialog.ShowInformation("Work context", "Incident id or URL required.", w)
			return
		}
		customer := picker.Chosen()
		if customer == "" {
			customer = mspsync.ResolveCustomerName(opts.CustomerNames, raw)
		}
		if err := opts.OnBind(provider.Selected, raw, customer, title.Text, notes.Text); err != nil {
			dialog.ShowError(err, w)
			return
		}
		dialog.ShowInformation("Work context", "Ops desk active. Session tree filtered to customer.", w)
	}, w)
}

// DocumentWorkOptions posts engineer documentation to an incident system.
type DocumentWorkOptions struct {
	DefaultIncident string
	Provider        string
	OnDocument      func(incidentID, engineerNote string, allTabs, includeMap, includeConfigs bool) (string, error)
}

// ShowDocumentWorkDialog packs evidence and posts to the incident bridge.
func ShowDocumentWorkDialog(w fyne.Window, opts DocumentWorkOptions) {
	if w == nil || opts.OnDocument == nil {
		return
	}
	incident := widget.NewEntry()
	incident.SetText(opts.DefaultIncident)
	incident.SetPlaceHolder("Incident id or URL")
	note := widget.NewEntry()
	note.SetPlaceHolder("What you did (optional)")
	allTabs := widget.NewCheck("Include all open terminal scrollbacks", nil)
	allTabs.SetChecked(true)
	includeMap := widget.NewCheck("Include latest customer topology map", nil)
	includeMap.SetChecked(true)
	includeConfigs := widget.NewCheck("Include running-config captures for touched hosts", nil)
	includeConfigs.SetChecked(true)
	body := container.NewVBox(
		widget.NewLabel("Post-change capture pack: scrollback + map + configs.\n"+
			"A local zip is saved; the incident note summarizes engineer work."),
		incident,
		note,
		allTabs,
		includeMap,
		includeConfigs,
	)
	dialog.ShowCustomConfirm("Document work to incident", "Post note", "Cancel", body, func(ok bool) {
		if !ok {
			return
		}
		go func() {
			msg, err := opts.OnDocument(incident.Text, note.Text, allTabs.Checked, includeMap.Checked, includeConfigs.Checked)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			dialog.ShowInformation("Work documented", msg, w)
		}()
	}, w)
}
