package installcmd

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/scottpeterman/pathfinderssh/internal/appinstall"
	"github.com/scottpeterman/pathfinderssh/internal/mspauth"
	"github.com/scottpeterman/pathfinderssh/internal/ui"
)

// Options configures a non-interactive install from the command line.
type Options struct {
	Setup  string
	Enroll bool
	Home   string
}

// Run copies the tool bundle to AppData, creates shortcuts, and optionally configures sign-in.
func Run(opts Options) (destExe string, err error) {
	dest, copied, err := appinstall.Ensure()
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
	if !opts.Enroll {
		fmt.Println("Run Pathfinder or pfinstall -install-gui to finish", setup, "sign-in setup.")
		return nil
	}
	auth := mspauth.NewAuthenticator(home)
	p, ok := mspauth.ParseSetupMode(setup)
	if !ok {
		return fmt.Errorf("unknown setup mode %q", setup)
	}
	enroll := mspauth.Enrollment{Provider: p}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	if _, _, err := auth.EnrollAndVerify(ctx, enroll); err != nil {
		return fmt.Errorf("enroll %s: %w", setup, err)
	}
	fmt.Println("Sign-in setup complete for", setup)
	return nil
}
