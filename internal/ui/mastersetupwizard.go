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

const fullMSPStepTotal = 6

// MasterSetupOptions wires full MSP admin setup (auth, security, APIs, engineer pack).
type MasterSetupOptions struct {
	Provider idp.Provider
	Home     string
	Enroll   EnrollHandler
}

// ShowMasterSetupWizard runs full MSP setup: branding → auth → security → APIs → engineer installers.
func ShowMasterSetupWizard(w fyne.Window, opts MasterSetupOptions) {
	if w == nil {
		return
	}
	provider := opts.Provider.Normalize()
	if !provider.RequiresCloudLogin() {
		dialog.ShowInformation("MSP setup", "Pick Microsoft 365 or Google Workspace.", w)
		return
	}

	var (
		step       int
		branding   *BrandingForm
		cloudForm  *AccessSetupForm
		security   *SecurityPolicyForm
	apiPanel   *MSPIntegrationPanel
	apiFields  SettingsFields
	cursorForm *CursorConfigForm
		bodySlot   = container.NewMax()
		stepBar    = widget.NewProgressBar()
		backBtn    = widget.NewButtonWithIcon("Back", theme.NavigateBackIcon(), nil)
		nextBtn    = widget.NewButtonWithIcon("Next", theme.NavigateNextIcon(), nil)
		cancelBtn  = widget.NewButton("Close", func() { fyne.CurrentApp().Quit() })
	)
	nextBtn.Importance = widget.HighImportance

	presetBrand, _, _ := mspbranding.Load()
	presetSec, _, _ := mspsecurity.Load()
	branding = NewBrandingForm(w, presetBrand)
	cloudForm = NewMasterSetupForm(w, provider)
	security = NewSecurityPolicyForm(presetSec)
	apiPanel = newMSPIntegrationPanel()
	cursorForm = NewCursorConfigForm()
	base, _ := LoadSettings(SettingsPath())
	apiFields = SettingsFieldsOf(base)
	apiPanel.load(apiFields)
	cursorForm.load(apiFields)

	title := provider.ChoiceLabel() + " full MSP setup"
	hero := installWizardHero(title, "Auth, security, integrations, and engineer installers")

	var showStep func()

	showStep = func() {
		stepBar.SetValue(float64(step) / float64(fullMSPStepTotal))
		backBtn.Show()
		nextBtn.Show()
		nextBtn.SetText("Next")
		nextBtn.SetIcon(theme.NavigateNextIcon())

		switch step {
		case 0:
			bodySlot.Objects = []fyne.CanvasObject{
				container.NewVBox(hero, fullStepBadge(1, "Branding"),
					widget.NewCard("MSP branding", "Logo and titles for engineers", branding.Content())),
			}
			backBtn.Hide()
			nextBtn.OnTapped = func() {
				if err := branding.Save(); err != nil {
					dialog.ShowError(err, w)
					return
				}
				step = 1
				showStep()
			}
		case 1:
			bodySlot.Objects = []fyne.CanvasObject{
				container.NewVBox(hero, fullStepBadge(2, "Cloud authentication"),
					widget.NewCard("Application registration", "Azure or Google Cloud OAuth app", cloudForm.Content())),
			}
			backBtn.OnTapped = func() { step = 0; showStep() }
			nextBtn.SetText("Verify tenant sign-in")
			nextBtn.SetIcon(theme.ConfirmIcon())
			nextBtn.OnTapped = func() {
				if err := cloudForm.Validate(); err != nil {
					dialog.ShowError(err, w)
					return
				}
				enroll := cloudForm.Enrollment()
				RunEnrollment(w, enroll, opts.Enroll, func(en mspauth.Enrollment, sess mspauth.UserSession) {
					step = 2
					showStep()
					who := sess.Email
					if who == "" {
						who = sess.Name
					}
					dialog.ShowInformation("Authentication verified",
						fmt.Sprintf("Enrollment saved.\nVerified as: %s", who), w)
				})
			}
		case 2:
			bodySlot.Objects = []fyne.CanvasObject{
				container.NewVBox(hero, fullStepBadge(3, "Security policy"),
					widget.NewCard("Organization security", "Shipped to engineer PCs (read-only, change windows, vault)", security.Content())),
			}
			backBtn.OnTapped = func() { step = 1; showStep() }
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
				step = 3
				showStep()
			}
		case 3:
			intScroll := container.NewScroll(apiPanel.content())
			intScroll.SetMinSize(fyne.NewSize(680, 320))
			cursorScroll := container.NewScroll(cursorForm.Content())
			cursorScroll.SetMinSize(fyne.NewSize(680, 140))
			bodySlot.Objects = []fyne.CanvasObject{
				container.NewVBox(hero, fullStepBadge(4, "API integrations & Cursor AI"),
					widget.NewCard("PSA, RMM, inventory, vault, incidents", "Saved to engineer package", intScroll),
					widget.NewCard("Cursor AI", "Troubleshoot addon for engineers", cursorScroll)),
			}
			backBtn.OnTapped = func() { step = 2; showStep() }
			nextBtn.OnTapped = func() {
				merged := apiPanel.fields(apiFields)
				merged = cursorForm.apply(merged)
				updated, errs := merged.Apply(LoadSettingsOrDefaults())
				if len(errs) > 0 {
					dialog.ShowError(errs[0], w)
					return
				}
				if err := SaveSettings(SettingsPath(), updated); err != nil {
					dialog.ShowError(err, w)
					return
				}
				SetSettings(updated)
				if err := SaveEngineerSettingsSnapshot(); err != nil {
					dialog.ShowError(err, w)
					return
				}
				step = 4
				showStep()
			}
		case 4:
			genBtn := widget.NewButtonWithIcon("Create engineer standalone installer…", theme.DownloadIcon(), func() {
				dialog.NewFolderOpen(func(lu fyne.ListableURI, err error) {
					if err != nil || lu == nil {
						return
					}
					installExe, err := appinstall.BuildEngineerPack(appinstall.EngineerPackOptions{DestDir: lu.Path()})
					if err != nil {
						dialog.ShowError(err, w)
						return
					}
					dialog.ShowInformation("Engineer standalone package ready",
						fmt.Sprintf("Distribute this folder to engineers.\nThey run only:\n%s\n\nNo admin auth or security setup on engineer PCs.", installExe), w)
				}, w).Show()
			})
			genBtn.Importance = widget.HighImportance
			bodySlot.Objects = []fyne.CanvasObject{
				container.NewVBox(hero, fullStepBadge(5, "Engineer standalone installers"),
					widget.NewCard("Build engineer package", "Branded install + pre-configured MSP (no admin tools)",
						container.NewVBox(
							widget.NewLabel("Creates a standalone installer exe with branding, org sign-in, security policy, and API settings."),
							widget.NewLabel("Engineer bundle contains only Pathfinder + pfseed — not master setup tools."),
							genBtn,
						))),
			}
			backBtn.OnTapped = func() { step = 3; showStep() }
			nextBtn.Hide()
		default:
			step = 0
			showStep()
		}
		bodySlot.Refresh()
	}

	footer := container.NewBorder(nil, nil, backBtn, container.NewHBox(cancelBtn, nextBtn), nil)
	w.SetContent(container.NewBorder(nil, container.NewPadded(footer), nil, nil, container.NewPadded(bodySlot)))
	showStep()
}

func fullStepBadge(step int, name string) fyne.CanvasObject {
	return widget.NewLabelWithStyle(
		fmt.Sprintf("Step %d of %d — %s", step, fullMSPStepTotal, name),
		fyne.TextAlignLeading,
		fyne.TextStyle{Bold: true},
	)
}

func LoadSettingsOrDefaults() Settings {
	s, err := LoadSettings(SettingsPath())
	if err != nil {
		return Defaults()
	}
	return s
}
