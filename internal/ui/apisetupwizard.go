package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// ShowAPISetupWizard is deprecated — integrations live in Settings → Tools.
func ShowAPISetupWizard(w fyne.Window) {
	if w == nil {
		return
	}
	w.SetContent(container.NewPadded(container.NewVBox(
		widget.NewLabelWithStyle("API setup moved to Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Configure Auvik, PSA, vault, Cursor AI, and other integrations in Pathfinder:"),
		widget.NewLabel("Settings → Tools"),
		widget.NewLabel("MSP org setup (branding, sign-in, engineer packs): pfsetup-msp.exe"),
		widget.NewButton("Close", func() { fyne.CurrentApp().Quit() }),
	)))
}
