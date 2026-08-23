// pfsetup-apis — standalone wizard for all MSP integration API credentials.
package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"github.com/scottpeterman/pathfinderssh/internal/ui"
)

func main() {
	a := app.NewWithID("com.pathfinder.pfsetup.apis")
	ui.LoadUserThemes()
	ui.ApplyInstallerTheme(a)
	if icon := ui.AppIcon(); icon != nil {
		a.SetIcon(icon)
	}
	w := a.NewWindow("PathfinderSSH — standalone MSP setup")
	w.Resize(fyne.NewSize(760, 640))
	w.CenterOnScreen()

	ui.ShowAPISetupWizard(w)
	w.ShowAndRun()
}
