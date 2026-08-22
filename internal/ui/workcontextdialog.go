package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/scottpeterman/pathfinderssh/internal/mspsync"
)

// WorkContextBindOptions configures bind/clear incident work context.
type WorkContextBindOptions struct {
	CustomerNames []string
	OnBind        func(incidentRaw, customer, title, notes string) error
	OnClear       func() error
}

// ShowBindWorkContextDialog binds an incident id or URL to the engineer session.
func ShowBindWorkContextDialog(w fyne.Window, opts WorkContextBindOptions) {
	if w == nil || opts.OnBind == nil {
		return
	}
	incident := widget.NewEntry()
	incident.SetPlaceHolder("PagerDuty incident id or URL")
	picker := NewCustomerFolderPicker(opts.CustomerNames, "")
	title := widget.NewEntry()
	title.SetPlaceHolder("Short title (optional)")
	notes := widget.NewEntry()
	notes.SetPlaceHolder("Engineer notes (optional)")
	body := container.NewVBox(
		widget.NewLabel("Bind active work context for documentation.\n"+
			"PSA/RMM stay in their apps — this augments PagerDuty with engineer notes."),
		widget.NewLabel("Incident id or URL:"),
		incident,
		widget.NewLabel("Customer folder (optional):"),
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
		if err := opts.OnBind(raw, customer, title.Text, notes.Text); err != nil {
			dialog.ShowError(err, w)
			return
		}
		dialog.ShowInformation("Work context", "Incident bound. Status bar shows active work.", w)
	}, w)
}

// DocumentWorkOptions posts engineer documentation to an incident system.
type DocumentWorkOptions struct {
	DefaultIncident string
	OnDocument      func(incidentID, engineerNote string, allTabs bool) (string, error)
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
	body := container.NewVBox(
		widget.NewLabel("Document engineer work to PagerDuty as an incident note.\n"+
			"A local evidence zip is saved; the note includes a summary and file details."),
		incident,
		note,
		allTabs,
	)
	dialog.ShowCustomConfirm("Document work to incident", "Post note", "Cancel", body, func(ok bool) {
		if !ok {
			return
		}
		go func() {
			msg, err := opts.OnDocument(incident.Text, note.Text, allTabs.Checked)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			dialog.ShowInformation("Work documented", msg, w)
		}()
	}, w)
}
