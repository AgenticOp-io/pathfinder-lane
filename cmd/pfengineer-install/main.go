// pfengineer-install — engineer standalone installer (pre-configured MSP, no admin setup).
package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"github.com/scottpeterman/pathfinderssh/internal/ui"
)

var version = "dev"

func main() {
	a := app.NewWithID("com.pathfinder.pfengineer.install")
	ui.LoadUserThemes()
	ui.ApplyInstallerTheme(a)
	if icon := ui.AppIcon(); icon != nil {
		a.SetIcon(icon)
	}
	w := a.NewWindow("Pathfinder engineer install")
	w.Resize(fyne.NewSize(720, 580))
	w.CenterOnScreen()

	ui.ShowEngineerInstallWizard(w, ui.EngineerInstallOptions{Version: version})
	w.ShowAndRun()
}
