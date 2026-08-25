package crtbridge

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/scottpeterman/pathfinderssh/internal/auvik"
	"github.com/scottpeterman/pathfinderssh/internal/crtapp"
	"github.com/scottpeterman/pathfinderssh/internal/crtimport"
)

// Install backs up the SecureCRT customer folder, rewrites sessions that need
// Auvik and/or FortiClient onto localhost, leaves others on standard SSH, and
// registers the agent to keep that mapping current.
func Install(ctx context.Context, opts Options) (Report, error) {
	var rep Report
	if strings.TrimSpace(opts.AppHome) == "" {
		return rep, fmt.Errorf("app home required")
	}
	_ = MigrateLegacyState(opts.AppHome)
	cfg := opts.settings()
	if strings.TrimSpace(cfg.TunnelBin) == "" {
		if p := crtapp.TunnelExe(); fileExists(p) {
			cfg.TunnelBin = p
			opts.TunnelBin = p
			opts.Settings.TunnelBin = p
		}
	}
	if err := SaveSettings(opts.AppHome, cfg); err != nil {
		return rep, err
	}
	sessionsDir := SessionsDir(opts.CRTConfig)
	if sessionsDir == "" {
		return rep, fmt.Errorf("SecureCRT Config\\Sessions not found")
	}
	if st, err := os.Stat(sessionsDir); err != nil || !st.IsDir() {
		return rep, fmt.Errorf("SecureCRT Sessions folder missing: %s", sessionsDir)
	}

	st, _ := LoadState(opts.AppHome)
	customerRoot := strings.TrimSpace(opts.CustomerRoot)
	if customerRoot == "" {
		customerRoot = st.CustomerRoot
	}
	if customerRoot == "" {
		customerRoot = DetectCustomerRoot(sessionsDir)
	}
	rep.CustomerRoot = customerRoot

	if strings.TrimSpace(st.InstalledAt) != "" && len(st.Sessions) > 0 {
		syncRep, err := Sync(ctx, opts)
		rep.BackupDir = st.BackupDir
		rep.Tunnelled = syncRep.Tunnelled
		rep.Direct = syncRep.Direct
		rep.Skipped = syncRep.Skipped
		rep.Errors = append(rep.Errors, syncRep.Errors...)
		if err != nil {
			return rep, err
		}
		applyAutostart(&rep, opts.AgentExe)
		return rep, nil
	}

	backupDir, err := BackupCustomerFolder(sessionsDir, customerRoot, opts.AppHome)
	if err != nil {
		return rep, err
	}
	rep.BackupDir = backupDir
	st.BackupDir = backupDir
	st.CustomerRoot = customerRoot
	st.CRTSessionsDir = sessionsDir
	st.InstalledAt = time.Now().UTC().Format(time.RFC3339)
	if err := SaveState(opts.AppHome, st); err != nil {
		return rep, err
	}

	syncRep, err := Sync(ctx, opts)
	rep.Tunnelled = syncRep.Tunnelled
	rep.Direct = syncRep.Direct
	rep.Skipped = syncRep.Skipped
	rep.Errors = append(rep.Errors, syncRep.Errors...)
	if err != nil {
		return rep, err
	}

	applyAutostart(&rep, opts.AgentExe)
	return rep, nil
}

// AfterInstall runs when the CRT companion is already present: sync sessions
// and ensure the agent is running. First time, it performs a full Install.
func AfterInstall(ctx context.Context, opts Options) (Report, error) {
	if strings.TrimSpace(opts.AgentExe) == "" {
		opts.AgentExe = DefaultAgentExe()
	}
	if SessionsDir(opts.CRTConfig) == "" {
		return Report{}, nil
	}
	_ = MigrateLegacyState(opts.AppHome)
	st, _ := LoadState(opts.AppHome)
	if strings.TrimSpace(st.InstalledAt) != "" {
		rep, err := Sync(ctx, opts)
		applyAutostart(&rep, opts.AgentExe)
		return rep, err
	}
	return Install(ctx, opts)
}

// Sync re-checks Auvik (when enabled) and FortiClient maps, then updates CRT
// sessions in place. Unmatched customers stay on / revert to standard SSH.
func Sync(ctx context.Context, opts Options) (Report, error) {
	var rep Report
	_ = MigrateLegacyState(opts.AppHome)
	st, err := LoadState(opts.AppHome)
	if err != nil {
		return rep, err
	}
	sessionsDir := st.CRTSessionsDir
	if sessionsDir == "" {
		sessionsDir = SessionsDir(opts.CRTConfig)
	}
	if sessionsDir == "" {
		return rep, fmt.Errorf("SecureCRT Sessions folder not found")
	}
	customerRoot := strings.TrimSpace(opts.CustomerRoot)
	if customerRoot == "" {
		customerRoot = st.CustomerRoot
	}
	if customerRoot == "" {
		customerRoot = DetectCustomerRoot(sessionsDir)
	}
	rep.CustomerRoot = customerRoot
	st.CustomerRoot = customerRoot
	st.CRTSessionsDir = sessionsDir

	cfg := opts.settings()
	rep.Mode = cfg.Mode

	tm, _ := auvik.LoadTenantMap(opts.AppHome)
	if cfg.AuvikEnabled() {
		if legacy := crtapp.LegacyPathfinderHome(); legacy != "" && legacy != opts.AppHome {
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
	}
	var tenants []auvik.Tenant
	apiOK := false
	if cfg.AuvikEnabled() && strings.TrimSpace(cfg.AuvikUser) != "" && strings.TrimSpace(cfg.AuvikKey) != "" {
		cli := auvik.New(cfg.AuvikUser, cfg.AuvikKey, cfg.AuvikBase)
		list, err := cli.ListTenants(ctx)
		if err != nil {
			rep.Errors = append(rep.Errors, "Auvik tenant check: "+err.Error())
		} else {
			tenants = list
			apiOK = true
		}
	}
	lookupReady := cfg.AuvikEnabled() && (apiOK || len(tm.Mappings) > 0 || len(tm.Domains) > 0)

	list, err := crtimport.Import(filepath.Dir(sessionsDir))
	if err != nil {
		// Import wants Config root (parent of Sessions).
		list, err = crtimport.Import(sessionsDir)
	}
	if err != nil {
		return rep, err
	}

	if st.Sessions == nil {
		st.Sessions = map[string]Session{}
	}

	seen := map[string]bool{}
	for _, cs := range list {
		rel := sessionRel(cs)
		if rel == "" {
			rep.Skipped++
			continue
		}
		seen[rel] = true
		full := filepath.Join(sessionsDir, filepath.FromSlash(rel))
		raw, err := os.ReadFile(full)
		if err != nil {
			rep.Skipped++
			continue
		}
		if cs.Protocol != "ssh" {
			rep.Skipped++
			continue
		}

		prev := st.Sessions[rel]
		host, port, ok := ReadSSHHostPort(raw)
		if !ok {
			rep.Skipped++
			continue
		}
		origHost, origPort := host, port
		if prev.OriginalHost != "" {
			origHost = prev.OriginalHost
			origPort = prev.OriginalPort
		} else if host == "127.0.0.1" || host == "::1" {
			// Already localhost and we have no original — leave it.
			rep.Skipped++
			continue
		}
		if origPort <= 0 {
			origPort = 22
		}

		customer := CustomerOfRel(rel, customerRoot)
		entry := classifySession(cfg, customer, origHost, origPort, rel, lookupReady, tenants, tm, prev)

		wantHost, wantPort := origHost, origPort
		if entry.Proxied() {
			wantHost = "127.0.0.1"
			wantPort = entry.FrontPort
		}
		if host != wantHost || port != wantPort {
			if err := os.WriteFile(full, PatchSSHHostPort(raw, wantHost, wantPort), 0o644); err != nil {
				rep.Errors = append(rep.Errors, rel+": "+err.Error())
				continue
			}
		}
		st.Sessions[rel] = entry
		if entry.Proxied() {
			rep.Tunnelled++
		} else {
			rep.Direct++
		}
	}

	for rel := range st.Sessions {
		if !seen[rel] {
			delete(st.Sessions, rel)
		}
	}
	if err := SaveState(opts.AppHome, st); err != nil {
		return rep, err
	}
	return rep, nil
}

func sessionRel(cs crtimport.Session) string {
	folder := strings.TrimSpace(cs.Folder)
	name := strings.TrimSpace(cs.Name)
	if name == "" {
		return ""
	}
	parts := []string{}
	if folder != "" {
		for _, p := range strings.Split(folder, " / ") {
			p = strings.TrimSpace(p)
			if p != "" {
				parts = append(parts, p)
			}
		}
	}
	parts = append(parts, name+".ini")
	return relKey(filepath.ToSlash(filepath.Join(parts...)))
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func deviceAddr(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return host
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}
	return host
}

// Uninstall restores original SSH host/port in CRT sessions, removes autostart,
// and leaves the backup folder in place.
func Uninstall(appHome string) error {
	st, err := LoadState(appHome)
	if err != nil {
		return err
	}
	sessionsDir := st.CRTSessionsDir
	for rel, s := range st.Sessions {
		if s.OriginalHost == "" {
			continue
		}
		full := filepath.Join(sessionsDir, filepath.FromSlash(rel))
		raw, err := os.ReadFile(full)
		if err != nil {
			continue
		}
		_ = os.WriteFile(full, PatchSSHHostPort(raw, s.OriginalHost, s.OriginalPort), 0o644)
	}
	_ = DisableAutostart()
	st.Sessions = map[string]Session{}
	st.InstalledAt = ""
	return SaveState(appHome, st)
}

// applyAutostart registers login start only when CRT sessions were rewritten
// onto localhost. OpenSSH/PuTTY proxy mode must not get a daemon.
func applyAutostart(rep *Report, exe string) {
	exe = strings.TrimSpace(exe)
	if rep.Tunnelled <= 0 {
		_ = DisableAutostart()
		return
	}
	if exe == "" || !fileExists(exe) {
		rep.Errors = append(rep.Errors, "autostart: agent binary not found — run pflane serve while using SecureCRT")
		return
	}
	if err := EnableAutostart(exe); err != nil {
		rep.Errors = append(rep.Errors, "autostart: "+err.Error())
	}
	if err := RestartAgent(exe); err != nil {
		rep.Errors = append(rep.Errors, "start agent: "+err.Error())
	}
}
