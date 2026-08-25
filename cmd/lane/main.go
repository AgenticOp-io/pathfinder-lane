// lane is the cross-platform last-mile CLI: map folders once, then each
// engineer keeps the SSH client they already use (OpenSSH, SecureCRT, PuTTY).
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/scottpeterman/pathfinderssh/internal/crtapp"
	"github.com/scottpeterman/pathfinderssh/internal/crtbridge"
	"github.com/scottpeterman/pathfinderssh/internal/lanectl"
	"github.com/scottpeterman/pathfinderssh/internal/vpnprov"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 || isHelp(os.Args[1]) {
		usage()
		if len(os.Args) < 2 {
			os.Exit(2)
		}
		return
	}
	cmd := os.Args[1]
	args := os.Args[2:]
	if cmd == "create" && len(args) > 0 {
		switch strings.ToLower(args[0]) {
		case "all":
			cmd, args = "create-all", args[1:]
		case "ssh", "crt", "putty":
			cmd, args = "create-"+strings.ToLower(args[0]), args[1:]
		}
	}
	var err error
	switch cmd {
	case "version", "-version", "--version":
		fmt.Println(version)
	case "status":
		err = cmdStatus()
	case "list":
		err = cmdList()
	case "list-vpns":
		err = cmdListVPNs()
	case "map":
		err = cmdMap()
	case "setup":
		err = cmdSetup()
	case "bind":
		err = cmdBind(args)
	case "unbind":
		err = cmdUnbind(args)
	case "map-set":
		err = cmdMapSet(args, false)
	case "map-auvik":
		err = cmdMapSet(args, true)
	case "create-all":
		err = cmdCreate(args, true, true, true)
	case "create-ssh":
		err = cmdCreate(args, true, false, false)
	case "create-crt":
		err = cmdCreate(args, false, true, false)
	case "create-putty":
		err = cmdCreate(args, false, false, true)
	case "serve":
		err = cmdServe()
	case "proxy":
		err = cmdProxy(args)
	case "ssh":
		err = cmdSSH(args)
	case "ensure":
		err = cmdEnsure(args)
	case "restore-putty":
		n, e := lanectl.RestorePutty(crtapp.Home())
		if e == nil {
			fmt.Printf("Restored %d PuTTY sessions\n", n)
		}
		err = e
	case "autostart":
		err = cmdAutostart(args)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", cmd)
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "lane: %v\n", err)
		os.Exit(1)
	}
}

func isHelp(s string) bool {
	return s == "-h" || s == "-help" || s == "--help" || s == "help"
}

func usage() {
	fmt.Fprintf(os.Stderr, `lane %s — keep using the SSH client you already have

  lane setup                     map VPNs + bind hosts, then write every client
  lane bind fw-01 Acme           this PuTTY/ssh name belongs to customer Acme
  lane map-set Acme TARGET       Acme → Forti / wireguard:name / zscaler:zpa
  lane create-all                rewrite OpenSSH + PuTTY + SecureCRT

Then open the same session as today:

  ssh acme-core                    OpenSSH — no daemon
  PuTTY saved session              local proxy — no daemon
  Pathfinder tree                  VPN comes up on Connect
  SecureCRT session                needs: lane serve (Windows: lane-crt at logon)

  lane status | list | list-vpns | map | unbind NAME
  lane create ssh|crt|putty
  lane ssh ALIAS | serve | ensure TARGET | restore-putty | autostart [off]
`, version)
}

func cmdStatus() error {
	opts := lanectl.BridgeOptions("")
	cfg := opts.Settings
	fmt.Printf("mode: %s\n", cfg.Mode)
	fmt.Printf("home: %s\n", opts.AppHome)
	fmt.Printf("bin:  %s\n", lanectl.LaneBin())
	if p := lanectl.PATHStatus(); p != "" {
		fmt.Printf("path: %s\n", p)
	} else {
		fmt.Println("path: lane not on PATH yet — open a new shell after setup")
	}
	fmt.Printf("vpn maps: %d   auvik maps: %d\n", len(cfg.VPNTunnels), len(cfg.AuvikTenants))
	hosts := lanectl.Discover(cfg, opts.AppHome, opts.CRTConfig)
	mapped, skipped := lanectl.FilterMapped(cfg, hosts)
	fmt.Printf("hosts: %d mapped, %d unmapped (left as-is)\n", len(mapped), len(skipped))
	if crtbridge.SessionsDir(opts.CRTConfig) != "" {
		fmt.Println("SecureCRT: found — create-all will rewrite mapped sessions; run lane serve")
	} else {
		fmt.Println("SecureCRT: not found")
	}
	fmt.Println("OpenSSH: ssh <alias> after create-all (no daemon)")
	fmt.Println("PuTTY: local proxy after create-all (no daemon)")
	return nil
}

func cmdList() error {
	opts := lanectl.BridgeOptions("")
	hosts := lanectl.Discover(opts.Settings, opts.AppHome, opts.CRTConfig)
	mapped, _ := lanectl.FilterMapped(opts.Settings, hosts)
	seen := map[string]bool{}
	for _, h := range mapped {
		seen[h.Host+":"+strconv.Itoa(h.Port)] = true
	}
	for _, h := range hosts {
		mark := "skip"
		if seen[h.Host+":"+strconv.Itoa(h.Port)] {
			mark = "map "
		}
		fmt.Printf("%s  %-10s  %-24s  %s:%d  folder=%s\n", mark, h.Source, h.Alias, h.Host, h.Port, h.Folder)
	}
	return nil
}

func cmdListVPNs() error {
	opts := lanectl.BridgeOptions("")
	s := opts.Settings
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	items := vpnprov.ListAll(ctx, vpnprov.Bins{FortiBin: s.VPNBin, WireGuard: s.WGBin, Zscaler: s.ZSABin})
	if len(items) == 0 {
		fmt.Println("(no tunnels)")
		return nil
	}
	for _, it := range items {
		fmt.Println(it.Label)
	}
	return nil
}

func cmdMap() error {
	s, err := crtbridge.LoadSettings(crtapp.Home())
	if err != nil {
		return err
	}
	if len(s.VPNTunnels) == 0 && len(s.AuvikTenants) == 0 {
		fmt.Println("(no maps)  lane map-set FOLDER TARGET")
		return nil
	}
	if len(s.VPNTunnels) > 0 {
		fmt.Println("# VPN")
		fmt.Print(crtbridge.FormatTunnelLines(s.VPNTunnels))
	}
	if len(s.AuvikTenants) > 0 {
		fmt.Println("# Auvik")
		fmt.Print(crtbridge.FormatTunnelLines(s.AuvikTenants))
	}
	return nil
}

func cmdMapSet(args []string, auvik bool) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: lane map-set FOLDER TARGET")
	}
	folder, target := args[0], strings.Join(args[1:], " ")
	if err := lanectl.MapSet("", folder, target, auvik); err != nil {
		return err
	}
	kind := "vpn"
	if auvik {
		kind = "auvik"
	}
	fmt.Printf("%s  %s = %s\n", kind, folder, target)
	return nil
}

func cmdBind(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("usage: lane bind NAME FOLDER")
	}
	if err := lanectl.BindHost("", args[0], strings.Join(args[1:], " ")); err != nil {
		return err
	}
	fmt.Printf("bind %s → %s\n", args[0], strings.Join(args[1:], " "))
	return nil
}

func cmdUnbind(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: lane unbind NAME")
	}
	return lanectl.UnbindHost("", args[0])
}

func cmdSetup() error {
	if dest, err := lanectl.InstallSelf(); err != nil {
		fmt.Fprintf(os.Stderr, "note: %v\n", err)
	} else {
		fmt.Println("lane installed at", dest)
		if p := lanectl.PATHStatus(); p != "" {
			fmt.Println("on PATH as", p)
		}
	}
	if !lanectl.InteractiveStdin() {
		fmt.Println("Non-interactive: bind hosts with `lane bind NAME FOLDER`, then create-all.")
		return cmdCreate(nil, true, true, true)
	}
	home := crtapp.Home()
	in := bufio.NewReader(os.Stdin)
	prompt := func(q string) string {
		fmt.Print(q)
		line, _ := in.ReadString('\n')
		return strings.TrimSpace(line)
	}

	s, err := crtbridge.LoadSettings(home)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	items := vpnprov.ListAll(ctx, vpnprov.Bins{FortiBin: s.VPNBin, WireGuard: s.WGBin, Zscaler: s.ZSABin})
	cancel()
	if len(s.VPNTunnels) == 0 && len(items) > 0 {
		fmt.Println("Name each tunnel with the customer you use in CRT/Pathfinder (empty = skip):")
		for _, it := range items {
			name := prompt("  " + it.Label + "  customer: ")
			if name == "" {
				continue
			}
			if err := lanectl.MapSet(home, name, it.Label, false); err != nil {
				return err
			}
		}
		s, _ = crtbridge.LoadSettings(home)
	}

	opts := lanectl.BridgeOptions(home)
	hosts := lanectl.Discover(opts.Settings, home, opts.CRTConfig)
	_, skipped := lanectl.FilterMapped(opts.Settings, hosts)
	if len(skipped) > 0 {
		fmt.Println("These sessions have no customer yet. Type a mapped customer, or skip:")
		if len(s.VPNTunnels) > 0 {
			fmt.Print("  mapped: ")
			first := true
			for k := range s.VPNTunnels {
				if !first {
					fmt.Print(", ")
				}
				fmt.Print(k)
				first = false
			}
			fmt.Println()
		}
		for _, h := range skipped {
			ans := prompt(fmt.Sprintf("  %s  %s  %s:%d  customer: ", h.Source, h.Alias, h.Host, h.Port))
			if ans == "" || strings.EqualFold(ans, "skip") {
				continue
			}
			key := h.Alias
			if key == "" {
				key = h.Name
			}
			if err := lanectl.BindHost(home, key, ans); err != nil {
				return err
			}
		}
	}
	return cmdCreate(nil, true, true, true)
}

func cmdCreate(args []string, ssh, crt, putty bool) error {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	via := fs.String("via", "auto", "auto (OpenSSH/PuTTY proxy, CRT agent), proxy, or agent")
	if err := fs.Parse(args); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	rep, err := lanectl.Create(ctx, lanectl.CreateRequest{
		Via:   *via,
		SSH:   ssh,
		CRT:   crt,
		Putty: putty,
	})
	printCreate(rep)
	return err
}

func printCreate(rep lanectl.CreateReport) {
	if rep.SSHPath != "" {
		fmt.Printf("OpenSSH config: %s (%d hosts)\n", rep.SSHPath, rep.SSHHosts)
	}
	if len(rep.Aliases) > 0 && len(rep.Aliases) <= 20 {
		fmt.Println("  ssh " + strings.Join(rep.Aliases, "\n  ssh "))
	} else if len(rep.Aliases) > 20 {
		fmt.Printf("  %d aliases — lane list\n", len(rep.Aliases))
	}
	if rep.CRTRan {
		fmt.Printf("SecureCRT: proxy=%d direct=%d skipped=%d\n", rep.CRT.Tunnelled, rep.CRT.Direct, rep.CRT.Skipped)
	}
	if rep.PuttyRewritten > 0 || rep.PuttySkipped > 0 {
		fmt.Printf("PuTTY: rewritten=%d skipped=%d\n", rep.PuttyRewritten, rep.PuttySkipped)
	}
	fmt.Printf("mapped=%d unmapped=%d via=%s\n", rep.Mapped, rep.Skipped, rep.Via)
	for _, n := range rep.Notes {
		fmt.Println(n)
	}
}

func cmdServe() error {
	opts := lanectl.BridgeOptions("")
	fmt.Printf("lane serve %s (mode %s)\n", version, opts.Settings.Mode)
	return crtbridge.RunAgent(context.Background(), opts)
}

func cmdProxy(args []string) error {
	fs := flag.NewFlagSet("proxy", flag.ContinueOnError)
	folder := fs.String("folder", "", "mapped customer folder")
	host := fs.String("host", "", "device host")
	port := fs.Int("port", 22, "ssh port")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *host == "" {
		return fmt.Errorf("proxy: -host required")
	}
	ctx := context.Background()
	return lanectl.RunProxy(ctx, crtapp.Home(), *folder, *host, *port, os.Stdin, os.Stdout)
}

func cmdSSH(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: lane ssh ALIAS [ssh-args]")
	}
	bin, err := exec.LookPath("ssh")
	if err != nil {
		return fmt.Errorf("openssh ssh not on PATH")
	}
	cmd := exec.Command(bin, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}

func cmdAutostart(args []string) error {
	off := len(args) > 0 && (args[0] == "off" || args[0] == "disable" || args[0] == "stop")
	if off {
		if err := crtbridge.DisableAutostart(); err != nil {
			return err
		}
		fmt.Println("autostart off")
		return nil
	}
	exe := lanectl.LaneBin()
	if p := crtapp.AgentExe(); fileExistsAgent(p) {
		exe = p
	}
	if err := crtbridge.EnableAutostart(exe); err != nil {
		return err
	}
	_ = crtbridge.RestartAgent(exe)
	fmt.Println("autostart on:", exe)
	return nil
}

func fileExistsAgent(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func cmdEnsure(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: lane ensure TARGET")
	}
	opts := lanectl.BridgeOptions("")
	s := opts.Settings
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	return vpnprov.Ensure(ctx, vpnprov.Bins{
		FortiBin: s.VPNBin, FortiTools: s.VPNTools, WireGuard: s.WGBin, Zscaler: s.ZSABin,
	}, strings.Join(args, " "))
}
