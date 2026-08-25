package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/scottpeterman/pathfinderssh/internal/auvik"
	"github.com/scottpeterman/pathfinderssh/internal/crtapp"
	"github.com/scottpeterman/pathfinderssh/internal/crtbridge"
	"github.com/scottpeterman/pathfinderssh/internal/ui"
	"github.com/scottpeterman/pathfinderssh/internal/vpnprov"
)

const folderSkip = "(skip)"

type vpnPick struct {
	tunnel string
	sel    *widget.Select
}

func runGUI() {
	a := app.NewWithID("io.agenticop.lane-install")
	ui.LoadUserThemes()
	ui.ApplyInstallerTheme(a)
	if icon := ui.AppIcon(); icon != nil {
		a.SetIcon(icon)
	}

	s := loadOrPrefill()
	probe := crtbridge.ProbeEnv(s.CRTConfig, crtapp.Home())
	updating := probe.Installed || probe.AgentPresent

	title := "Install Lane"
	if updating {
		title = "Update Lane"
	}
	w := a.NewWindow(title)
	w.Resize(fyne.NewSize(860, 940))
	w.CenterOnScreen()

	heroTitle := "Lane"
	heroSub := "Map what this PC already has (CRT folders, FortiClient tunnels, Auvik tenants) to what each customer needs. After that, opening a session is automatic."
	if updating {
		heroSub = "Refresh the maps and rewrite SecureCRT sessions. Runtime never guesses names."
	}
	titleLbl := widget.NewLabelWithStyle(heroTitle, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	subLbl := widget.NewLabel(heroSub)
	subLbl.Wrapping = fyne.TextWrapWord
	hero := container.NewVBox(titleLbl, subLbl)
	if ic := ui.AppIcon(); ic != nil {
		hero = container.NewHBox(widget.NewIcon(ic), container.NewPadded(container.NewVBox(titleLbl, subLbl)))
	}

	stepBar := widget.NewProgressBar()
	stepBar.SetValue(0)

	detect := widget.NewLabel(probeSummary(probe))
	detect.Wrapping = fyne.TextWrapWord

	mode := widget.NewRadioGroup([]string{
		"Mixed — Auvik and customer VPNs only where you map a folder",
		"VPN only — mapped FortiClient / WireGuard / Zscaler (no Auvik)",
		"Auvik only — mapped CRT folders (no customer VPN)",
	}, nil)
	switch s.Mode {
	case crtbridge.AutoFortiClient:
		mode.SetSelected("VPN only — mapped FortiClient / WireGuard / Zscaler (no Auvik)")
	case crtbridge.AutoAuvik:
		mode.SetSelected("Auvik only — mapped CRT folders (no customer VPN)")
	default:
		mode.SetSelected("Mixed — Auvik and customer VPNs only where you map a folder")
	}

	auvikUser := widget.NewEntry()
	auvikUser.SetText(s.AuvikUser)
	auvikKey := widget.NewPasswordEntry()
	auvikKey.SetText(s.AuvikKey)
	auvikBase := widget.NewEntry()
	auvikBase.SetText(s.AuvikBase)
	if auvikBase.Text == "" {
		auvikBase.SetPlaceHolder("https://auvikapi.us1.my.auvik.com")
	}
	vpnBin := widget.NewEntry()
	vpnBin.SetText(s.VPNBin)
	vpnTools := widget.NewEntry()
	vpnTools.SetText(s.VPNTools)
	vpnDefault := widget.NewEntry()
	vpnDefault.SetText(s.VPNDefault)
	vpnDefault.SetPlaceHolder("Leave blank unless unmapped folders should use one tunnel")
	vpnMap := widget.NewMultiLineEntry()
	vpnMap.SetText(crtbridge.FormatTunnelLines(s.VPNTunnels))
	vpnMap.SetPlaceHolder("Folder=wireguard:acme  or  Folder=zscaler:zpa  or  Folder=FortiTunnel")
	vpnMap.SetMinRowsVisible(4)
	customer := widget.NewEntry()
	customer.SetText(s.CustomerRoot)
	if customer.Text == "" {
		customer.SetText(probe.CustomerRoot)
	}
	customer.SetPlaceHolder("usually 3_Customers (detected if blank)")

	folders := crtbridge.ListCustomerFolders(probe.SessionsDir, strings.TrimSpace(customer.Text))
	folderOpts := append([]string{folderSkip}, folders...)
	var picks []*vpnPick
	vpnNote := widget.NewLabel("Reading installed VPNs (FortiClient, WireGuard, Zscaler)…")
	vpnNote.Wrapping = fyne.TextWrapWord
	vpnRows := container.NewVBox()
	rebuildVPNRows := func(items []vpnprov.Item, existing map[string]string) {
		picks = nil
		vpnRows.Objects = nil
		inv := map[string]string{}
		for folder, tun := range existing {
			if _, ok := inv[tun]; !ok {
				inv[tun] = folder
			}
		}
		for _, it := range items {
			sel := widget.NewSelect(folderOpts, nil)
			chosen := folderSkip
			if f := inv[it.Label]; f != "" {
				chosen = f
			} else if f := inv[it.Name]; f != "" {
				chosen = f
			} else if f := crtbridge.SuggestFolder(it.Name, folders); f != "" {
				chosen = f
			}
			sel.SetSelected(chosen)
			picks = append(picks, &vpnPick{tunnel: it.Label, sel: sel})
			vpnRows.Add(container.NewBorder(nil, nil, widget.NewLabel(it.Kind+" / "+it.Name), nil, sel))
		}
		vpnRows.Refresh()
	}
	refreshVPNs := func() {
		bin := strings.TrimSpace(vpnBin.Text)
		vpnNote.SetText("Reading installed VPNs (FortiClient, WireGuard, Zscaler)…")
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			items := vpnprov.ListAll(ctx, vpnprov.Bins{FortiBin: bin, WireGuard: s.WGBin, Zscaler: s.ZSABin})
			cancel()
			note := "Official CLIs only. Map each tunnel to a CRT folder — names do not have to match. WireGuard is wireguard:iface, Zscaler ZPA is zscaler:zpa (add zscaler:zpa:user for a partner tenant)."
			if len(items) == 0 {
				note = "No FortiClient, WireGuard, or Zscaler CLI tunnels found. Type Folder=wireguard:name or Folder=zscaler:zpa below, or install the vendor client and Refresh."
			}
			fyne.Do(func() {
				vpnNote.SetText(note)
				rebuildVPNRows(items, mergeVPNPicks(picks, vpnMap.Text))
			})
		}()
	}
	refreshBtn := widget.NewButton("Refresh VPN list", refreshVPNs)

	auvikMap := widget.NewMultiLineEntry()
	auvikMap.SetText(crtbridge.FormatTunnelLines(s.AuvikTenants))
	auvikMap.SetPlaceHolder("Folder=AuvikTenantOrDomain  (e.g. Nanook Wireless=nanook)")
	auvikMap.SetMinRowsVisible(4)
	var auvikPicks []*vpnPick
	auvikNote := widget.NewLabel("Enter Auvik username and API key, then Refresh tenant list. CRT folder names will not match Auvik domains — map them here.")
	auvikNote.Wrapping = fyne.TextWrapWord
	auvikRows := container.NewVBox()
	rebuildAuvikRows := func(names []string, existing map[string]string) {
		auvikPicks = nil
		auvikRows.Objects = nil
		inv := map[string]string{}
		for folder, tenant := range existing {
			if _, ok := inv[tenant]; !ok {
				inv[tenant] = folder
			}
		}
		for _, name := range names {
			sel := widget.NewSelect(folderOpts, nil)
			chosen := folderSkip
			if f := inv[name]; f != "" {
				chosen = f
			} else if f := crtbridge.SuggestFolder(name, folders); f != "" {
				chosen = f
			}
			sel.SetSelected(chosen)
			auvikPicks = append(auvikPicks, &vpnPick{tunnel: name, sel: sel})
			auvikRows.Add(container.NewBorder(nil, nil, widget.NewLabel(name), nil, sel))
		}
		auvikRows.Refresh()
	}
	pathfinderTM := func() auvik.TenantMap {
		tm, _ := auvik.LoadTenantMap(crtapp.Home())
		if legacy := crtapp.LegacyPathfinderHome(); legacy != "" && legacy != crtapp.Home() {
			if extra, err := auvik.LoadTenantMap(legacy); err == nil {
				if tm.Mappings == nil {
					tm.Mappings = map[string]string{}
				}
				if tm.Domains == nil {
					tm.Domains = map[string]string{}
				}
				for k, v := range extra.Mappings {
					if _, ok := tm.Mappings[k]; !ok {
						tm.Mappings[k] = v
					}
				}
				for k, v := range extra.Domains {
					if _, ok := tm.Domains[k]; !ok {
						tm.Domains[k] = v
					}
				}
			}
		}
		if tm.Mappings == nil {
			tm.Mappings = map[string]string{}
		}
		return tm
	}
	refreshAuvik := func() {
		user := strings.TrimSpace(auvikUser.Text)
		key := strings.TrimSpace(auvikKey.Text)
		base := strings.TrimSpace(auvikBase.Text)
		auvikNote.SetText("Reading Auvik tenants…")
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			c := auvik.New(user, key, base)
			tenants, err := c.ListTenants(ctx)
			names := make([]string, 0, len(tenants))
			for _, t := range tenants {
				if n := strings.TrimSpace(t.Name); n != "" {
					names = append(names, n)
				}
			}
			note := "Auvik tenants from the API. Pick the CRT folder each tenant covers — names do not have to match."
			if err != nil {
				names = nil
				note = err.Error()
			} else if len(names) == 0 {
				note = "Auvik returned no tenants. Check the API key, then Refresh."
			}
			fyne.Do(func() {
				existing := crtbridge.SeedAuvikTenants(mergeVPNPicks(auvikPicks, auvikMap.Text), pathfinderTM(), tenants)
				auvikNote.SetText(note)
				rebuildAuvikRows(names, existing)
			})
		}()
	}
	refreshAuvikBtn := widget.NewButton("Refresh Auvik tenant list", refreshAuvik)

	form := widget.NewForm(
		widget.NewFormItem("Automation", mode),
		widget.NewFormItem("Auvik username", auvikUser),
		widget.NewFormItem("Auvik API key", auvikKey),
		widget.NewFormItem("Auvik base URL", auvikBase),
		widget.NewFormItem("FortiVPN.exe", vpnBin),
		widget.NewFormItem("FortiClientTools", vpnTools),
		widget.NewFormItem("Default VPN tunnel", vpnDefault),
		widget.NewFormItem("CRT customer folder", customer),
	)
	mapBox := container.NewVBox(
		vpnNote,
		refreshBtn,
		vpnRows,
		widget.NewLabel("Extra or nested Folder=target lines (wireguard:name, zscaler:zpa, or a Forti connection name):"),
		vpnMap,
	)
	auvikBox := container.NewVBox(
		auvikNote,
		refreshAuvikBtn,
		auvikRows,
		widget.NewLabel("Extra or nested Folder=AuvikTenant lines (when the API list is empty, or for a subfolder):"),
		auvikMap,
	)

	status := widget.NewLabel("This installer updates SecureCRT session files when the add-on needs localhost proxies. Pathfinder stays a separate product.")
	status.Wrapping = fyne.TextWrapWord

	primaryLabel := "Install and update SecureCRT"
	if updating {
		primaryLabel = "Update SecureCRT sessions"
	}
	installBtn := widget.NewButtonWithIcon(primaryLabel, theme.DownloadIcon(), nil)
	installBtn.Importance = widget.HighImportance
	uninstallBtn := widget.NewButton("Uninstall", nil)
	cancelBtn := widget.NewButton("Cancel", func() { a.Quit() })

	if !probe.SecureCRTFound {
		installBtn.Disable()
		status.SetText("SecureCRT Config\\Sessions was not found. Install VanDyke SecureCRT, then run this installer again.")
	}

	bodySlot := container.NewMax()
	var phase int
	var lastRep crtbridge.Report

	collect := func() crtbridge.Settings {
		out := s
		sel := mode.Selected
		switch {
		case strings.HasPrefix(sel, "FortiClient"), strings.HasPrefix(sel, "VPN"):
			out.Mode = crtbridge.AutoFortiClient
		case strings.HasPrefix(sel, "Auvik"):
			out.Mode = crtbridge.AutoAuvik
		default:
			out.Mode = crtbridge.AutoMixed
		}
		out.AuvikUser = strings.TrimSpace(auvikUser.Text)
		out.AuvikKey = strings.TrimSpace(auvikKey.Text)
		out.AuvikBase = strings.TrimSpace(auvikBase.Text)
		out.VPNBin = strings.TrimSpace(vpnBin.Text)
		out.VPNTools = strings.TrimSpace(vpnTools.Text)
		out.VPNDefault = strings.TrimSpace(vpnDefault.Text)
		out.VPNTunnels = mergeVPNPicks(picks, vpnMap.Text)
		out.AuvikTenants = mergeVPNPicks(auvikPicks, auvikMap.Text)
		out.CustomerRoot = strings.TrimSpace(customer.Text)
		out.Normalize()
		return out
	}

	var showPhase func()
	showPhase = func() {
		switch phase {
		case 0:
			stepBar.SetValue(0)
			bodySlot.Objects = []fyne.CanvasObject{
				container.NewVScroll(container.NewVBox(
					hero,
					stepBar,
					widget.NewCard("SecureCRT on this PC", "", detect),
					widget.NewCard("Automation", "Setup is the map. After install, opening a session is automatic.", form),
					widget.NewCard("Customer VPNs → CRT folders", "FortiClient, WireGuard, and Zscaler ZPA. What they have → CRT folder. Opening a session switches the VPN automatically.", mapBox),
					widget.NewCard("Auvik tenants → CRT folders", "What they have (Auvik domain) → what they need (CRT folder). Unmapped folders stay on standard SSH.", auvikBox),
				)),
			}
			installBtn.Show()
			cancelBtn.Show()
			uninstallBtn.Show()
		case 1:
			stepBar.SetValue(0.5)
			working := "Updating SecureCRT sessions…"
			if !updating {
				working = "Installing and rewriting SecureCRT sessions…"
			}
			bodySlot.Objects = []fyne.CanvasObject{
				container.NewVBox(
					hero,
					stepBar,
					widget.NewCard("Working", working, container.NewVBox(
						widget.NewLabel("Stopping the old agent, copying binaries, rewriting session host/port, starting the agent."),
						widget.NewProgressBarInfinite(),
					)),
				),
			}
			installBtn.Hide()
			cancelBtn.Hide()
			uninstallBtn.Hide()
		case 2:
			stepBar.SetValue(1)
			msg := fmt.Sprintf("Mode: %s\nCustomer folder: %s\nLocalhost proxy: %d\nStandard SSH: %d\nSkipped: %d",
				lastRep.Mode, lastRep.CustomerRoot, lastRep.Tunnelled, lastRep.Direct, lastRep.Skipped)
			if lastRep.BackupDir != "" {
				msg += "\nBackup: " + lastRep.BackupDir
			}
			if len(lastRep.Errors) > 0 {
				msg += "\n\n" + strings.Join(lastRep.Errors, "\n")
			}
			closeBtn := widget.NewButton("Close", func() { a.Quit() })
			closeBtn.Importance = widget.HighImportance
			bodySlot.Objects = []fyne.CanvasObject{
				container.NewVBox(
					hero,
					stepBar,
					widget.NewCard("Ready", "Open the SSH client you already use. Mapped FortiClient / WireGuard / Zscaler / Auvik run automatically. OpenSSH and PuTTY do not need the agent; SecureCRT does (starts at logon).", container.NewVBox(
						widget.NewLabel(msg),
						widget.NewLabel("If SecureCRT was already open, start a new session (or restart SecureCRT) so it reads the updated .ini files."),
						closeBtn,
					)),
				),
			}
			installBtn.Hide()
			cancelBtn.Hide()
			uninstallBtn.Hide()
		}
		bodySlot.Refresh()
	}

	doWork := func(cfg crtbridge.Settings) {
		phase = 1
		showPhase()
		go func() {
			rep, err := runInstall("", cfg)
			fyne.Do(func() {
				if err != nil {
					phase = 0
					showPhase()
					status.SetText(err.Error())
					dialog.ShowError(err, w)
					return
				}
				lastRep = rep
				phase = 2
				showPhase()
				status.SetText("Done. The agent starts at logon and when you open a SecureCRT session.")
			})
		}()
	}

	installBtn.OnTapped = func() {
		cfg := collect()
		if !probe.SecureCRTFound {
			dialog.ShowError(fmt.Errorf("SecureCRT is not installed"), w)
			return
		}
		if probe.CRTRunning {
			dialog.ShowConfirm("SecureCRT is open",
				"Session files will still be updated on disk. After this finishes, open a new session (or restart SecureCRT) so it uses the localhost tunnels.",
				func(ok bool) {
					if ok {
						doWork(cfg)
					}
				}, w)
			return
		}
		doWork(cfg)
	}

	uninstallBtn.OnTapped = func() {
		dialog.ShowConfirm("Uninstall",
			"Restore original SecureCRT SSH hosts and remove Lane from AppData?\nBackups under ~/.lane are kept.",
			func(ok bool) {
				if !ok {
					return
				}
				_ = crtbridge.StopAgent()
				if err := uninstallCRT(); err != nil {
					dialog.ShowError(err, w)
					return
				}
				status.SetText("Uninstalled. SecureCRT sessions restored to standard SSH.")
				detect.SetText("Companion removed. SecureCRT sessions restored.")
				installBtn.SetText("Install and update SecureCRT")
				installBtn.Enable()
			}, w)
	}

	showPhase()
	refreshVPNs()
	if strings.TrimSpace(s.AuvikUser) != "" && strings.TrimSpace(s.AuvikKey) != "" {
		refreshAuvik()
	}
	footer := container.NewBorder(nil, nil, uninstallBtn, container.NewHBox(cancelBtn, installBtn), nil)
	w.SetContent(container.NewBorder(
		nil,
		container.NewVBox(container.NewPadded(status), container.NewPadded(footer)),
		nil, nil,
		container.NewPadded(bodySlot),
	))
	w.ShowAndRun()
}

func probeSummary(p crtbridge.Probe) string {
	var b strings.Builder
	if p.SecureCRTFound {
		b.WriteString("SecureCRT sessions: " + p.SessionsDir + "\n")
		if p.CustomerRoot != "" {
			b.WriteString("Customer folder: " + p.CustomerRoot + "\n")
		}
	} else {
		b.WriteString("SecureCRT was not found (expected %AppData%\\VanDyke\\Config\\Sessions).\n")
	}
	if p.Installed {
		b.WriteString("Existing add-on: this run will update binaries and rewrite sessions that need localhost proxies.\n")
	} else {
		b.WriteString("First install: backup the customer folder, then rewrite Auvik/FortiClient sessions to 127.0.0.1.\n")
	}
	if p.CRTRunning {
		b.WriteString("SecureCRT is currently running — new sessions pick up the change; restart CRT for already-open tabs.\n")
	}
	if p.AgentRunning {
		b.WriteString("The CRT agent is running and will be restarted so FortiClient auto-switch is live.\n")
	}
	return strings.TrimSpace(b.String())
}

func mergeVPNPicks(picks []*vpnPick, extra string) map[string]string {
	out := map[string]string{}
	for _, p := range picks {
		if p == nil || p.sel == nil {
			continue
		}
		f := strings.TrimSpace(p.sel.Selected)
		if f == "" || f == folderSkip {
			continue
		}
		out[f] = p.tunnel
	}
	for k, v := range crtbridge.ParseTunnelLines(extra) {
		out[k] = v
	}
	return out
}
