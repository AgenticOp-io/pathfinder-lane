// SecureCRT import wizard: pick which top-level CRT folder is the customer list.
package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/scottpeterman/pathfinderssh/internal/sessions"
)

// CRTImportChoice is what the wizard returns.
type CRTImportChoice struct {
	CustomerRoot string // CRT top-level folder name, or "" for none
	Replace      bool   // wipe inventory before import
}

// ShowCRTImportWizard asks which SecureCRT folder holds the customer list.
// topLevel are CRT top-level folder names from the pending import.
func ShowCRTImportWizard(w fyne.Window, configPath string, supported, skipped int, topLevel []string, onOK func(CRTImportChoice)) {
	hint := widget.NewLabel(fmt.Sprintf(
		"Found %d importable sessions in:\n%s\n\n%d unsupported (RDP/etc.) will be skipped. Passwords are never imported.\n\nPick the SecureCRT folder that is your customer list. Its subfolders become Customers/<name> with their nested folders kept. Everything else goes under Unassigned (folders kept).",
		supported, configPath, skipped))
	hint.Wrapping = fyne.TextWrapWord

	options := []string{"(None — put everything under Unassigned)"}
	defaultIdx := 0
	for i, name := range topLevel {
		options = append(options, name)
		low := strings.ToLower(name)
		if name == sessions.LegacyCRTCustomersRoot || strings.Contains(low, "customer") {
			defaultIdx = i + 1
		}
	}
	sel := widget.NewSelect(options, nil)
	if defaultIdx < len(options) {
		sel.SetSelected(options[defaultIdx])
	} else if len(options) > 0 {
		sel.SetSelected(options[0])
	}

	replace := widget.NewCheck("Replace current inventory (recommended after a bad import)", nil)
	replace.SetChecked(true)

	form := container.NewVBox(
		hint,
		widget.NewLabel("Customer list folder"),
		sel,
		replace,
	)

	d := dialog.NewCustomConfirm("Import SecureCRT", "Import", "Cancel", form, func(ok bool) {
		if !ok || onOK == nil {
			return
		}
		choice := CRTImportChoice{Replace: replace.Checked}
		picked := sel.Selected
		if picked != "" && !strings.HasPrefix(picked, "(None") {
			choice.CustomerRoot = picked
		}
		onOK(choice)
	}, w)
	d.Resize(fyne.NewSize(640, 480))
	d.Show()
}
