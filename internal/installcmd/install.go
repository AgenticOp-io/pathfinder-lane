package installcmd

import (
	"fmt"
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/appinstall"
	"github.com/scottpeterman/pathfinderssh/internal/mspauth"
	"github.com/scottpeterman/pathfinderssh/internal/ui"
)

// Options configures a non-interactive install from the command line.
type Options struct {
	Setup     string
	Enroll    bool
	Home      string
	BundleDir string // optional folder containing pathfinder.exe, pfseed.exe, ...
	Update    bool   // reinstall / refresh binaries (same as install; documents intent)
}

// Run copies the tool bundle to AppData, creates shortcuts, and optionally configures sign-in.
func Run(opts Options) (destExe string, err error) {
	dest, copied, err := appinstall.EnsureFrom(opts.BundleDir)
	if err != nil {
		return "", err
	}
	if err := appinstall.CreateShortcuts(dest); err != nil {
		return "", err
	}
	if err := applySetup(opts); err != nil {
		return dest, err
	}
	if copied {
		fmt.Println("Installed to", dest)
	} else if opts.Update {
		fmt.Println("Already up to date at", dest)
	} else {
		fmt.Println("Already installed at", dest)
	}
	return dest, nil
}

func applySetup(opts Options) error {
	setup := strings.TrimSpace(opts.Setup)
	if setup == "" {
		return nil
	}
	home := strings.TrimSpace(opts.Home)
	if home == "" {
		home = ui.GetAppHome()
	}
	if mspauth.HeadlessSetup(setup) {
		if err := mspauth.SaveSoloSetup(home); err != nil {
			return fmt.Errorf("setup %s: %w", setup, err)
		}
		fmt.Println("Solo mode — no Microsoft/Google sign-in required.")
		return nil
	}
	if _, ok := mspauth.ParseSetupMode(setup); ok {
		fmt.Println("Cloud organization setup uses separate tools after install:")
		fmt.Println("  pfsetup-o365.exe   Microsoft 365 tenant registration")
		fmt.Println("  pfsetup-google.exe Google Workspace registration")
		fmt.Println("  pfsetup-apis.exe   API keys for PSA/RMM/inventory")
		return nil
	}
	return fmt.Errorf("unknown setup mode %q (use solo)", setup)
}
