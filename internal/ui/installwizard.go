package ui

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/scottpeterman/pathfinderssh/internal/appinstall"
	"github.com/scottpeterman/pathfinderssh/internal/idp"
	"github.com/scottpeterman/pathfinderssh/internal/mspauth"
)

// InstallWizardOptions wires the graphical installer.
type InstallWizardOptions struct {
	Version     string
	PresetSetup string
	Home        string
	Enroll      EnrollHandler
}

// ShowInstallWizard runs the multi-step install UI in w.
func ShowInstallWizard(w fyne.Window, opts InstallWizardOptions) {
	if w == nil {
		return
	}
	preset, hasPreset := mspauth.ParseSetupMode(opts.PresetSetup)
	if !hasPreset {
		preset = idp.ProviderLocal
	}

	var (
		step       int
		destExe    string
		accessForm *AccessSetupForm
		bodySlot   = container.NewMax()
		backBtn    = widget.NewButtonWithIcon("Back", theme.NavigateBackIcon(), nil)
		nextBtn    = widget.NewButtonWithIcon("Next", theme.NavigateNextIcon(), nil)
		cancelBtn  = widget.NewButton("Cancel", func() { fyne.CurrentApp().Quit() })
	)

	title := widget.NewLabelWithStyle("Install PathfinderSSH", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})

	var showStep func()

	runInstall := func() {
		if accessForm == nil {
			return
		}
		if err := accessForm.Validate(); err != nil {
			dialog.ShowError(err, w)
			return
		}
		enroll := accessForm.Enrollment()
		step = 3
		showStep()

		go func() {
			dest, err := installToAppData()
			if err != nil {
				fyne.Do(func() {
					step = 2
					showStep()
					cancelBtn.Show()
					dialog.ShowError(err, w)
				})
				return
			}
			destExe = dest

			enrollFn := opts.Enroll
			if enrollFn == nil {
				auth := mspauth.NewAuthenticator(opts.Home)
				enrollFn = auth.EnrollAndVerify
			}

			if enroll.Provider == mspauth.ProviderLocal {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()
				_, _, enrollErr := enrollFn(ctx, enroll)
				fyne.Do(func() {
					if enrollErr != nil {
						step = 2
						showStep()
						cancelBtn.Show()
						dialog.ShowError(enrollErr, w)
						return
					}
					step = 4
					showStep()
				})
				return
			}

			fyne.Do(func() {
				progress := dialog.NewCustomWithoutButtons("Signing in",
					widget.NewLabel("Complete sign-in in your browser, then return here…"), w)
				progress.Show()
				go func() {
					ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
					defer cancel()
					_, _, enrollErr := enrollFn(ctx, enroll)
					fyne.Do(func() {
						progress.Hide()
						if enrollErr != nil {
							step = 2
							showStep()
							cancelBtn.Show()
							dialog.ShowError(enrollErr, w)
							return
						}
						step = 4
						showStep()
					})
				}()
			})
		}()
	}

	showStep = func() {
		backBtn.Disable()
		nextBtn.SetText("Next")
		nextBtn.SetIcon(theme.NavigateNextIcon())

		switch step {
		case 0:
			ver := strings.TrimSpace(opts.Version)
			if ver != "" {
				ver = " · " + ver
			}
			bodySlot.Objects = []fyne.CanvasObject{
				container.NewVBox(
					widget.NewLabel("Welcome! This copies PathfinderSSH MSP to your user folder\n"+
						"(pathfinder, pfseed, installer tools), creates shortcuts, and sets up sign-in."),
					widget.NewLabel("Free software (GPL-3.0). Source: github.com/AgenticOp-io/pathfinderssh-msp"+ver),
					widget.NewLabel("Step 1 of 3 — Welcome"),
				),
			}
			backBtn.Hide()
		case 1:
			if hasPreset {
				step = 2
				showStep()
				return
			}
			bodySlot.Objects = []fyne.CanvasObject{
				container.NewVBox(
					widget.NewLabel("Step 2 of 3 — How do you sign in?"),
					widget.NewLabel("Solo needs no Microsoft or Google app registration."),
					makeModePicker(func(p idp.Provider) {
						preset = p
						step = 2
						showStep()
					}),
				),
			}
			backBtn.Show()
			nextBtn.Hide()
		case 2:
			accessForm = NewAccessSetupForm(w, preset)
			accessForm.SetProvider(preset)
			bodySlot.Objects = []fyne.CanvasObject{
				container.NewVBox(
					widget.NewLabel("Step 3 of 3 — Sign-in details"),
					accessForm.Content(),
				),
			}
			backBtn.Show()
			if !hasPreset {
				backBtn.OnTapped = func() { step = 1; showStep() }
			} else {
				backBtn.OnTapped = func() { step = 0; showStep() }
			}
			nextBtn.Show()
			nextBtn.SetText("Install")
			nextBtn.SetIcon(theme.DownloadIcon())
			nextBtn.OnTapped = runInstall
		case 3:
			bodySlot.Objects = []fyne.CanvasObject{
				container.NewVBox(
					widget.NewLabel("Installing…"),
					widget.NewLabel("Copying files and creating shortcuts."),
				),
			}
			backBtn.Hide()
			nextBtn.Hide()
			cancelBtn.Hide()
		case 4:
			msg := "Installed to:\n" + destExe + "\n\nDesktop and Start Menu shortcuts are ready."
			openBtn := widget.NewButtonWithIcon("Open PathfinderSSH", theme.MediaPlayIcon(), func() {
				if destExe != "" {
					launchInstalled(destExe)
				}
				fyne.CurrentApp().Quit()
			})
			closeBtn := widget.NewButton("Close", func() { fyne.CurrentApp().Quit() })
			bodySlot.Objects = []fyne.CanvasObject{
				container.NewVBox(
					widget.NewLabelWithStyle("Done!", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
					widget.NewLabel(msg),
					container.NewHBox(openBtn, closeBtn),
				),
			}
			backBtn.Hide()
			nextBtn.Hide()
			cancelBtn.Hide()
		}
		bodySlot.Refresh()
	}

	nextBtn.OnTapped = func() {
		if step == 0 {
			if hasPreset {
				step = 2
			} else {
				step = 1
			}
			showStep()
		}
	}
	backBtn.OnTapped = func() {
		if step == 2 && !hasPreset {
			step = 1
		} else if step >= 2 {
			step = 0
		}
		showStep()
	}

	footer := container.NewBorder(nil, nil, backBtn, container.NewHBox(cancelBtn, nextBtn), nil)
	root := container.NewBorder(title, footer, nil, nil, container.NewPadded(bodySlot))
	w.SetContent(root)
	showStep()
}

func makeModePicker(onPick func(idp.Provider)) fyne.CanvasObject {
	solo := widget.NewButtonWithIcon(idp.ProviderLocal.ChoiceLabel(), theme.AccountIcon(), func() {
		onPick(idp.ProviderLocal)
	})
	solo.Importance = widget.HighImportance

	ms := widget.NewButtonWithIcon(idp.ProviderEntra.ChoiceLabel(), theme.ComputerIcon(), func() {
		onPick(idp.ProviderEntra)
	})
	google := widget.NewButtonWithIcon(idp.ProviderGoogle.ChoiceLabel(), theme.ComputerIcon(), func() {
		onPick(idp.ProviderGoogle)
	})
	return container.NewVBox(solo, ms, google)
}

func installToAppData() (string, error) {
	dest, _, err := appinstall.Ensure()
	if err != nil {
		return "", fmt.Errorf("copy to AppData: %w", err)
	}
	if err := appinstall.CreateShortcuts(dest); err != nil {
		return "", fmt.Errorf("create shortcuts: %w", err)
	}
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
