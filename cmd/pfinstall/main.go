// pfinstall — PathfinderSSH MSP installer (graphical and command-line).
//
// Graphical (default):
//   go run ./cmd/pfinstall
//   pfinstall.exe -install-gui
//   pfinstall.exe -install-gui -setup o365
//
// Silent / scripted:
//   pfinstall.exe -install
//   pfinstall.exe -install -setup solo
//   pfinstall.exe -install -setup o365 -enroll
//   pfinstall.exe -uninstall
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"github.com/scottpeterman/pathfinderssh/internal/appinstall"
	"github.com/scottpeterman/pathfinderssh/internal/installcmd"
	"github.com/scottpeterman/pathfinderssh/internal/mspauth"
	"github.com/scottpeterman/pathfinderssh/internal/ui"
)

func main() {
	var (
		doInstall    = flag.Bool("install", false, "copy bundle to AppData and create shortcuts (no GUI)")
		doInstallGUI = flag.Bool("install-gui", false, "graphical install wizard")
		doUninstall  = flag.Bool("uninstall", false, "remove AppData install and shortcuts")
		setup        = flag.String("setup", "", "access mode: solo, o365, google")
		doEnroll     = flag.Bool("enroll", false, "complete cloud sign-in during CLI install")
	)
	flag.Parse()

	if *doUninstall {
		if err := appinstall.Uninstall(); err != nil {
			fmt.Fprintf(os.Stderr, "uninstall: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Removed", appinstall.Root())
		return
	}

	setupMode := strings.TrimSpace(*setup)
	gui := *doInstallGUI || (!*doInstall && flag.NArg() == 0 && setupMode == "" && !*doEnroll)

	if *doInstall {
		_, err := installcmd.Run(installcmd.Options{
			Setup:  setupMode,
			Enroll: *doEnroll,
			Home:   ui.GetAppHome(),
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "install: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if gui {
		runInstallGUI(setupMode)
		return
	}

	fmt.Fprintln(os.Stderr, "pfinstall: use -install, -install-gui, or -uninstall")
	flag.PrintDefaults()
	os.Exit(2)
}

func runInstallGUI(setupPreset string) {
	a := app.NewWithID("com.pathfinder.pfinstall")
	ui.LoadUserThemes()
	base, _ := ui.LoadSettings(ui.SettingsPath())
	ui.SetSettings(base)
	ui.ApplyAppTheme(a, base.AppVariant())
	if icon := ui.AppIcon(); icon != nil {
		a.SetIcon(icon)
	}
	w := a.NewWindow("Install PathfinderSSH MSP")
	w.Resize(fyne.NewSize(640, 520))
	w.CenterOnScreen()

	home := ui.GetAppHome()
	auth := mspauth.NewAuthenticator(home)
	ui.ShowInstallWizard(w, ui.InstallWizardOptions{
		PresetSetup: setupPreset,
		Home:        home,
		Enroll:      auth.EnrollAndVerify,
	})
	w.ShowAndRun()
}
