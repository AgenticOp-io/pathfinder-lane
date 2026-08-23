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

// AccessSetupForm is the shared Solo / Microsoft 365 / Google picker + cloud fields.
type AccessSetupForm struct {
	ProviderSel *widget.Select
	content     fyne.CanvasObject

	tenant *widget.Entry
	client *widget.Entry
	domain *widget.Entry
}

// NewAccessSetupForm builds sign-in mode UI with optional preset provider.
func NewAccessSetupForm(w fyne.Window, preset idp.Provider) *AccessSetupForm {
	f := &AccessSetupForm{}
	choices := []string{
		idp.ProviderLocal.ChoiceLabel(),
		idp.ProviderEntra.ChoiceLabel(),
		idp.ProviderGoogle.ChoiceLabel(),
	}
	f.ProviderSel = widget.NewSelect(choices, nil)
	f.ProviderSel.SetSelected(choices[0])
	if p := preset.Normalize(); p.RequiresCloudLogin() {
		f.ProviderSel.SetSelected(p.ChoiceLabel())
	}

	f.tenant = widget.NewEntry()
	f.tenant.SetPlaceHolder("Directory (tenant) ID â€” from Azure app Overview")
	f.client = widget.NewEntry()
	f.client.SetPlaceHolder("Client ID")
	f.domain = widget.NewEntry()
	f.domain.SetPlaceHolder("Your email domain (e.g. contoso.com)")

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

	entraBox := container.NewVBox(
		widget.NewLabel("One-time setup in Azure (about 5 minutes):\n"+
			"1. New registration â†’ name PathfinderSSH, single tenant\n"+
			"2. Redirect URI (Web): http://127.0.0.1:53682/callback\n"+
			"3. Authentication â†’ Allow public client flows: Yes\n"+
			"4. Copy Tenant ID and Client ID below"),
		container.NewHBox(msHelp, openEntra),
		widget.NewForm(
			widget.NewFormItem("Tenant ID", f.tenant),
			widget.NewFormItem("Client ID", f.client),
			widget.NewFormItem("Email domain", f.domain),
		),
	)
	googleBox := container.NewVBox(
		widget.NewLabel("One-time setup in Google Cloud:\n"+
			"1. Create project (or pick one)\n"+
			"2. Credentials â†’ Create OAuth client ID â†’ Desktop (or Web with redirect\n"+
			"   http://127.0.0.1:53682/callback)\n"+
			"3. Copy Client ID below"),
		container.NewHBox(googleHelp, openGoogle),
		widget.NewForm(
			widget.NewFormItem("Client ID", f.client),
			widget.NewFormItem("Workspace domain", f.domain),
		),
	)
	soloNote := widget.NewLabel("No Microsoft or Google login. Your Windows user is enough.\n"+
		"Configure Auvik, PSA, vault, and Cursor AI in Settings → Tools after install.")

	entraBox.Hide()
	googleBox.Hide()
	soloNote.Hide()

	refresh := func() {
		entraBox.Hide()
		googleBox.Hide()
		soloNote.Hide()
		switch f.ProviderSel.Selected {
		case idp.ProviderEntra.ChoiceLabel():
			entraBox.Show()
		case idp.ProviderGoogle.ChoiceLabel():
			googleBox.Show()
		default:
			soloNote.Show()
		}
	}
	f.ProviderSel.OnChanged = func(string) { refresh() }
	refresh()

	f.content = container.NewVBox(
		widget.NewLabel("Pick one. Solo is easiest â€” no cloud app registration."),
		widget.NewForm(widget.NewFormItem("Sign-in", f.ProviderSel)),
		soloNote,
		entraBox,
		googleBox,
	)
	return f
}

// Content returns the form widget tree.
func (f *AccessSetupForm) Content() fyne.CanvasObject {
	if f == nil {
		return widget.NewLabel("")
	}
	return f.content
}

// SetProvider selects sign-in mode by provider constant.
func (f *AccessSetupForm) SetProvider(p idp.Provider) {
	if f == nil || f.ProviderSel == nil {
		return
	}
	f.ProviderSel.SetSelected(p.ChoiceLabel())
}

// Enrollment builds an enrollment record from current field values.
func (f *AccessSetupForm) Enrollment() mspauth.Enrollment {
	if f == nil {
		return mspauth.Enrollment{Provider: mspauth.ProviderLocal}
	}
	switch idp.ProviderFromChoiceLabel(f.ProviderSel.Selected) {
	case idp.ProviderEntra:
		return mspauth.Enrollment{
			Provider: mspauth.ProviderEntra,
			TenantID: f.tenant.Text,
			ClientID: f.client.Text,
			Domain:   f.domain.Text,
		}
	case idp.ProviderGoogle:
		return mspauth.Enrollment{
			Provider: mspauth.ProviderGoogle,
			ClientID: f.client.Text,
			Domain:   f.domain.Text,
		}
	default:
		return mspauth.Enrollment{Provider: mspauth.ProviderLocal}
	}
}

// Validate checks enrollment fields.
func (f *AccessSetupForm) Validate() error {
	return mspauth.ValidateEnrollment(f.Enrollment())
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

// EnrollHandler saves enrollment and optional verification sign-in.
type EnrollHandler func(ctx context.Context, enroll mspauth.Enrollment) (mspauth.Enrollment, mspauth.UserSession, error)

// RunEnrollment validates, enrolls, and reports completion (browser for cloud modes).
func RunEnrollment(w fyne.Window, enroll mspauth.Enrollment, enrollFn EnrollHandler, onComplete func(mspauth.Enrollment, mspauth.UserSession)) {
	if enrollFn == nil {
		dialog.ShowError(fmt.Errorf("enrollment handler not configured"), w)
		return
	}
	if err := mspauth.ValidateEnrollment(enroll); err != nil {
		dialog.ShowError(err, w)
		return
	}
	if enroll.Provider == mspauth.ProviderLocal {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		enroll, sess, err := enrollFn(ctx, enroll)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		if onComplete != nil {
			onComplete(enroll, sess)
		}
		return
	}
	progress := dialog.NewCustomWithoutButtons("Signing in",
		widget.NewLabel("Complete sign-in in your browser, then return here..."), w)
	progress.Show()
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		defer cancel()
		enroll, sess, err := enrollFn(ctx, enroll)
		fyne.Do(func() {
			progress.Hide()
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if onComplete != nil {
				onComplete(enroll, sess)
			}
		})
	}()
}

