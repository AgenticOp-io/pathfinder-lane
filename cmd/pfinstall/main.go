// pfinstall — graphical installer (solo, Microsoft 365, or Google).
//
//	go run ./cmd/pfinstall
//	go run ./cmd/pfinstall --setup o365
package main

import (
	"context"
	"flag"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"github.com/scottpeterman/pathfinderssh/internal/mspauth"
	"github.com/scottpeterman/pathfinderssh/internal/ui"
)

func main() {
	setup := flag.String("setup", "", "preset: solo, o365, google")
	flag.Parse()

	a := app.NewWithID("com.pathfinder.pfinstall")
	w := a.NewWindow("Install PathfinderSSH")
	w.Resize(fyne.NewSize(640, 520))
	w.CenterOnScreen()

	home := ui.GetAppHome()
	auth := mspauth.NewAuthenticator(home)

	ui.ShowInstallWizard(w, ui.InstallWizardOptions{
		PresetSetup: *setup,
		Home:        home,
		Enroll: func(ctx context.Context, enroll mspauth.Enrollment) (mspauth.Enrollment, mspauth.UserSession, error) {
			return auth.EnrollAndVerify(ctx, enroll)
		},
	})

	w.ShowAndRun()
}
