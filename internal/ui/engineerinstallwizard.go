package ui

import (
	"fmt"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/scottpeterman/pathfinderssh/internal/appinstall"
	"github.com/scottpeterman/pathfinderssh/internal/mspbranding"
)

// EngineerInstallOptions configures the engineer-only standalone installer.
type EngineerInstallOptions struct {
	Version string
}

// ShowEngineerInstallWizard installs a pre-configured MSP workstation (no admin setup).
func ShowEngineerInstallWizard(w fyne.Window, opts EngineerInstallOptions) {
	if w == nil {
		return
	}

	var (
		phase      int
		destExe    string
		bodySlot   = container.NewMax()
		stepBar    = widget.NewProgressBar()
		installBtn = widget.NewButtonWithIcon("Install", theme.DownloadIcon(), nil)
		cancelBtn  = widget.NewButton("Cancel", func() { fyne.CurrentApp().Quit() })
	)
	installBtn.Importance = widget.HighImportance

	packDir := bundleDirFromExe()
	title, subtitle := brandedInstallTitles(packDir)
	if !hasBundleEnrollment(packDir) {
		dialog.ShowError(fmt.Errorf("this installer is missing organization sign-in (msp-enrollment.json) — use the MSP admin full setup package"), w)
		return
	}

	hero := installWizardHero(title, subtitle)

	var showPhase func()

	runInstall := func() {
		phase = 1
		showPhase()
		go func() {
			dest, err := engineerInstallToAppData()
			if err != nil {
				fyne.Do(func() {
					phase = 0
					showPhase()
					dialog.ShowError(err, w)
				})
				return
			}
			destExe = dest
			fyne.Do(func() {
				phase = 2
				showPhase()
			})
		}()
	}
	installBtn.OnTapped = runInstall

	showPhase = func() {
		stepBar.SetValue(float64(phase) / 2.0)
		switch phase {
		case 0:
			ver := opts.Version
			if ver != "" {
				ver = "Version " + ver
			}
			bodySlot.Objects = []fyne.CanvasObject{
				container.NewVBox(
					hero,
					stepBar,
					widget.NewCard(
						"Engineer workstation install",
						"Pre-configured by your MSP — sign in with your work account after install",
						container.NewVBox(
							widget.NewLabel("Installs Pathfinder with your organization branding, sign-in, security policy, and API integrations."),
							widget.NewLabel("Add or change Auvik, PSA, vault, or Cursor AI anytime in Settings → Tools after install."),
							widget.NewLabel("You do not need Azure, Google, or API admin setup on this PC."),
							widget.NewLabel(ver),
						),
					),
				),
			}
			installBtn.Show()
			cancelBtn.Show()
		case 1:
			bodySlot.Objects = []fyne.CanvasObject{
				container.NewVBox(
					hero,
					stepBar,
					widget.NewCard("Installing", "Copying application files", container.NewVBox(
						widget.NewLabel("Installing…"),
						widget.NewProgressBarInfinite(),
					)),
				),
			}
			installBtn.Hide()
			cancelBtn.Hide()
		case 2:
			openBtn := widget.NewButtonWithIcon("Open and sign in", theme.MediaPlayIcon(), func() {
				if destExe != "" {
					launchInstalled(destExe)
				}
				fyne.CurrentApp().Quit()
			})
			openBtn.Importance = widget.HighImportance
			closeBtn := widget.NewButton("Close", func() { fyne.CurrentApp().Quit() })
			bodySlot.Objects = []fyne.CanvasObject{
				container.NewVBox(
					hero,
					stepBar,
					widget.NewCard(
						"Ready to sign in",
						"Use your work account when Pathfinder opens",
						container.NewVBox(
							widget.NewLabel("Installed to:\n"+destExe),
							widget.NewLabel("Security policy, integrations, and Cursor AI settings were applied from your MSP package."),
							widget.NewLabel("To add Auvik or other integrations locally: Settings → Tools in Pathfinder."),
							container.NewHBox(openBtn, closeBtn),
						),
					),
				),
			}
			installBtn.Hide()
			cancelBtn.Hide()
		}
		bodySlot.Refresh()
	}

	footer := container.NewBorder(nil, nil, nil, container.NewHBox(cancelBtn, installBtn), nil)
	w.SetContent(container.NewBorder(nil, container.NewPadded(footer), nil, nil, container.NewPadded(bodySlot)))
	showPhase()
}

func engineerInstallToAppData() (string, error) {
	packDir := bundleDirFromExe()
	if err := appinstall.StageOrgBundleFrom(packDir); err != nil {
		return "", fmt.Errorf("stage organization files: %w", err)
	}
	if err := ApplyStagedMSPConfig(packDir); err != nil {
		return "", fmt.Errorf("apply MSP policy: %w", err)
	}
	toolDir := packDir
	bundleSub := filepath.Join(packDir, "bundle")
	if st, err := os.Stat(filepath.Join(bundleSub, "pathfinder.exe")); err == nil && !st.IsDir() {
		toolDir = bundleSub
	}
	dest, _, err := appinstall.EnsureFrom(toolDir)
	if err != nil {
		return "", fmt.Errorf("copy to AppData: %w", err)
	}
	if err := appinstall.CreateShortcuts(dest); err != nil {
		return "", fmt.Errorf("create shortcuts: %w", err)
	}
	SetLogoPath(mspbranding.LogoPath())
	return dest, nil
}
