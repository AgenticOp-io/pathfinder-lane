// lane-crt is the standalone SecureCRT companion (not Pathfinder).
//
//	lane-crt -install   backup CRT customer folder, apply templates, start agent
//	lane-crt -uninstall restore original SSH host/port, remove autostart
//	lane-crt -sync      re-check customers and update CRT sessions once
//	lane-crt            run the background agent (default after install)
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/scottpeterman/pathfinderssh/internal/crtapp"
	"github.com/scottpeterman/pathfinderssh/internal/crtbridge"
	"github.com/scottpeterman/pathfinderssh/internal/crtimport"
)

var version = "0.93"

func main() {
	var (
		doInstall   = flag.Bool("install", false, "backup SecureCRT customer folder, apply templates, start agent")
		doUninstall = flag.Bool("uninstall", false, "restore original SSH sessions and remove autostart")
		doSync      = flag.Bool("sync", false, "re-check customers and update CRT sessions once")
		customer    = flag.String("customer-root", "", "SecureCRT top-level folder that is the customer list")
		crtConfig   = flag.String("crt-config", "", "VanDyke Config directory (default AppData\\Roaming\\VanDyke\\Config)")
		mode        = flag.String("mode", "", "automation: mixed, forticlient, or auvik")
	)
	flag.Parse()

	log.SetFlags(log.Ltime | log.Lmicroseconds)
	teeLog()

	home := crtapp.Home()
	_ = crtbridge.MigrateLegacyState(home)
	opts := optionsFromSettings(home)
	if *customer != "" {
		opts.CustomerRoot = *customer
		opts.Settings.CustomerRoot = *customer
	}
	if *crtConfig != "" {
		opts.CRTConfig = *crtConfig
		opts.Settings.CRTConfig = *crtConfig
	} else if opts.CRTConfig == "" {
		opts.CRTConfig = crtimport.DefaultConfig()
	}
	if strings.TrimSpace(*mode) != "" {
		opts.Settings.Mode = *mode
		opts.Settings.Normalize()
	}
	opts.AgentExe = agentExe()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	switch {
	case *doUninstall:
		if err := crtbridge.Uninstall(home); err != nil {
			log.Fatal(err)
		}
		fmt.Println("SecureCRT sessions restored to standard SSH. CRT bridge autostart removed.")
		return
	case *doInstall:
		if err := crtbridge.SaveSettings(home, opts.Settings); err != nil {
			log.Fatal(err)
		}
		rep, err := crtbridge.AfterInstall(ctx, opts)
		printReport(rep)
		if err != nil {
			log.Fatal(err)
		}
		return
	case *doSync:
		rep, err := crtbridge.Sync(ctx, opts)
		printReport(rep)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	hideConsole()
	runCtx, stop := context.WithCancel(context.Background())
	defer stop()
	log.Printf("lane-crt %s agent starting (mode %s)", version, opts.Settings.Mode)
	if err := crtbridge.RunAgent(runCtx, opts); err != nil && err != context.Canceled {
		log.Fatal(err)
	}
}

func optionsFromSettings(home string) crtbridge.Options {
	s, err := crtbridge.LoadSettings(home)
	if err != nil {
		s = crtbridge.Settings{Mode: crtbridge.AutoMixed}
	}
	return crtbridge.Options{
		AppHome:      home,
		CRTConfig:    s.CRTConfig,
		CustomerRoot: s.CustomerRoot,
		AuvikUser:    s.AuvikUser,
		AuvikKey:     s.AuvikKey,
		AuvikBase:    s.AuvikBase,
		TunnelBin:    s.TunnelBin,
		Settings:     s,
	}
}

func agentExe() string {
	if p := crtapp.AgentExe(); fileExists(p) {
		return p
	}
	if exe, err := os.Executable(); err == nil {
		return exe
	}
	return crtapp.ExeName("lane-crt")
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func printReport(rep crtbridge.Report) {
	fmt.Printf("Mode: %s\n", rep.Mode)
	fmt.Printf("Customer folder: %s\n", rep.CustomerRoot)
	if rep.BackupDir != "" {
		fmt.Printf("Backup: %s\n", rep.BackupDir)
	}
	fmt.Printf("Localhost proxy (Auvik and/or VPN): %d\nStandard SSH: %d\nSkipped: %d\n",
		rep.Tunnelled, rep.Direct, rep.Skipped)
	for _, e := range rep.Errors {
		fmt.Printf("warning: %s\n", e)
	}
}

func teeLog() {
	dir := crtapp.LogsDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	f, err := os.OpenFile(filepath.Join(dir, "crt-bridge.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	log.SetOutput(io.MultiWriter(os.Stderr, f))
}
