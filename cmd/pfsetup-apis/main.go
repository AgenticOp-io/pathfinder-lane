// pfsetup-apis — deprecated. Integrations live in Pathfinder Settings → Tools.
package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/scottpeterman/pathfinderssh/internal/ui"
)

func main() {
	a := app.NewWithID("com.pathfinder.pfsetup.apis")
	ui.LoadUserThemes()
	ui.ApplyInstallerTheme(a)
	if icon := ui.AppIcon(); icon != nil {
		a.SetIcon(icon)
	}
	w := a.NewWindow("PathfinderSSH — integrations moved")
	w.Resize(fyne.NewSize(520, 280))
	w.CenterOnScreen()
	w.SetContent(container.NewPadded(container.NewVBox(
		widget.NewLabelWithStyle("API setup is in Settings", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Auvik, PSA, vault, Cursor AI, and other integrations are configured in Pathfinder:"),
		widget.NewLabel("Settings → Tools"),
		widget.NewLabel("This helper is no longer used. For MSP org setup (branding, sign-in, engineer packs), run pfsetup-msp.exe."),
		widget.NewButton("Close", func() { a.Quit() }),
	)))
	w.ShowAndRun()
}
