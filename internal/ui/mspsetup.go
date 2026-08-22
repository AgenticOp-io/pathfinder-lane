package ui

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/scottpeterman/pathfinderssh/internal/idp"
	"github.com/scottpeterman/pathfinderssh/internal/mspauth"
)

// MSPSetupMode is which setup step to show.
type MSPSetupMode int

const (
	MSPSetupEnroll MSPSetupMode = iota // pick access mode + optional cloud IDs
	MSPSetupLogin                      // engineer signs in
)

// MSPSetupOptions wires MSP enrollment and engineer login.
type MSPSetupOptions struct {
	Mode MSPSetupMode
	// PresetProvider pre-selects Microsoft 365 or Google from -setup o365|google.
	PresetProvider idp.Provider
	// Existing enrollment when Mode is Login.
	Enrollment mspauth.Enrollment
	OnComplete func(enroll mspauth.Enrollment, sess mspauth.UserSession)
	Authenticate func(ctx context.Context, enroll mspauth.Enrollment) (mspauth.UserSession, error)
	Enroll       func(ctx context.Context, enroll mspauth.Enrollment) (mspauth.Enrollment, mspauth.UserSession, error)
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

func openHelpURL(w fyne.Window, raw string) {
	u, err := url.Parse(raw)
	if err != nil {
		dialog.ShowError(err, w)
		return
	}
	if err := fyne.CurrentApp().OpenURL(u); err != nil {
		dialog.ShowInformation("Open in browser", raw, w)
	}
}

func showMSPEnrollDialog(w fyne.Window, opts MSPSetupOptions) {
	choices := []string{
		idp.ProviderLocal.ChoiceLabel(),
		idp.ProviderEntra.ChoiceLabel(),
		idp.ProviderGoogle.ChoiceLabel(),
	}
	providerSel := widget.NewSelect(choices, nil)
	providerSel.SetSelected(choices[0])
	if p := opts.PresetProvider.Normalize(); p.RequiresCloudLogin() {
		providerSel.SetSelected(p.ChoiceLabel())
	}

	tenant := widget.NewEntry()
	tenant.SetPlaceHolder("Directory (tenant) ID — from Azure app Overview")
	client := widget.NewEntry()
	client.SetPlaceHolder("Client ID")
	domain := widget.NewEntry()
	domain.SetPlaceHolder("Your email domain (e.g. contoso.com)")

	msHelp := widget.NewButtonWithIcon("Microsoft setup steps", theme.HelpIcon(), func() {
		openHelpURL(w, "https://learn.microsoft.com/en-us/entra/identity-platform/quickstart-register-app")
	})
	googleHelp := widget.NewButtonWithIcon("Google setup steps", theme.HelpIcon(), func() {
		openHelpURL(w, "https://developers.google.com/identity/protocols/oauth2/native-app")
	})
	openEntra := widget.NewButtonWithIcon("Open Azure app registrations", theme.NavigateNextIcon(), func() {
		openHelpURL(w, "https://entra.microsoft.com/#view/Microsoft_AAD_RegisteredApps/ApplicationsListBlade")
	})
	openGoogle := widget.NewButtonWithIcon("Open Google credentials", theme.NavigateNextIcon(), func() {
		openHelpURL(w, "https://console.cloud.google.com/apis/credentials")
	})

	entraHelp := container.NewVBox(
		widget.NewLabel("One-time setup in Azure (about 5 minutes):\n"+
			"1. New registration → name PathfinderSSH, single tenant\n"+
			"2. Redirect URI (Web): http://127.0.0.1:53682/callback\n"+
			"3. Authentication → Allow public client flows: Yes\n"+
			"4. Copy Tenant ID and Client ID below"),
		container.NewHBox(msHelp, openEntra),
		widget.NewForm(
			widget.NewFormItem("Tenant ID", tenant),
			widget.NewFormItem("Client ID", client),
			widget.NewFormItem("Email domain", domain),
		),
	)
	googleHelpBox := container.NewVBox(
		widget.NewLabel("One-time setup in Google Cloud:\n"+
			"1. Create project (or pick one)\n"+
			"2. Credentials → Create OAuth client ID → Desktop (or Web with redirect\n"+
			"   http://127.0.0.1:53682/callback)\n"+
			"3. Copy Client ID below"),
		container.NewHBox(googleHelp, openGoogle),
		widget.NewForm(
			widget.NewFormItem("Client ID", client),
			widget.NewFormItem("Workspace domain", domain),
		),
	)
	soloNote := widget.NewLabel("No Microsoft or Google login. Your Windows user is enough.\n"+
		"Create a vault password when prompted — credentials stay on this PC.")

	entraHelp.Hide()
	googleHelpBox.Hide()
	soloNote.Hide()

	refreshProviderUI := func() {
		entraHelp.Hide()
		googleHelpBox.Hide()
		soloNote.Hide()
		switch providerSel.Selected {
		case idp.ProviderEntra.ChoiceLabel():
			entraHelp.Show()
		case idp.ProviderGoogle.ChoiceLabel():
			googleHelpBox.Show()
		default:
			soloNote.Show()
		}
	}
	providerSel.OnChanged = func(string) { refreshProviderUI() }
	refreshProviderUI()

	body := container.NewVBox(
		widget.NewLabelWithStyle("How do you sign in?", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Pick one. Solo is easiest — no cloud app registration."),
		widget.NewForm(widget.NewFormItem("Sign-in", providerSel)),
		soloNote,
		entraHelp,
		googleHelpBox,
	)

	d := dialog.NewCustomConfirm("PathfinderSSH setup", "Continue", "Cancel", body, func(ok bool) {
		if !ok {
			fyne.CurrentApp().Quit()
			return
		}
		var enroll mspauth.Enrollment
		switch idp.ProviderFromChoiceLabel(providerSel.Selected) {
		case idp.ProviderEntra:
			enroll = mspauth.Enrollment{
				Provider: mspauth.ProviderEntra,
				TenantID: tenant.Text,
				ClientID: client.Text,
				Domain:   domain.Text,
			}
		case idp.ProviderGoogle:
			enroll = mspauth.Enrollment{
				Provider: mspauth.ProviderGoogle,
				ClientID: client.Text,
				Domain:   domain.Text,
			}
		default:
			enroll = mspauth.Enrollment{Provider: mspauth.ProviderLocal}
		}
		if err := mspauth.ValidateEnrollment(enroll); err != nil {
			dialog.ShowError(err, w)
			return
		}
		if enroll.Provider == mspauth.ProviderLocal {
			if opts.Enroll == nil {
				dialog.ShowError(fmt.Errorf("enrollment handler not configured"), w)
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			enroll, sess, err := opts.Enroll(ctx, enroll)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if opts.OnComplete != nil {
				opts.OnComplete(enroll, sess)
			}
			return
		}
		if opts.Enroll == nil {
			dialog.ShowError(fmt.Errorf("enrollment handler not configured"), w)
			return
		}
		progress := dialog.NewCustomWithoutButtons("Signing in",
			widget.NewLabel("Complete sign-in in your browser, then return here…"), w)
		progress.Show()
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()
			enroll, sess, err := opts.Enroll(ctx, enroll)
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
