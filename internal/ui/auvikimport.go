package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/scottpeterman/pathfinderssh/internal/auvik"
)

// AuvikImportOptions configures the import dialog.
type AuvikImportOptions struct {
	Tenants       []auvik.Tenant
	CustomerNames []string
	// OnImport runs with selected tenant IDs, customer folder, and import options.
	OnImport func(tenantIDs []string, customerFolder string, opts auvik.ImportOptions) (auvik.ImportStats, error)
}

// ShowAuvikImportDialog picks Auvik client(s) and imports SSH inventory.
func ShowAuvikImportDialog(w fyne.Window, opts AuvikImportOptions) {
	if w == nil {
		return
	}
	if len(opts.Tenants) == 0 {
		dialog.ShowInformation("Auvik", "No Auvik clients returned for this account.", w)
		return
	}
	labels := make([]string, len(opts.Tenants))
	labelToID := map[string]string{}
	for i, t := range opts.Tenants {
		labels[i] = t.Name
		if t.ID != "" {
			labelToID[t.Name] = t.ID
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
	netOnly := widget.NewCheck("Network gear only (switch/router/firewall/AP…)", nil)
	netOnly.SetChecked(true)
	loginOK := widget.NewCheck("Require Auvik login authorized", nil)
	loginOK.SetChecked(true)
	user := widget.NewEntry()
	user.SetPlaceHolder("Default SSH username (optional)")
	cred := widget.NewEntry()
	cred.SetPlaceHolder("Vault credential name (optional)")

	body := container.NewVBox(
		widget.NewLabel("Import devices into Customers/<client>/Auvik/ as SSH sessions.\n"+
			"Existing sessions under the same customer are merged; inventory updates IPs on sync.\n"+
			"Pair with IT Glue, Hudu, or Passportal for credentials after import."),
		sel,
		widget.NewLabel("Customer folder under Customers/:"),
	)
	if len(opts.CustomerNames) > 0 {
		body.Add(picker.Select)
	}
	body.Add(picker.New)
	body.Add(netOnly)
	body.Add(loginOK)
	body.Add(user)
	body.Add(cred)

	d := dialog.NewCustomConfirm("Import from Auvik", "Import", "Cancel", body, func(ok bool) {
		if !ok || opts.OnImport == nil {
			return
		}
		id := labelToID[sel.Selected]
		if id == "" {
			dialog.ShowInformation("Auvik", "Pick a client.", w)
			return
		}
		customer := picker.Chosen()
		if customer == "" {
			dialog.ShowInformation("Auvik", "Customer folder name required.", w)
			return
		}
		imp := auvik.ImportOptions{
			NetworkGearOnly:        netOnly.Checked,
			RequireLoginAuthorized: loginOK.Checked,
			DefaultUsername:        user.Text,
			DefaultCredential:      cred.Text,
		}
		st, err := opts.OnImport([]string{id}, customer, imp)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		msg := st.Summary()
		if len(st.Errors) > 0 {
			msg += "\nErrors: " + fmt.Sprintf("%v", st.Errors)
		}
		dialog.ShowInformation("Auvik import", msg, w)
	}, w)
	d.Resize(fyne.NewSize(520, 420))
	d.Show()
}
