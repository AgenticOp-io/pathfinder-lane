package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/scottpeterman/pathfinderssh/internal/idp"
)

// NewMasterSetupForm builds a single-provider org enrollment form (no mode picker).
func NewMasterSetupForm(w fyne.Window, provider idp.Provider) *AccessSetupForm {
	p := provider.Normalize()
	switch p {
	case idp.ProviderEntra:
		return newEntraMasterForm(w)
	case idp.ProviderGoogle:
		return newGoogleMasterForm(w)
	default:
		return NewAccessSetupForm(w, idp.ProviderLocal)
	}
}

func newEntraMasterForm(w fyne.Window) *AccessSetupForm {
	f := &AccessSetupForm{}
	f.tenant = widget.NewEntry()
	f.tenant.SetPlaceHolder("Directory (tenant) ID")
	f.client = widget.NewEntry()
	f.client.SetPlaceHolder("Application (client) ID")
	f.domain = widget.NewEntry()
	f.domain.SetPlaceHolder("Email domain (e.g. contoso.com)")

	msHelp := widget.NewButtonWithIcon("Microsoft setup guide", theme.HelpIcon(), func() {
		openHelpURL(w, "https://learn.microsoft.com/en-us/entra/identity-platform/quickstart-register-app")
	})
	openEntra := widget.NewButtonWithIcon("Open Azure app registrations", theme.NavigateNextIcon(), func() {
		openHelpURL(w, "https://entra.microsoft.com/#view/Microsoft_AAD_RegisteredApps/ApplicationsListBlade")
	})

	f.content = container.NewVBox(
		widget.NewLabelWithStyle("Microsoft 365 organization setup", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Register one Azure app for your tenant, then sign in as a super admin to verify."),
		widget.NewLabel("Redirect URI (Web): http://127.0.0.1:53682/callback · Allow public client flows: Yes"),
		container.NewHBox(msHelp, openEntra),
		widget.NewForm(
			widget.NewFormItem("Tenant ID", f.tenant),
			widget.NewFormItem("Client ID", f.client),
			widget.NewFormItem("Email domain", f.domain),
		),
	)
	f.ProviderSel = widget.NewSelect([]string{idp.ProviderEntra.ChoiceLabel()}, nil)
	f.ProviderSel.SetSelected(idp.ProviderEntra.ChoiceLabel())
	return f
}

func newGoogleMasterForm(w fyne.Window) *AccessSetupForm {
	f := &AccessSetupForm{}
	f.client = widget.NewEntry()
	f.client.SetPlaceHolder("OAuth client ID")
	f.domain = widget.NewEntry()
	f.domain.SetPlaceHolder("Workspace domain (e.g. example.com)")

	googleHelp := widget.NewButtonWithIcon("Google setup guide", theme.HelpIcon(), func() {
		openHelpURL(w, "https://developers.google.com/identity/protocols/oauth2/native-app")
	})
	openGoogle := widget.NewButtonWithIcon("Open Google credentials", theme.NavigateNextIcon(), func() {
		openHelpURL(w, "https://console.cloud.google.com/apis/credentials")
	})

	f.content = container.NewVBox(
		widget.NewLabelWithStyle("Google Workspace organization setup", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Create a Google Cloud OAuth client for your Workspace, then sign in as a super admin."),
		widget.NewLabel("Redirect: http://127.0.0.1:53682/callback (Desktop or Web client)"),
		container.NewHBox(googleHelp, openGoogle),
		widget.NewForm(
			widget.NewFormItem("Client ID", f.client),
			widget.NewFormItem("Workspace domain", f.domain),
		),
	)
	f.ProviderSel = widget.NewSelect([]string{idp.ProviderGoogle.ChoiceLabel()}, nil)
	f.ProviderSel.SetSelected(idp.ProviderGoogle.ChoiceLabel())
	return f
}
