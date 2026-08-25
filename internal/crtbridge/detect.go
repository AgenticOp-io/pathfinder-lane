package crtbridge

import (
	"os"
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/crtapp"
	"github.com/scottpeterman/pathfinderssh/internal/crtimport"
)

// Probe is what the installer shows before it rewrites SecureCRT sessions.
type Probe struct {
	CRTConfig      string
	SessionsDir    string
	CustomerRoot   string
	Installed      bool
	AgentPresent   bool
	CRTRunning     bool
	AgentRunning   bool
	SecureCRTFound bool
}

// ProbeEnv inspects this Windows profile for SecureCRT and a prior companion.
func ProbeEnv(crtConfig, appHome string) Probe {
	p := Probe{
		CRTConfig: strings.TrimSpace(crtConfig),
	}
	if p.CRTConfig == "" {
		p.CRTConfig = crtimport.DefaultConfig()
	}
	p.SessionsDir = SessionsDir(p.CRTConfig)
	if p.SessionsDir != "" {
		if st, err := os.Stat(p.SessionsDir); err == nil && st.IsDir() {
			p.SecureCRTFound = true
			p.CustomerRoot = DetectCustomerRoot(p.SessionsDir)
		}
	}
	if appHome == "" {
		appHome = crtapp.Home()
	}
	st, _ := LoadState(appHome)
	p.Installed = strings.TrimSpace(st.InstalledAt) != "" && len(st.Sessions) > 0
	if p.CustomerRoot == "" {
		p.CustomerRoot = st.CustomerRoot
	}
	if ag := crtapp.AgentExe(); fileExists(ag) {
		p.AgentPresent = true
	}
	p.CRTRunning = ProcessRunning("SecureCRT.exe")
	p.AgentRunning = ProcessRunning("lane-crt.exe")
	return p
}
