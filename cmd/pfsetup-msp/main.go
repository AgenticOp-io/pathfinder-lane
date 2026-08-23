// pfsetup-msp — single MSP admin wizard (branding, cloud auth, security, engineer packs).
// API keys and Cursor are configured in Pathfinder Settings → Tools, not in this wizard.
package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"github.com/scottpeterman/pathfinderssh/internal/idp"
	"github.com/scottpeterman/pathfinderssh/internal/mspauth"
	"github.com/scottpeterman/pathfinderssh/internal/ui"
)

func main() {
	a := app.NewWithID("com.pathfinder.pfsetup.msp")
	ui.LoadUserThemes()
	ui.ApplyInstallerTheme(a)
	if icon := ui.AppIcon(); icon != nil {
		a.SetIcon(icon)
	}
	w := a.NewWindow("PathfinderSSH — MSP setup")
	w.Resize(fyne.NewSize(720, 580))
	w.CenterOnScreen()

	home := ui.GetAppHome()
	auth := mspauth.NewAuthenticator(home)
	// ProviderLocal triggers the in-wizard Microsoft 365 / Google picker.
	ui.ShowMasterSetupWizard(w, ui.MasterSetupOptions{
		Provider: idp.ProviderLocal,
		Home:     home,
		Enroll:   auth.EnrollAndVerify,
	})
	w.ShowAndRun()
}
