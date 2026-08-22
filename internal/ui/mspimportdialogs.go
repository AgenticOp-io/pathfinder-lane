package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// InventorySyncOptions configures a generic RMM inventory import.
type InventorySyncOptions struct {
	Title           string
	Help            string
	CustomerNames   []string
	SuggestedCustomer string
	OnSync          func(customerName string) (string, error)
}

// ShowInventorySyncDialog picks a customer folder and runs inventory sync.
func ShowInventorySyncDialog(w fyne.Window, opts InventorySyncOptions) {
	if w == nil || opts.OnSync == nil {
		return
	}
	picker := NewCustomerFolderPicker(opts.CustomerNames, opts.SuggestedCustomer)
	body := container.NewVBox(
		widget.NewLabel(opts.Help),
		widget.NewLabel("Customer folder under Customers/:"),
	)
	if len(opts.CustomerNames) > 0 {
		body.Add(picker.Select)
	}
	body.Add(picker.New)

	dialog.ShowCustomConfirm(opts.Title, "Sync", "Cancel", body, func(ok bool) {
		if !ok {
			return
		}
		name := picker.Chosen()
		if name == "" {
			dialog.ShowInformation(opts.Title, "Customer name required.", w)
			return
		}
		go func() {
			msg, err := opts.OnSync(name)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			dialog.ShowInformation(opts.Title, msg, w)
		}()
	}, w)
}

// DocVaultImportOptions configures a documentation-platform credential import.
type DocVaultImportOptions struct {
	Title         string
	Organizations []string
	CustomerNames []string
	OnImport      func(orgLabel string, customerFolder string, linkSessions bool) (string, error)
}

// ShowDocVaultImportDialog imports passwords into vault and optionally links sessions.
func ShowDocVaultImportDialog(w fyne.Window, opts DocVaultImportOptions) {
	if w == nil || opts.OnImport == nil {
		return
	}
	if len(opts.Organizations) == 0 {
		dialog.ShowInformation(opts.Title, "No organizations returned for this API key.", w)
		return
	}
	sel := widget.NewSelect(opts.Organizations, nil)
	sel.SetSelected(opts.Organizations[0])
	picker := NewCustomerFolderPicker(opts.CustomerNames, opts.Organizations[0])
	sel.OnChanged = func(label string) {
		picker.SetSuggested(opts.CustomerNames, label)
	}
	link := widget.NewCheck("Link vault credentials to SSH sessions under this customer", nil)
	link.SetChecked(true)
	body := container.NewVBox(
		widget.NewLabel("Import passwords into your encrypted local vault.\n"+
			"Plaintext exists only in memory during import."),
		sel,
		widget.NewLabel("Session tree folder for linking:"),
	)
	if len(opts.CustomerNames) > 0 {
		body.Add(picker.Select)
	}
	body.Add(picker.New)
	body.Add(link)

	dialog.ShowCustomConfirm(opts.Title, "Import", "Cancel", body, func(ok bool) {
		if !ok {
			return
		}
		label := sel.Selected
		customer := picker.Chosen()
		go func() {
			msg, err := opts.OnImport(label, customer, link.Checked)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			dialog.ShowInformation(opts.Title, msg, w)
		}()
	}, w)
}

// PassportalImportOptions imports Passportal passwords (no org list API).
type PassportalImportOptions struct {
	CustomerNames     []string
	SuggestedCustomer string
	OnImport          func(customerName string, linkSessions bool) (string, error)
}

// ShowPassportalImportDialog imports Passportal credentials with customer folder for linking.
func ShowPassportalImportDialog(w fyne.Window, opts PassportalImportOptions) {
	if w == nil || opts.OnImport == nil {
		return
	}
	picker := NewCustomerFolderPicker(opts.CustomerNames, opts.SuggestedCustomer)
	link := widget.NewCheck("Link vault credentials to SSH sessions under this customer", nil)
	link.SetChecked(true)
	body := container.NewVBox(
		widget.NewLabel("Import Passportal passwords into the encrypted vault."),
		widget.NewLabel("Customer folder for session linking:"),
	)
	if len(opts.CustomerNames) > 0 {
		body.Add(picker.Select)
	}
	body.Add(picker.New)
	body.Add(link)

	dialog.ShowCustomConfirm("Passportal", "Import", "Cancel", body, func(ok bool) {
		if !ok {
			return
		}
		go func() {
			msg, err := opts.OnImport(picker.Chosen(), link.Checked)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			dialog.ShowInformation("Passportal", msg, w)
		}()
	}, w)
}
