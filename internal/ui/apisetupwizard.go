package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ShowAPISetupWizard edits MSP integrations and Cursor AI (standalone / solo setup).
func ShowAPISetupWizard(w fyne.Window) {
	if w == nil {
		return
	}

	panel := newMSPIntegrationPanel()
	cursor := NewCursorConfigForm()
	base, err := LoadSettings(SettingsPath())
	if err != nil {
		base = Defaults()
	}
	fields := SettingsFieldsOf(base)
	panel.load(fields)
	cursor.load(fields)

	hero := installWizardHero("Standalone MSP setup", "Integrations (Auvik, PSA, vault…) and Cursor AI — no cloud sign-in required")

	intScroll := container.NewScroll(panel.content())
	intScroll.SetMinSize(fyne.NewSize(680, 380))
	cursorScroll := container.NewScroll(cursor.Content())
	cursorScroll.SetMinSize(fyne.NewSize(680, 160))

	saveBtn := widget.NewButtonWithIcon("Save settings", theme.DocumentSaveIcon(), nil)
	saveBtn.Importance = widget.HighImportance
	closeBtn := widget.NewButton("Close", func() { fyne.CurrentApp().Quit() })

	saveBtn.OnTapped = func() {
		merged := panel.fields(fields)
		merged = cursor.apply(merged)
		updated, errs := merged.Apply(base)
		if len(errs) > 0 {
			dialog.ShowError(errs[0], w)
			return
		}
		if err := SaveSettings(SettingsPath(), updated); err != nil {
			dialog.ShowError(err, w)
			return
		}
		SetSettings(updated)
		dialog.ShowInformation("Saved",
			"MSP integrations and Cursor settings saved to\n"+SettingsPath()+"\n\n"+
				"Sync actions run from Pathfinder File menu after launch.", w)
	}

	body := container.NewBorder(
		container.NewVBox(
			hero,
			widget.NewLabel("Optional: leave blank any service you do not use. Solo and standalone installs use the same MSP stack as cloud-enrolled orgs."),
			widget.NewLabelWithStyle("Integrations", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		),
		container.NewHBox(closeBtn, saveBtn),
		nil, nil,
		container.NewVBox(intScroll, widget.NewSeparator(), cursorScroll),
	)
	w.SetContent(container.NewPadded(body))
}
