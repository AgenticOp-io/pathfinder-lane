package ui

import (
	"context"
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/scottpeterman/pathfinderssh/internal/idp"
	"github.com/scottpeterman/pathfinderssh/internal/mspauth"
)

// MSPSetupMode is which setup step to show.
type MSPSetupMode int

const (
	MSPSetupEnroll MSPSetupMode = iota
	MSPSetupLogin
)

// MSPSetupOptions wires MSP enrollment and engineer login.
type MSPSetupOptions struct {
	Mode           MSPSetupMode
	PresetProvider idp.Provider
	Enrollment     mspauth.Enrollment
	OnComplete     func(enroll mspauth.Enrollment, sess mspauth.UserSession)
	Authenticate   func(ctx context.Context, enroll mspauth.Enrollment) (mspauth.UserSession, error)
	Enroll         EnrollHandler
}

// ShowMSPSetupDialog blocks on first-run setup or engineer login.
func ShowMSPSetupDialog(w fyne.Window, opts MSPSetupOptions) {
	if w == nil {
		return
	}
	switch opts.Mode {
	case MSPSetupLogin:
		showMSPLoginDialog(w, opts)
	default:
		showMSPEnrollDialog(w, opts)
	}
}

func showMSPEnrollDialog(w fyne.Window, opts MSPSetupOptions) {
	form := NewAccessSetupForm(w, opts.PresetProvider)
	body := container.NewVBox(
		widget.NewLabelWithStyle("How do you sign in?", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		form.Content(),
	)

	d := dialog.NewCustomConfirm("PathfinderSSH setup", "Continue", "Cancel", body, func(ok bool) {
		if !ok {
			fyne.CurrentApp().Quit()
			return
		}
		RunEnrollment(w, form.Enrollment(), opts.Enroll, opts.OnComplete)
	}, w)
	d.Resize(fyne.NewSize(620, 480))
	d.Show()
}

func showMSPLoginDialog(w fyne.Window, opts MSPSetupOptions) {
	enroll := opts.Enrollment
	label := enroll.Provider.ChoiceLabel()
	if enroll.Domain != "" {
		label += " · " + enroll.Domain
	}
	body := container.NewVBox(
		widget.NewLabelWithStyle("Sign in", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Sign in with your work account to continue.\nMode: "+label),
	)
	d := dialog.NewCustomConfirm("Sign in required", "Sign in", "Quit", body, func(ok bool) {
		if !ok {
			fyne.CurrentApp().Quit()
			return
		}
		if opts.Authenticate == nil {
			dialog.ShowError(fmt.Errorf("authenticate handler not configured"), w)
			return
		}
		progress := dialog.NewCustomWithoutButtons("Signing in",
			widget.NewLabel("Complete sign-in in your browser…"), w)
		progress.Show()
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()
			sess, err := opts.Authenticate(ctx, enroll)
			fyne.Do(func() {
				progress.Hide()
				if err != nil {
					dialog.ShowError(err, w)
					return
				}
				if opts.OnComplete != nil {
					opts.OnComplete(enroll, sess)
				}
			})
		}()
	}, w)
	d.Resize(fyne.NewSize(480, 220))
	d.Show()
}
