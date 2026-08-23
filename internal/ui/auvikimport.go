package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/scottpeterman/pathfinderssh/internal/auvik"
)

// AuvikImportOptions configures the multi-client import dialog.
type AuvikImportOptions struct {
	Tenants       []auvik.Tenant
	CustomerNames []string
	// OnImport syncs the selected Auvik clients (each into its own Customers/<name>/Auvik folder).
	OnImport func(tenants []auvik.Tenant, opts auvik.ImportOptions) (auvik.ImportStats, error)
}

// ShowAuvikImportDialog lets the operator pick one or more Auvik clients (all selected by default).
func ShowAuvikImportDialog(w fyne.Window, opts AuvikImportOptions) {
	if w == nil {
		return
	}
	if len(opts.Tenants) == 0 {
		dialog.ShowInformation("Auvik", "No Auvik clients returned for this account.", w)
		return
	}

	checks := make([]*widget.Check, len(opts.Tenants))
	rows := make([]fyne.CanvasObject, 0, len(opts.Tenants))
	for i, t := range opts.Tenants {
		label := strings.TrimSpace(t.Name)
		if label == "" {
			label = t.ID
		}
		c := widget.NewCheck(label, nil)
		c.SetChecked(true) // all clients integrated by default
		checks[i] = c
		rows = append(rows, c)
	}

	setAll := func(on bool) {
		for _, c := range checks {
			c.SetChecked(on)
		}
	}
	selectAll := widget.NewButton("Select all", func() { setAll(true) })
	selectNone := widget.NewButton("Select none", func() { setAll(false) })

	list := container.NewVScroll(container.NewVBox(rows...))
	list.SetMinSize(fyne.NewSize(480, 220))

	netOnly := widget.NewCheck("Infra only (switch/router/firewall/AP + server/VM/hypervisor)", nil)
	netOnly.SetChecked(true)
	loginOK := widget.NewCheck("Require Auvik login authorized (skip if status unknown)", nil)
	loginOK.SetChecked(false) // device/info only includes deviceDetail; login status usually absent

	user := widget.NewEntry()
	user.SetPlaceHolder("Default SSH username for new sessions (optional)")
	cred := widget.NewEntry()
	cred.SetPlaceHolder("Default local vault credential name (optional)")

	body := container.NewVBox(
		widget.NewLabel(fmt.Sprintf("%d Auvik client(s). Each selected client syncs into Customers/<client>/Auvik/.", len(opts.Tenants))),
		widget.NewLabel("All clients are selected by default so a full MSP inventory is imported in one pass."),
		container.NewHBox(selectAll, selectNone),
		list,
		netOnly,
		loginOK,
		user,
		cred,
	)

	d := dialog.NewCustomConfirm("Sync from Auvik", "Sync selected", "Cancel", body, func(ok bool) {
		if !ok || opts.OnImport == nil {
			return
		}
		var picked []auvik.Tenant
		for i, c := range checks {
			if c.Checked {
				picked = append(picked, opts.Tenants[i])
			}
		}
		if len(picked) == 0 {
			dialog.ShowInformation("Auvik", "Select at least one client.", w)
			return
		}
		imp := auvik.ImportOptions{
			NetworkGearOnly:        netOnly.Checked,
			RequireLoginAuthorized: loginOK.Checked,
			DefaultUsername:        user.Text,
			DefaultCredential:      cred.Text,
		}
		// Run off the UI thread — multi-client sync can take a while.
		go func() {
			st, err := opts.OnImport(picked, imp)
			fyne.Do(func() {
				if err != nil {
					dialog.ShowError(err, w)
					return
				}
				msg := st.Summary()
				if len(st.Notes) > 0 {
					msg += "\n\n" + strings.Join(st.Notes, "\n")
				}
				if len(st.Errors) > 0 {
					msg += "\n\nErrors:\n" + strings.Join(st.Errors, "\n")
				}
				ShowSyncResultDialog(w, "Auvik sync", msg, st.ErrorCount())
			})
		}()
	}, w)
	d.Resize(fyne.NewSize(560, 520))
	d.Show()
}
