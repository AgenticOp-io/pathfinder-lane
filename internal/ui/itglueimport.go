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
			"Pair with Auvik: import devices from Auvik, then link credentials from IT Glue."),
		sel,
		updateVault,
		linkSessions,
		onlyEmpty,
		sshOnly,
	)

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
			CustomerName:   sel.Selected,
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
	d.Resize(fyne.NewSize(560, 400))
	d.Show()
}
