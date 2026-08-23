// pfenroll — deprecated: use pfsetup-o365 or pfsetup-google for org enrollment.
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"

	"github.com/scottpeterman/pathfinderssh/internal/appinstall"
	"github.com/scottpeterman/pathfinderssh/internal/ui"
)

func main() {
	enrollmentPath := flag.String("enrollment", "", "path to msp-enrollment.json (sets PATHFINDER_MSP_ENROLLMENT)")
	flag.Parse()
	if p := strings.TrimSpace(*enrollmentPath); p != "" {
		os.Setenv("PATHFINDER_MSP_ENROLLMENT", p)
	}

	a := app.NewWithID("com.pathfinder.pfenroll")
	ui.LoadUserThemes()
	ui.ApplyInstallerTheme(a)
	w := a.NewWindow("Organization setup")
	w.Resize(fyne.NewSize(520, 320))
	w.CenterOnScreen()

	body := fmt.Sprintf(
		"Organization enrollment moved to dedicated setup tools:\n\n"+
			"• pfsetup-o365.exe — Microsoft 365\n"+
			"• pfsetup-google.exe — Google Workspace\n\n"+
			"Installed copy: %s",
		appinstall.BinDir(),
	)
	dialog.ShowInformation("Use pfsetup-o365 or pfsetup-google", body, w)
	w.ShowAndRun()
}
