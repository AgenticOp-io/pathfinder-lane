// pfsetup-o365 â€” Microsoft 365 organization master setup (separate from main installer).
package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"github.com/scottpeterman/pathfinderssh/internal/idp"
	"github.com/scottpeterman/pathfinderssh/internal/mspauth"
	"github.com/scottpeterman/pathfinderssh/internal/ui"
)

func main() {
	a := app.NewWithID("com.pathfinder.pfsetup.o365")
	ui.LoadUserThemes()
	ui.ApplyInstallerTheme(a)
	if icon := ui.AppIcon(); icon != nil {
		a.SetIcon(icon)
	}
	w := a.NewWindow("Microsoft 365 â€” MSP setup")
	w.Resize(fyne.NewSize(720, 580))
	w.CenterOnScreen()

	home := ui.GetAppHome()
	auth := mspauth.NewAuthenticator(home)
	ui.ShowMasterSetupWizard(w, ui.MasterSetupOptions{
		Provider: idp.ProviderEntra,
		Home:     home,
		Enroll:   auth.EnrollAndVerify,
	})
	w.ShowAndRun()
}

