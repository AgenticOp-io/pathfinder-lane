package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/scottpeterman/pathfinderssh/internal/appinstall"
	"github.com/scottpeterman/pathfinderssh/internal/idp"
	"github.com/scottpeterman/pathfinderssh/internal/mspauth"
	"github.com/scottpeterman/pathfinderssh/internal/mspbranding"
	"github.com/scottpeterman/pathfinderssh/internal/mspsecurity"
)

// MasterSetupOptions wires MSP admin setup (branding, auth, security, engineer pack).
// API keys and Cursor live in Settings → Tools after install — not in this wizard.
type MasterSetupOptions struct {
	Provider idp.Provider
	Home     string
	Enroll   EnrollHandler
}

// ShowMasterSetupWizard runs MSP setup: [provider] → branding → auth → security → engineer pack.
// When Provider is already Microsoft 365 or Google, the provider step is skipped.
func ShowMasterSetupWizard(w fyne.Window, opts MasterSetupOptions) {
	if w == nil {
		return
	}

	provider := opts.Provider.Normalize()
	pickProvider := !provider.RequiresCloudLogin()

	var (
		step      int
		branding  *BrandingForm
		cloudForm *AccessSetupForm
		security  *SecurityPolicyForm
		bodySlot  = container.NewMax()
		stepBar   = widget.NewProgressBar()
		backBtn   = widget.NewButtonWithIcon("Back", theme.NavigateBackIcon(), nil)
		nextBtn   = widget.NewButtonWithIcon("Next", theme.NavigateNextIcon(), nil)
		cancelBtn = widget.NewButton("Close", func() { fyne.CurrentApp().Quit() })
	)
	nextBtn.Importance = widget.HighImportance

	presetBrand, _, _ := mspbranding.Load()
	presetSec, _, _ := mspsecurity.Load()
	branding = NewBrandingForm(w, presetBrand)
	security = NewSecurityPolicyForm(presetSec)

	rebuildCloudForm := func() {
		cloudForm = NewMasterSetupForm(w, provider)
	}
	if !pickProvider {
		rebuildCloudForm()
	}

	stepTotal := func() int {
		if pickProvider {
			return 5
		}
		return 4
	}
	stepLabel := func(n int, name string) fyne.CanvasObject {
		return widget.NewLabelWithStyle(
			fmt.Sprintf("Step %d of %d — %s", n, stepTotal(), name),
			fyne.TextAlignLeading,
			fyne.TextStyle{Bold: true},
		)
	}

	heroTitle := "MSP setup"
	if !pickProvider {
		heroTitle = provider.ChoiceLabel() + " MSP setup"
	}
	hero := installWizardHero(heroTitle, "Branding, cloud sign-in, security, and engineer installers")

	var showStep func()

	// Logical steps after optional provider pick: branding=0, auth=1, security=2, engineer=3
	logical := func() int {
		if pickProvider {
			return step - 1
		}
		return step
	}

	showStep = func() {
		stepBar.SetValue(float64(step+1) / float64(stepTotal()))
		backBtn.Show()
		nextBtn.Show()
		nextBtn.SetText("Next")
		nextBtn.SetIcon(theme.NavigateNextIcon())

		// Provider pick (unified MSP entry only)
		if pickProvider && step == 0 {
			m365 := widget.NewButtonWithIcon("Microsoft 365", theme.ComputerIcon(), nil)
			google := widget.NewButtonWithIcon("Google Workspace", theme.ComputerIcon(), nil)
			m365.Importance = widget.HighImportance
			pick := func(p idp.Provider) {
				provider = p
				rebuildCloudForm()
				step = 1
				showStep()
			}
			m365.OnTapped = func() { pick(idp.ProviderEntra) }
			google.OnTapped = func() { pick(idp.ProviderGoogle) }
			bodySlot.Objects = []fyne.CanvasObject{
				container.NewVBox(hero, stepLabel(1, "Cloud provider"),
					widget.NewCard("Choose identity provider", "One MSP wizard for either cloud",
						container.NewVBox(
							widget.NewLabel("API keys (Auvik, PSA, Cursor, …) are configured later in Pathfinder Settings → Tools."),
							m365,
							google,
						))),
			}
			backBtn.Hide()
			nextBtn.Hide()
			bodySlot.Refresh()
			return
		}

		switch logical() {
		case 0: // branding
			display := 1
			if pickProvider {
				display = 2
			}
			bodySlot.Objects = []fyne.CanvasObject{
				container.NewVBox(hero, stepLabel(display, "Branding"),
					widget.NewCard("MSP branding", "Logo and titles for engineers", branding.Content())),
			}
			if pickProvider {
				backBtn.OnTapped = func() { step = 0; showStep() }
			} else {
				backBtn.Hide()
			}
			nextBtn.OnTapped = func() {
				if err := branding.Save(); err != nil {
					dialog.ShowError(err, w)
					return
				}
				step++
				showStep()
			}
		case 1: // auth
			display := 2
			if pickProvider {
				display = 3
			}
			bodySlot.Objects = []fyne.CanvasObject{
				container.NewVBox(hero, stepLabel(display, "Cloud authentication"),
					widget.NewCard("Application registration", "Azure or Google Cloud OAuth app", cloudForm.Content())),
			}
			backBtn.OnTapped = func() { step--; showStep() }
			nextBtn.SetText("Verify tenant sign-in")
			nextBtn.SetIcon(theme.ConfirmIcon())
			nextBtn.OnTapped = func() {
				if err := cloudForm.Validate(); err != nil {
					dialog.ShowError(err, w)
					return
				}
				enroll := cloudForm.Enrollment()
				RunEnrollment(w, enroll, opts.Enroll, func(en mspauth.Enrollment, sess mspauth.UserSession) {
					step++
					showStep()
					who := sess.Email
					if who == "" {
						who = sess.Name
					}
					dialog.ShowInformation("Authentication verified",
						fmt.Sprintf("Enrollment saved.\nVerified as: %s", who), w)
				})
			}
		case 2: // security
			display := 3
			if pickProvider {
				display = 4
			}
			bodySlot.Objects = []fyne.CanvasObject{
				container.NewVBox(hero, stepLabel(display, "Security policy"),
					widget.NewCard("Organization security", "Shipped to engineer PCs (read-only, change windows, vault)", security.Content())),
			}
			backBtn.OnTapped = func() { step--; showStep() }
			nextBtn.OnTapped = func() {
				if err := security.Save(); err != nil {
					dialog.ShowError(err, w)
					return
				}
				base, _ := LoadSettings(SettingsPath())
				merged := ApplyPolicyToSettings(base, security.Policy())
				if err := SaveSettings(SettingsPath(), merged.Normalized()); err != nil {
					dialog.ShowError(err, w)
					return
				}
				step++
				showStep()
			}
		case 3: // engineer pack
			display := 4
			if pickProvider {
				display = 5
			}
			genBtn := widget.NewButtonWithIcon("Create engineer standalone installer…", theme.DownloadIcon(), func() {
				dialog.NewFolderOpen(func(lu fyne.ListableURI, err error) {
					if err != nil || lu == nil {
						return
					}
					// Snapshot whatever is already in Settings (APIs/Cursor configured there).
					_ = SaveEngineerSettingsSnapshot()
					installExe, err := appinstall.BuildEngineerPack(appinstall.EngineerPackOptions{DestDir: lu.Path()})
					if err != nil {
						dialog.ShowError(err, w)
						return
					}
					dialog.ShowInformation("Engineer standalone package ready",
						fmt.Sprintf("Distribute this folder to engineers.\nThey run only:\n%s\n\nAPIs and Cursor are configured in Settings → Tools (on this admin PC before packaging, or on each engineer PC).", installExe), w)
				}, w).Show()
			})
			genBtn.Importance = widget.HighImportance
			bodySlot.Objects = []fyne.CanvasObject{
				container.NewVBox(hero, stepLabel(display, "Engineer installers"),
					widget.NewCard("Build engineer package", "Branded install + org sign-in + security (no admin tools)",
						container.NewVBox(
							widget.NewLabel("Creates a branded engineer installer with branding, org sign-in, and security policy."),
							widget.NewLabel("Configure Auvik, PSA, Cursor, and other APIs in Pathfinder Settings → Tools — not in this wizard."),
							widget.NewLabel("Optional: set APIs in Settings on this PC before packaging to ship them to engineers."),
							genBtn,
						))),
			}
			backBtn.OnTapped = func() { step--; showStep() }
			nextBtn.Hide()
		default:
			step = 0
			showStep()
			return
		}
		bodySlot.Refresh()
	}

	footer := container.NewBorder(nil, nil, backBtn, container.NewHBox(cancelBtn, nextBtn), nil)
	w.SetContent(container.NewBorder(nil, container.NewPadded(footer), nil, nil, container.NewPadded(bodySlot)))
	showStep()
}

// LoadSettingsOrDefaults returns disk settings or Defaults().
func LoadSettingsOrDefaults() Settings {
	s, err := LoadSettings(SettingsPath())
	if err != nil {
		return Defaults()
	}
	return s
}
