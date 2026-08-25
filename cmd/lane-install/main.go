// lane-install — standalone SecureCRT companion installer.
// Pathfinder stays a separate product; this does not rewrite CRT from Pathfinder.
//
//	lane-install.exe                 graphical wizard (install or update)
//	lane-install.exe -install        CLI install / rewrite SecureCRT sessions
//	lane-install.exe -update         same as -install
//	lane-install.exe -install -mode mixed|forticlient|auvik
//	lane-install.exe -uninstall
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/scottpeterman/pathfinderssh/internal/auvik"
	"github.com/scottpeterman/pathfinderssh/internal/crtapp"
	"github.com/scottpeterman/pathfinderssh/internal/crtbridge"
	"github.com/scottpeterman/pathfinderssh/internal/crtimport"
	"github.com/scottpeterman/pathfinderssh/internal/fortivpn"
	"github.com/scottpeterman/pathfinderssh/internal/lanectl"
	"github.com/scottpeterman/pathfinderssh/internal/ui"
	"github.com/scottpeterman/pathfinderssh/internal/vpnprov"
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
		doInstall     = flag.Bool("install", false, "copy binaries, rewrite CRT sessions, start agent")
		doUpdate      = flag.Bool("update", false, "same as -install: refresh binaries and update SecureCRT sessions")
		doInstallGUI  = flag.Bool("install-gui", false, "graphical installer")
		doUninstall   = flag.Bool("uninstall", false, "restore CRT sessions and remove the companion")
		mode          = flag.String("mode", "", "mixed, forticlient, or auvik")
		auvikUser     = flag.String("auvik-user", "", "Auvik API username")
		auvikKey      = flag.String("auvik-key", "", "Auvik API key")
		auvikBase     = flag.String("auvik-base", "", "Auvik API base URL")
		vpnBin        = flag.String("vpn-bin", "", "FortiVPN.exe path")
		vpnTools      = flag.String("vpn-tools", "", "FortiSSLVPNclient.exe path")
		vpnDefault    = flag.String("vpn-default", "", "default FortiClient tunnel name")
		vpnMap        = flag.String("vpn-map", "", "Folder=Tunnel lines; everything under the folder uses that VPN")
		auvikMap      = flag.String("auvik-map", "", "Folder=AuvikTenant lines; everything under the folder uses that tenant")
		customer      = flag.String("customer-root", "", "SecureCRT customer folder name")
		from          = flag.String("from", "", "folder containing lane-crt.exe")
		doListVPNs    = flag.Bool("list-vpns", false, "print FortiClient, WireGuard, and Zscaler tunnels on this PC")
		doListTenants = flag.Bool("list-tenants", false, "print Auvik tenants from the API")
	)
	flag.Parse()

	if *doListTenants {
		s := loadOrPrefill()
		if *auvikUser != "" {
			s.AuvikUser = *auvikUser
		}
		if *auvikKey != "" {
			s.AuvikKey = *auvikKey
		}
		if *auvikBase != "" {
			s.AuvikBase = *auvikBase
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		tenants, err := auvik.New(s.AuvikUser, s.AuvikKey, s.AuvikBase).ListTenants(ctx)
		cancel()
		if err != nil {
			fmt.Fprintf(os.Stderr, "list-tenants: %v\n", err)
			os.Exit(1)
		}
		if len(tenants) == 0 {
			fmt.Println("(no tenants)")
			return
		}
		for _, t := range tenants {
			fmt.Println(t.Name)
		}
		return
	}

	if *doListVPNs {
		s := loadOrPrefill()
		if *vpnBin != "" {
			s.VPNBin = *vpnBin
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		items := vpnprov.ListAll(ctx, vpnprov.Bins{FortiBin: s.VPNBin, WireGuard: s.WGBin, Zscaler: s.ZSABin})
		cancel()
		if len(items) == 0 {
			fmt.Println("(no tunnels)")
			return
		}
		for _, it := range items {
			fmt.Println(it.Label)
		}
		return
	}

	if *doUninstall {
		if err := uninstallCRT(); err != nil {
			fmt.Fprintf(os.Stderr, "uninstall: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Restored SecureCRT sessions. Removed", crtapp.Root())
		return
	}

	cli := *doInstall || *doUpdate
	gui := *doInstallGUI || (!cli && len(os.Args) == 1)
	if cli {
		s := loadOrPrefill()
		if strings.TrimSpace(*mode) != "" {
			s.Mode = *mode
		}
		if *auvikUser != "" {
			s.AuvikUser = *auvikUser
		}
		if *auvikKey != "" {
			s.AuvikKey = *auvikKey
		}
		if *auvikBase != "" {
			s.AuvikBase = *auvikBase
		}
		if *vpnBin != "" {
			s.VPNBin = *vpnBin
		}
		if *vpnTools != "" {
			s.VPNTools = *vpnTools
		}
		if *vpnDefault != "" {
			s.VPNDefault = *vpnDefault
		}
		if *vpnMap != "" {
			s.VPNTunnels = crtbridge.ParseTunnelLines(*vpnMap)
		}
		if *auvikMap != "" {
			s.AuvikTenants = crtbridge.ParseTunnelLines(*auvikMap)
		}
		if *customer != "" {
			s.CustomerRoot = *customer
		}
		s.Normalize()
		rep, err := runInstall(*from, s)
		printReport(rep)
		if err != nil {
			fmt.Fprintf(os.Stderr, "install: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if gui {
		runGUI()
		return
	}
	fmt.Fprintln(os.Stderr, "lane-install: use -install, -install-gui, -uninstall, or -version")
	os.Exit(2)
}

func handleEarlyArgs(args []string) (bool, int) {
	for _, a := range args {
		switch a {
		case "-version", "--version":
			fmt.Println("lane-install", version)
			return true, 0
		case "-help", "--help", "-h":
			fmt.Println("lane-install — install Lane (standalone SecureCRT companion)")
			fmt.Println()
			fmt.Println("  lane-install.exe                         graphical wizard (install or update)")
			fmt.Println("  lane-install.exe -install -mode mixed    Auvik + FortiClient only where folders are mapped")
			fmt.Println("  lane-install.exe -update                 refresh binaries and rewrite SecureCRT sessions")
			fmt.Println("  lane-install.exe -install -mode forticlient  mapped CRT folders (no Auvik)")
			fmt.Println("  lane-install.exe -list-vpns")
			fmt.Println("  lane-install.exe -list-tenants")
			fmt.Println("  lane-install.exe -install -mode auvik")
			fmt.Println("  lane-install.exe -uninstall")
			return true, 0
		}
	}
	return false, 0
}

func wantsCLI(args []string) bool {
	for _, a := range args {
		switch a {
		case "-install", "-update", "-uninstall", "-version", "--version", "-help", "-h", "--help", "-list-vpns", "-list-tenants":
			return true
		}
	}
	return false
}

func loadOrPrefill() crtbridge.Settings {
	home := crtapp.Home()
	s, err := crtbridge.LoadSettings(home)
	if err != nil {
		s = crtbridge.Settings{Mode: crtbridge.AutoMixed}
	}
	if s.AuvikUser == "" {
		if ps, err := ui.LoadSettings(ui.SettingsPath()); err == nil {
			s.AuvikUser = ps.AuvikUsername
			s.AuvikKey = ps.AuvikAPIKey
			s.AuvikBase = ps.AuvikBaseURL
			if s.TunnelBin == "" {
				s.TunnelBin = ps.AuvikTunnelPath
			}
		}
	}
	if s.VPNBin == "" {
		s.VPNBin = fortivpn.DefaultBin()
	}
	if s.VPNTools == "" {
		s.VPNTools = fortivpn.DefaultTools()
	}
	if s.WGBin == "" {
		s.WGBin = vpnprov.DefaultWireGuardBin()
	}
	if s.ZSABin == "" {
		s.ZSABin = vpnprov.DefaultZscalerBin()
	}
	if s.CRTConfig == "" {
		s.CRTConfig = crtimport.DefaultConfig()
	}
	s.Normalize()
	return s
}

func runInstall(from string, s crtbridge.Settings) (crtbridge.Report, error) {
	if from == "" {
		if exe, err := os.Executable(); err == nil {
			from = filepath.Dir(exe)
		}
	}
	if crtbridge.SessionsDir(s.CRTConfig) == "" {
		return crtbridge.Report{}, fmt.Errorf("SecureCRT Config\\Sessions not found — install VanDyke SecureCRT first")
	}
	_ = crtbridge.StopAgent()
	time.Sleep(400 * time.Millisecond)
	if err := crtapp.CopyBundle(from); err != nil {
		return crtbridge.Report{}, err
	}
	if s.TunnelBin == "" && fileExists(crtapp.TunnelExe()) {
		s.TunnelBin = crtapp.TunnelExe()
	}
	home := crtapp.Home()
	_ = crtbridge.MigrateLegacyState(home)
	if err := crtbridge.SaveSettings(home, s); err != nil {
		return crtbridge.Report{}, err
	}
	opts := crtbridge.Options{
		AppHome:      home,
		CRTConfig:    s.CRTConfig,
		CustomerRoot: s.CustomerRoot,
		AuvikUser:    s.AuvikUser,
		AuvikKey:     s.AuvikKey,
		AuvikBase:    s.AuvikBase,
		TunnelBin:    s.TunnelBin,
		AgentExe:     crtapp.AgentExe(),
		Settings:     s,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	rep, err := crtbridge.AfterInstall(ctx, opts)
	if _, cerr := lanectl.Create(ctx, lanectl.CreateRequest{SSH: true, Putty: true, CRT: false}); cerr != nil {
		rep.Errors = append(rep.Errors, "OpenSSH/PuTTY: "+cerr.Error())
	}
	if inst := crtapp.InstallerExe(); fileExists(inst) {
		_ = crtapp.CreateShortcuts(inst)
	}
	return rep, err
}

func uninstallCRT() error {
	_ = crtbridge.StopAgent()
	home := crtapp.Home()
	if err := crtbridge.Uninstall(home); err != nil && !os.IsNotExist(err) {
		return err
	}
	crtapp.RemoveShortcuts()
	return os.RemoveAll(crtapp.Root())
}

func printReport(rep crtbridge.Report) {
	fmt.Printf("Mode: %s\nCustomer folder: %s\n", rep.Mode, rep.CustomerRoot)
	if rep.BackupDir != "" {
		fmt.Printf("Backup: %s\n", rep.BackupDir)
	}
	fmt.Printf("Localhost proxy: %d\nStandard SSH: %d\nSkipped: %d\n", rep.Tunnelled, rep.Direct, rep.Skipped)
	for _, e := range rep.Errors {
		fmt.Printf("warning: %s\n", e)
	}
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}
