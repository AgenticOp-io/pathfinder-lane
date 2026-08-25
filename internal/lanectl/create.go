package lanectl

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/appinstall"
	"github.com/scottpeterman/pathfinderssh/internal/auvik"
	"github.com/scottpeterman/pathfinderssh/internal/crtapp"
	"github.com/scottpeterman/pathfinderssh/internal/crtbridge"
	"github.com/scottpeterman/pathfinderssh/internal/crtimport"
)

type CreateRequest struct {
	AppHome   string
	CRTConfig string
	LaneBin   string
	Via       string // auto, proxy, agent
	SSH       bool
	CRT       bool
	Putty     bool
}

type CreateReport struct {
	Via            string
	SSHPath        string
	SSHHosts       int
	CRT            crtbridge.Report
	CRTRan         bool
	PuttyRewritten int
	PuttySkipped   int
	Mapped         int
	Skipped        int
	Notes          []string
	Aliases        []string
}

func NormalizeVia(via string) string {
	switch strings.ToLower(strings.TrimSpace(via)) {
	case "agent":
		return "agent"
	case "proxy":
		return "proxy"
	default:
		return "auto"
	}
}

func sshVia(via string) string {
	if via == "agent" {
		return "agent"
	}
	return "proxy"
}

func puttyVia(via string) string {
	if via == "agent" {
		return "agent"
	}
	return "proxy"
}

func LaneBin() string {
	if p := crtapp.LaneExe(); fileExists(p) {
		return p
	}
	if p := filepath.Join(appinstall.BinDir(), crtapp.ExeName("pflane")); fileExists(p) {
		return p
	}
	if exe, err := os.Executable(); err == nil {
		if abs, err := filepath.Abs(exe); err == nil && !isGoRunCache(abs) {
			return abs
		}
	}
	return "pflane"
}

func BridgeOptions(appHome string) crtbridge.Options {
	if appHome == "" {
		appHome = crtapp.Home()
	}
	s, err := crtbridge.LoadSettings(appHome)
	if err != nil {
		s = crtbridge.Settings{Mode: crtbridge.AutoMixed}
	}
	crt := s.CRTConfig
	if crt == "" {
		crt = crtimport.DefaultConfig()
	}
	return crtbridge.Options{
		AppHome:      appHome,
		CRTConfig:    crt,
		CustomerRoot: s.CustomerRoot,
		AuvikUser:    s.AuvikUser,
		AuvikKey:     s.AuvikKey,
		AuvikBase:    s.AuvikBase,
		TunnelBin:    s.TunnelBin,
		Settings:     s,
	}
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func Create(ctx context.Context, req CreateRequest) (CreateReport, error) {
	var rep CreateReport
	if req.AppHome == "" {
		req.AppHome = crtapp.Home()
	}
	if req.LaneBin == "" {
		if dest, err := InstallSelf(); err == nil {
			req.LaneBin = dest
		} else {
			req.LaneBin = LaneBin()
		}
	}
	if link, err := InstallOnPATH(req.LaneBin); err == nil && link != "" {
		rep.Notes = append(rep.Notes, "PATH: "+link)
	}
	req.Via = NormalizeVia(req.Via)
	rep.Via = req.Via

	opts := BridgeOptions(req.AppHome)
	if req.CRTConfig != "" {
		opts.CRTConfig = req.CRTConfig
		opts.Settings.CRTConfig = req.CRTConfig
	}
	cfg := opts.Settings
	hosts := Discover(cfg, req.AppHome, opts.CRTConfig)
	mapped, skipped := FilterMapped(cfg, hosts)
	rep.Mapped = len(mapped)
	rep.Skipped = len(skipped)
	if len(cfg.VPNTunnels) == 0 && len(cfg.AuvikTenants) == 0 {
		rep.Notes = append(rep.Notes, "No folder maps yet. Run: pflane map-set FOLDER VPN-TARGET")
	}

	if req.SSH {
		sshHosts := make([]SSHHost, 0, len(mapped))
		for _, h := range mapped {
			sh := SSHHost{
				Alias:    h.Alias,
				Folder:   h.Folder,
				Host:     h.Host,
				Port:     h.Port,
				User:     h.User,
				Via:      sshVia(req.Via),
				JumpHost: h.JumpHost,
				JumpPort: h.JumpPort,
				JumpUser: h.JumpUser,
			}
			if sh.Via == "agent" {
				entry := classifyHost(cfg, h)
				sh.FrontPort = entry.FrontPort
				if sh.FrontPort == 0 {
					sh.Via = "proxy"
				}
			}
			sshHosts = append(sshHosts, sh)
			rep.Aliases = append(rep.Aliases, sh.Alias)
		}
		path, err := WriteSSHConfig(req.AppHome, req.LaneBin, sshHosts)
		if err != nil {
			return rep, err
		}
		rep.SSHPath = path
		rep.SSHHosts = len(sshHosts)
		if len(sshHosts) == 0 {
			rep.Notes = append(rep.Notes, "OpenSSH: no mapped hosts (unmapped sessions stay as they are)")
		} else if sshVia(req.Via) == "proxy" {
			rep.Notes = append(rep.Notes, "OpenSSH: ssh <alias>  (ProxyCommand — no daemon)")
		} else {
			rep.Notes = append(rep.Notes, "OpenSSH aliases point at 127.0.0.1 — run: pflane serve")
		}
	}

	if req.CRT {
		dir := crtbridge.SessionsDir(opts.CRTConfig)
		if dir == "" {
			rep.Notes = append(rep.Notes, "SecureCRT not found on this OS/profile — skipped")
		} else if st, err := os.Stat(dir); err != nil || !st.IsDir() {
			rep.Notes = append(rep.Notes, "SecureCRT Sessions folder missing — skipped")
		} else {
			if p := crtapp.AgentExe(); fileExists(p) {
				opts.AgentExe = p
			} else if req.LaneBin != "" && fileExists(req.LaneBin) {
				opts.AgentExe = req.LaneBin
			} else if p := crtapp.LaneExe(); fileExists(p) {
				opts.AgentExe = p
			}
			crtRep, err := crtbridge.AfterInstall(ctx, opts)
			rep.CRT = crtRep
			rep.CRTRan = true
			if err != nil {
				rep.Notes = append(rep.Notes, "SecureCRT: "+err.Error())
			} else {
				rep.Notes = append(rep.Notes, fmt.Sprintf("SecureCRT: %d localhost proxy, %d standard SSH — needs: pflane serve (login autostart if those were rewritten)", crtRep.Tunnelled, crtRep.Direct))
			}
		}
	}

	st, _ := crtbridge.LoadState(req.AppHome)
	if st.OtherSessions == nil {
		st.OtherSessions = map[string]crtbridge.Session{}
	}
	for k := range st.OtherSessions {
		if strings.HasPrefix(k, "ssh/") || strings.HasPrefix(k, "putty/") {
			delete(st.OtherSessions, k)
		}
	}
	if req.SSH && sshVia(req.Via) == "agent" {
		for _, h := range mapped {
			entry := classifyHost(cfg, h)
			if !entry.Proxied() {
				continue
			}
			st.OtherSessions["ssh/"+h.Alias] = entry
		}
	}

	if req.Putty {
		n, skip, notes, err := rewritePutty(req, cfg, mapped, &st)
		rep.PuttyRewritten = n
		rep.PuttySkipped = skip
		rep.Notes = append(rep.Notes, notes...)
		if err != nil {
			return rep, err
		}
	}

	if err := crtbridge.SaveState(req.AppHome, st); err != nil {
		return rep, err
	}
	return rep, nil
}

func classifyHost(cfg crtbridge.Settings, h Host) crtbridge.Session {
	entry := crtbridge.ClassifySession(cfg, h.Folder, h.Host, h.Port, h.RelKey(), true, nil, auvik.TenantMap{}, crtbridge.Session{})
	if entry.Proxied() {
		return entry
	}
	if !Mapped(cfg, h) {
		return entry
	}
	entry.OriginalHost = h.Host
	entry.OriginalPort = h.Port
	entry.Customer = h.Folder
	entry.DeviceIP = h.Host
	entry.Mode = "proxy"
	entry.FrontPort = crtbridge.FrontPortFor(h.RelKey())
	if v := cfg.VPNTunnelForSession(h.RelKey(), h.Folder); v != "" {
		entry.VPNTunnel = v
	}
	if cfg.AuvikTenantForSession(h.RelKey(), h.Folder) != "" {
		entry.UseAuvik = true
	}
	return entry
}

func rewritePutty(req CreateRequest, cfg crtbridge.Settings, mapped []Host, st *crtbridge.State) (rewritten, skipped int, notes []string, err error) {
	entries := listPutty()
	if len(entries) == 0 {
		if runtime.GOOS == "windows" {
			notes = append(notes, "PuTTY: no sessions in the registry")
		} else {
			notes = append(notes, "PuTTY: no ~/.putty/sessions (skipped)")
		}
		return 0, 0, notes, nil
	}
	want := map[string]Host{}
	for _, h := range mapped {
		if strings.EqualFold(h.Source, "putty") || want[strings.ToLower(h.Name)].Host == "" {
			want[strings.ToLower(h.Name)] = h
		}
	}
	store := loadPuttyOrig(req.AppHome)
	via := puttyVia(req.Via)
	for _, e := range entries {
		if !puttySSH(e) {
			skipped++
			continue
		}
		h, ok := want[strings.ToLower(e.Name)]
		if !ok {
			// Also match by discovered host if the PuTTY name isn't the folder session name.
			for _, m := range mapped {
				if strings.EqualFold(m.Host, e.Host) && m.Port == e.Port {
					h, ok = m, true
					break
				}
			}
		}
		if !ok {
			skipped++
			continue
		}
		rememberPuttyOrig(store, e)
		orig := origFor(store, e)
		host, port := orig.Host, orig.Port
		proxyMethod := puttyProxyMethodLocal()
		proxyCmd := LaneProxyCommand(req.LaneBin, h.Folder, "%host", "%port")
		if via == "agent" {
			entry := classifyHost(cfg, h)
			if !entry.Proxied() {
				skipped++
				continue
			}
			host, port = "127.0.0.1", entry.FrontPort
			proxyMethod, proxyCmd = 0, ""
			st.OtherSessions["putty/"+e.Name] = entry
		}
		if err := applyPutty(e, host, port, proxyMethod, proxyCmd); err != nil {
			return rewritten, skipped, notes, fmtPuttyErr(e.Name, err)
		}
		rewritten++
	}
	if err := savePuttyOrig(req.AppHome, store); err != nil {
		return rewritten, skipped, notes, err
	}
	if rewritten == 0 {
		notes = append(notes, "PuTTY: no mapped sessions (folder map does not cover them)")
	} else if via == "proxy" {
		notes = append(notes, fmt.Sprintf("PuTTY: %d sessions use local proxy (no daemon)", rewritten))
	} else {
		notes = append(notes, fmt.Sprintf("PuTTY: %d sessions → 127.0.0.1 — run: pflane serve", rewritten))
	}
	return rewritten, skipped, notes, nil
}

func RestorePutty(appHome string) (int, error) {
	if appHome == "" {
		appHome = crtapp.Home()
	}
	store := loadPuttyOrig(appHome)
	n := 0
	for _, e := range listPutty() {
		o, ok := store[e.Name]
		if !ok || o.Host == "" {
			continue
		}
		if err := applyPutty(e, o.Host, o.Port, o.ProxyMethod, o.ProxyCmd); err != nil {
			return n, fmtPuttyErr(e.Name, err)
		}
		n++
	}
	st, _ := crtbridge.LoadState(appHome)
	for k := range st.OtherSessions {
		if strings.HasPrefix(k, "putty/") {
			delete(st.OtherSessions, k)
		}
	}
	return n, crtbridge.SaveState(appHome, st)
}
