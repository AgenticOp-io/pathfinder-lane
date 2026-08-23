// pfinstall — PathfinderSSH MSP installer (primary entry point).
//
// Graphical (double-click):
//   pfinstall.exe
//
// Graphical (explicit):
//   pfinstall.exe -install-gui
//   pfinstall.exe -install-gui -setup o365
//
// Command line:
//   pfinstall.exe -install
//   pfinstall.exe -install -setup solo
//   pfinstall.exe -update
//   pfinstall.exe -from C:\path\to\dist\windows -install
//   pfinstall.exe -uninstall
//   pfinstall.exe -version
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	"github.com/scottpeterman/pathfinderssh/internal/appinstall"
	"github.com/scottpeterman/pathfinderssh/internal/installcmd"
	"github.com/scottpeterman/pathfinderssh/internal/ui"
	"github.com/scottpeterman/pathfinderssh/internal/winconsole"
)

var version = "dev"

func main() {
	if early, code := handleEarlyArgs(os.Args[1:]); early {
		if code != 0 {
			os.Exit(code)
		}
		return
	}

	if wantsCLI(os.Args[1:]) {
		winconsole.AttachOrAllocate()
	}

	var (
		doInstall    = flag.Bool("install", false, "copy bundle to AppData and create shortcuts")
		doInstallGUI = flag.Bool("install-gui", false, "graphical install wizard")
		doUpdate     = flag.Bool("update", false, "reinstall / refresh binaries in AppData")
		doUninstall  = flag.Bool("uninstall", false, "remove AppData install and shortcuts")
		setup        = flag.String("setup", "", "access mode: solo, o365, google")
		doEnroll     = flag.Bool("enroll", false, "complete cloud sign-in during CLI install")
		bundleFrom   = flag.String("from", "", "folder containing pathfinder.exe and bundled tools (default: beside this exe)")
		showVersion  = flag.Bool("version", false, "print version and exit")
	)
	flag.Parse()

	if *showVersion {
		fmt.Println("pfinstall", version)
		return
	}

	if *doUninstall {
		if err := appinstall.Uninstall(); err != nil {
			fmt.Fprintf(os.Stderr, "uninstall: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Removed", appinstall.Root())
		return
	}

	setupMode := strings.TrimSpace(*setup)
	cli := *doInstall || *doUpdate || setupMode != "" || *doEnroll
	// Double-click (no args) opens the wizard; any other argv is CLI unless -install-gui.
	gui := *doInstallGUI || (!cli && len(os.Args) == 1)

	if cli {
		_, err := installcmd.Run(installcmd.Options{
			Setup:     setupMode,
			Enroll:    *doEnroll,
			Home:      ui.GetAppHome(),
			BundleDir: *bundleFrom,
			Update:    *doUpdate,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "install: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if gui {
		runInstallGUI(setupMode)
		return
	}

	fmt.Fprintln(os.Stderr, "pfinstall: use -install, -install-gui, -update, -uninstall, or -version")
	flag.PrintDefaults()
	os.Exit(2)
}

func handleEarlyArgs(args []string) (handled bool, exitCode int) {
	for _, a := range args {
		switch a {
		case "-version", "--version":
			fmt.Println("pfinstall", version)
			return true, 0
		case "-help", "--help", "-h":
			fmt.Println("pfinstall — install PathfinderSSH MSP (GUI or CLI)")
			fmt.Println()
			fmt.Println("  pfinstall.exe                    graphical wizard (double-click)")
			fmt.Println("  pfinstall.exe -install           copy bundle to AppData")
			fmt.Println("  pfinstall.exe -update            refresh installed binaries")
			fmt.Println("  pfinstall.exe -install-gui       graphical wizard")
			fmt.Println("  pfinstall.exe -uninstall         remove install")
			fmt.Println("  pfinstall.exe -setup solo|o365|google")
			fmt.Println("  pfinstall.exe -enroll            cloud sign-in during CLI install")
			fmt.Println("  pfinstall.exe -from <dir>        bundle folder")
			fmt.Println("  pfinstall.exe -version")
			return true, 0
		}
	}
	return false, 0
}

func wantsCLI(args []string) bool {
	for _, a := range args {
		switch a {
		case "-install", "-update", "-uninstall", "-version", "--version", "-help", "-h", "--help":
			return true
		case "-setup", "-enroll", "-from":
			return true
		case "-install-gui":
			return false
		}
		if strings.HasPrefix(a, "-setup=") || strings.HasPrefix(a, "-from=") {
			return true
		}
	}
	return false
}

func runInstallGUI(setupPreset string) {
	a := app.NewWithID("com.pathfinder.pfinstall")
	ui.LoadUserThemes()
	ui.ApplyInstallerTheme(a)
	if icon := ui.AppIcon(); icon != nil {
		a.SetIcon(icon)
	}
	w := a.NewWindow("Install PathfinderSSH MSP")
	w.Resize(fyne.NewSize(720, 580))
	w.CenterOnScreen()

	home := ui.GetAppHome()
	_ = setupPreset
	ui.ShowInstallWizard(w, ui.InstallWizardOptions{
		Version: version,
		Home:    home,
	})
	w.ShowAndRun()
}
