package ui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/scottpeterman/pathfinderssh/internal/appinstall"
	"github.com/scottpeterman/pathfinderssh/internal/mspauth"
	"github.com/scottpeterman/pathfinderssh/internal/mspbranding"
)

const installPhaseTotal = 2 // ready, complete (progress is transient)

// InstallWizardOptions wires the graphical file installer (no cloud OAuth).
type InstallWizardOptions struct {
	Version string
	Home    string
}

// ShowInstallWizard runs a short install flow: ready → progress → complete.
func ShowInstallWizard(w fyne.Window, opts InstallWizardOptions) {
	if w == nil {
		return
	}

	var (
		phase      int
		destExe    string
		soloMode   = widget.NewCheck("Solo mode (local vault — configure Auvik, PSA, and Cursor without Microsoft/Google sign-in)", nil)
		bodySlot   = container.NewMax()
		stepBar    = widget.NewProgressBar()
		installBtn = widget.NewButtonWithIcon("Install", theme.DownloadIcon(), nil)
		cancelBtn  = widget.NewButton("Cancel", func() { fyne.CurrentApp().Quit() })
	)
	soloMode.SetChecked(true)
	installBtn.Importance = widget.HighImportance
	stepBar.SetValue(0)

	bundleDir := bundleDirFromExe()
	title, subtitle := brandedInstallTitles(bundleDir)
	if hasBundleEnrollment(bundleDir) {
		soloMode.SetChecked(false)
		soloMode.Hide()
	}

	hero := installWizardHero(title, subtitle)

	var showPhase func()

	runInstall := func() {
		phase = 1
		showPhase()

		go func() {
			dest, err := installToAppData()
			if err != nil {
				fyne.Do(func() {
					phase = 0
					showPhase()
					dialog.ShowError(err, w)
				})
				return
			}
			destExe = dest

			if soloMode.Checked {
				home := strings.TrimSpace(opts.Home)
				if home == "" {
					home = GetAppHome()
				}
				if err := mspauth.SaveSoloSetup(home); err != nil {
					fyne.Do(func() {
						phase = 0
						showPhase()
						dialog.ShowError(err, w)
					})
					return
				}
			}

			fyne.Do(func() {
				phase = 2
				showPhase()
			})
		}()
	}

	installBtn.OnTapped = runInstall

	showPhase = func() {
		stepBar.SetValue(float64(phase) / float64(installPhaseTotal))

		switch phase {
		case 0:
			ver := strings.TrimSpace(opts.Version)
			if ver != "" {
				ver = "Version " + ver
			}
			bodySlot.Objects = []fyne.CanvasObject{
				container.NewVBox(
					hero,
					stepBar,
					widget.NewCard(
						"Ready to install",
						"Copies apps to AppData and creates shortcuts",
						container.NewVBox(
							widget.NewLabel("Includes: pathfinder, pfseed, integration setup tools, and installers."),
							widget.NewLabel("Your sessions and vault in ~/.pathfinderssh are kept when you update."),
							soloMode,
							widget.NewLabel(ver),
						),
					),
					widget.NewLabel("Cloud full MSP setup (Microsoft 365 / Google) runs separately after install."),
				),
			}
			installBtn.Show()
			cancelBtn.Show()
		case 1:
			bodySlot.Objects = []fyne.CanvasObject{
				container.NewVBox(
					hero,
					stepBar,
					widget.NewCard(
						"Installing",
						"Copying files",
						container.NewVBox(
							widget.NewLabel("Installing PathfinderSSH MSP…"),
							widget.NewProgressBarInfinite(),
						),
					),
				),
			}
			installBtn.Hide()
			cancelBtn.Hide()
		case 2:
			msg := "Installed to:\n" + destExe
			openBtn := widget.NewButtonWithIcon("Open PathfinderSSH", theme.MediaPlayIcon(), func() {
				if destExe != "" {
					launchInstalled(destExe)
				}
				fyne.CurrentApp().Quit()
			})
			openBtn.Importance = widget.HighImportance

			apiBtn := widget.NewButtonWithIcon("Standalone MSP setup (Auvik, APIs, Cursor AI)", theme.SettingsIcon(), func() {
				launchBundledTool("pfsetup-apis", w)
			})
			m365Btn := widget.NewButtonWithIcon("Full MSP setup — Microsoft 365", theme.ComputerIcon(), func() {
				launchBundledTool("pfsetup-o365", w)
			})
			googleBtn := widget.NewButtonWithIcon("Full MSP setup — Google Workspace", theme.ComputerIcon(), func() {
				launchBundledTool("pfsetup-google", w)
			})
			closeBtn := widget.NewButton("Close", func() { fyne.CurrentApp().Quit() })

			bodySlot.Objects = []fyne.CanvasObject{
				container.NewVBox(
					hero,
					stepBar,
					widget.NewCard(
						"Installation complete",
						"Run full MSP setup or open Pathfinder",
						container.NewVBox(
							widget.NewLabel(msg),
							widget.NewLabel("MSP admin next steps (auth, security, engineer packages):"),
							container.NewVBox(m365Btn, googleBtn),
							widget.NewLabel("Standalone / solo: configure integrations and Cursor without cloud sign-in:"),
							apiBtn,
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
	root := container.NewBorder(nil, container.NewPadded(footer), nil, nil, container.NewPadded(bodySlot))
	w.SetContent(root)
	showPhase()
}

func launchBundledTool(name string, w fyne.Window) {
	if err := appinstall.LaunchTool(name); err != nil {
		dialog.ShowError(fmt.Errorf("launch %s: %w", name, err), w)
	}
}

func installWizardHero(title, subtitle string) fyne.CanvasObject {
	titleLbl := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	subLbl := widget.NewLabel(subtitle)
	text := container.NewVBox(titleLbl, subLbl)
	if ic := AppIcon(); ic != nil {
		return container.NewHBox(widget.NewIcon(ic), container.NewPadded(text))
	}
	return text
}

func brandedInstallTitles(bundleDir string) (title, subtitle string) {
	title = "PathfinderSSH MSP"
	subtitle = "Install tools for this Windows profile"
	if bundleDir != "" {
		if b, ok, _ := mspbranding.LoadFile(filepath.Join(bundleDir, "msp-branding.json")); ok {
			title = b.InstallTitle()
			subtitle = b.InstallSubtitle()
			return title, subtitle
		}
	}
	if b, ok, _ := mspbranding.Load(); ok {
		title = b.InstallTitle()
		subtitle = b.InstallSubtitle()
	}
	return title, subtitle
}

func bundleDirFromExe() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Dir(exe)
}

func hasBundleEnrollment(bundleDir string) bool {
	if bundleDir == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(bundleDir, "msp-enrollment.json"))
	return err == nil
}

func installToAppData() (string, error) {
	packDir := bundleDirFromExe()
	if err := appinstall.StageOrgBundleFrom(packDir); err != nil {
		return "", fmt.Errorf("stage organization files: %w", err)
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

func launchInstalled(exe string) {
	if runtime.GOOS == "windows" {
		cmd := exec.Command(exe)
		cmd.Dir = appinstall.Root()
		_ = cmd.Start()
		return
	}
	_ = exec.Command(exe).Start()
}
