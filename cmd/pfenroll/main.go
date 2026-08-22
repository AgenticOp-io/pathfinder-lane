// pfenroll — MSP org enrollment wizard for super admins (no SSH UI).
//
//	go run ./cmd/pfenroll
//	go run ./cmd/pfenroll --enrollment %PROGRAMDATA%\PathfinderSSH-MSP\msp-enrollment.json
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"

	"github.com/scottpeterman/pathfinderssh/internal/mspauth"
	"github.com/scottpeterman/pathfinderssh/internal/ui"
)

func main() {
	enrollmentPath := flag.String("enrollment", "", "path to msp-enrollment.json (sets PATHFINDER_MSP_ENROLLMENT)")
	flag.Parse()
	if p := strings.TrimSpace(*enrollmentPath); p != "" {
		os.Setenv("PATHFINDER_MSP_ENROLLMENT", p)
	}

	a := app.NewWithID("com.pathfinder.pfenroll")
	w := a.NewWindow("Pathfinder MSP — Enroll organization")
	w.Resize(fyne.NewSize(640, 520))
	w.CenterOnScreen()

	home := ui.GetAppHome()
	auth := mspauth.NewAuthenticator(home)

	ui.ShowMSPSetupDialog(w, ui.MSPSetupOptions{
		Mode: ui.MSPSetupEnroll,
		Enroll: func(ctx context.Context, enroll mspauth.Enrollment) (mspauth.Enrollment, mspauth.UserSession, error) {
			return auth.EnrollAndVerify(ctx, enroll)
		},
		OnComplete: func(enroll mspauth.Enrollment, sess mspauth.UserSession) {
			path := mspauth.EnrollmentPath()
			who := firstNonEmpty(sess.Email, sess.Name, sess.Subject)
			msg := fmt.Sprintf("Enrollment saved to %s\nProvider: %s", path, enroll.Provider.Label())
			if who != "" {
				msg += "\nSigned in as: " + who
			}
			dialog.ShowInformation("Enrollment complete", msg, w)
			go func() {
				time.Sleep(400 * time.Millisecond)
				fyne.Do(func() { w.Close() })
			}()
		},
	})

	w.ShowAndRun()
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
