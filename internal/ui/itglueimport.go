package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/scottpeterman/pathfinderssh/internal/itglue"
)

// ITGlueImportOptions configures the import dialog.
type ITGlueImportOptions struct {
	Organizations []itglue.Organization
	CustomerNames []string
	OnImport      func(orgID string, opts itglue.ImportDialogOptions) (itglue.ImportResult, error)
}

// ShowITGlueImportDialog imports passwords from IT Glue into the vault and links sessions.
func ShowITGlueImportDialog(w fyne.Window, opts ITGlueImportOptions) {
	if w == nil {
		return
	}
	if len(opts.Organizations) == 0 {
		dialog.ShowInformation("IT Glue", "No organizations returned for this API key.", w)
		return
	}
	labels := make([]string, len(opts.Organizations))
	labelToID := map[string]string{}
	for i, o := range opts.Organizations {
		labels[i] = o.Name
		if o.ID != "" {
			labelToID[o.Name] = o.ID
		}
	}
	sel := widget.NewSelect(labels, nil)
	if len(labels) > 0 {
		sel.SetSelected(labels[0])
	}
	picker := NewCustomerFolderPicker(opts.CustomerNames, labels[0])
	sel.OnChanged = func(label string) {
		picker.SetSuggested(opts.CustomerNames, label)
	}
	updateVault := widget.NewCheck("Update existing IT Glue credentials in vault", nil)
	updateVault.SetChecked(true)
	linkSessions := widget.NewCheck("Link vault credentials to SSH sessions under this customer", nil)
	linkSessions.SetChecked(true)
	onlyEmpty := widget.NewCheck("Only sessions without a credential", nil)
	onlyEmpty.SetChecked(true)
	sshOnly := widget.NewCheck("Import SSH / network passwords only (skip blank names)", nil)
	sshOnly.SetChecked(true)

	body := container.NewVBox(
		widget.NewLabel("Import passwords from IT Glue into your encrypted vault.\n"+
			"API key must have Password Access enabled. Secrets stay in the vault — not logged.\n"+
			"Pair with any inventory source: sync devices first, then link credentials here."),
		sel,
		widget.NewLabel("Session tree folder for linking:"),
	)
	if len(opts.CustomerNames) > 0 {
		body.Add(picker.Select)
	}
	body.Add(picker.New)
	body.Add(updateVault)
	body.Add(linkSessions)
	body.Add(onlyEmpty)
	body.Add(sshOnly)

	d := dialog.NewCustomConfirm("Import from IT Glue", "Import", "Cancel", body, func(ok bool) {
		if !ok || opts.OnImport == nil {
			return
		}
		id := labelToID[sel.Selected]
		if id == "" {
			dialog.ShowInformation("IT Glue", "Pick an organization.", w)
			return
		}
		imp := itglue.ImportDialogOptions{
			UpdateVault:    updateVault.Checked,
			LinkSessions:   linkSessions.Checked,
			OnlyEmptyCreds: onlyEmpty.Checked,
			SSHFilter:      sshOnly.Checked,
			CustomerName:   picker.Chosen(),
		}
		st, err := opts.OnImport(id, imp)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		msg := st.Summary()
		if len(st.Errors) > 0 {
			msg += "\nErrors: " + fmt.Sprintf("%v", st.Errors)
		}
		dialog.ShowInformation("IT Glue import", msg, w)
	}, w)
	d.Resize(fyne.NewSize(560, 440))
	d.Show()
}
