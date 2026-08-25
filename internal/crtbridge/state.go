// Package crtbridge is the standalone SecureCRT companion: backup the customer
// session folder, rewrite sessions onto localhost proxies when Auvik and/or
// FortiClient apply, and leave everyone else on standard SSH.
package crtbridge

import (
	"crypto/sha1"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/scottpeterman/pathfinderssh/internal/crtapp"
)

const (
	stateFileName = "crt-bridge.json"
	modeProxy     = "proxy"
	modeDirect    = "direct"
	modeAuvik     = "auvik" // legacy state from the first CRT rewrite
)

// State is persisted under ~/.pathfinder-crt/crt-bridge.json.
type State struct {
	InstalledAt    string             `json:"installed_at,omitempty"`
	BackupDir      string             `json:"backup_dir,omitempty"`
	CustomerRoot   string             `json:"customer_root,omitempty"`
	CRTSessionsDir string             `json:"crt_sessions_dir,omitempty"`
	Sessions       map[string]Session `json:"sessions,omitempty"`
	// OtherSessions are OpenSSH/PuTTY listeners (keys ssh/<alias>, putty/<name>).
	// CRT Sync never deletes these.
	OtherSessions map[string]Session `json:"other_sessions,omitempty"`
}

// Session is one rewritten (or explicitly left-direct) CRT .ini.
type Session struct {
	OriginalHost string `json:"original_host"`
	OriginalPort int    `json:"original_port"`
	Customer     string `json:"customer,omitempty"`
	Domain       string `json:"domain,omitempty"`
	DeviceIP     string `json:"device_ip,omitempty"`
	FrontPort    int    `json:"front_port,omitempty"`
	Mode         string `json:"mode"`
	UseAuvik     bool   `json:"use_auvik,omitempty"`
	VPNTunnel    string `json:"vpn_tunnel,omitempty"`
}

func (s *Session) normalize() {
	if s == nil {
		return
	}
	if s.Mode == modeAuvik {
		s.Mode = modeProxy
		s.UseAuvik = true
	}
}

func (s Session) Proxied() bool {
	return s.Mode == modeProxy || s.Mode == modeAuvik
}

func (st State) AllListeners() []Session {
	n := len(st.Sessions) + len(st.OtherSessions)
	out := make([]Session, 0, n)
	for _, s := range st.Sessions {
		out = append(out, s)
	}
	for _, s := range st.OtherSessions {
		out = append(out, s)
	}
	return out
}

func (st State) ListenerByFrontPort(port int) (Session, bool) {
	if port <= 0 {
		return Session{}, false
	}
	for _, s := range st.AllListeners() {
		if s.FrontPort == port {
			return s, true
		}
	}
	return Session{}, false
}

// Options configure install/sync/agent.
type Options struct {
	AppHome      string
	CRTConfig    string
	CustomerRoot string
	AuvikUser    string
	AuvikKey     string
	AuvikBase    string
	AgentExe     string
	TunnelBin    string
	Settings     Settings
}

func (o Options) settings() Settings {
	s := o.Settings
	s.Normalize()
	if s.Mode == "" {
		s.Mode = AutoMixed
	}
	if strings.TrimSpace(s.AuvikUser) == "" {
		s.AuvikUser = o.AuvikUser
		s.AuvikKey = o.AuvikKey
		s.AuvikBase = o.AuvikBase
	}
	if strings.TrimSpace(s.TunnelBin) == "" {
		s.TunnelBin = o.TunnelBin
	}
	if strings.TrimSpace(s.CustomerRoot) == "" {
		s.CustomerRoot = o.CustomerRoot
	}
	if strings.TrimSpace(s.CRTConfig) == "" {
		s.CRTConfig = o.CRTConfig
	}
	return s
}

// Report is what install/sync prints.
type Report struct {
	BackupDir    string
	CustomerRoot string
	Mode         string
	Tunnelled    int // localhost proxy (Auvik and/or VPN)
	Direct       int
	Skipped      int
	Errors       []string
}

func StatePath(appHome string) string {
	return filepath.Join(appHome, stateFileName)
}

func LoadState(appHome string) (State, error) {
	raw, err := os.ReadFile(StatePath(appHome))
	if err != nil {
		if os.IsNotExist(err) {
			return State{Sessions: map[string]Session{}, OtherSessions: map[string]Session{}}, nil
		}
		return State{}, err
	}
	var st State
	if err := json.Unmarshal(raw, &st); err != nil {
		return State{}, err
	}
	if st.Sessions == nil {
		st.Sessions = map[string]Session{}
	}
	if st.OtherSessions == nil {
		st.OtherSessions = map[string]Session{}
	}
	for k, s := range st.Sessions {
		s.normalize()
		st.Sessions[k] = s
	}
	for k, s := range st.OtherSessions {
		s.normalize()
		st.OtherSessions[k] = s
	}
	return st, nil
}

func SaveState(appHome string, st State) error {
	if st.Sessions == nil {
		st.Sessions = map[string]Session{}
	}
	if st.OtherSessions == nil {
		st.OtherSessions = map[string]Session{}
	}
	raw, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(appHome, 0o755); err != nil {
		return err
	}
	return os.WriteFile(StatePath(appHome), raw, 0o600)
}

func nowStamp() string {
	return time.Now().Format("20060102-150405")
}

func relKey(rel string) string {
	return filepath.ToSlash(strings.TrimPrefix(rel, `\`))
}

func FrontPortFor(rel string) int {
	return frontListenPort(rel)
}

// frontListenPort is the localhost port SecureCRT connects to. Distinct from
// AuvikTunnel's 42000–51999 range so the agent can splice.
func frontListenPort(rel string) int {
	sum := sha1.Sum([]byte("crt-front:" + relKey(rel)))
	n := int(sum[0])<<8 | int(sum[1])
	return 52000 + n%10000
}

// DefaultAgentExe is the standalone CRT-bridge binary.
func DefaultAgentExe() string {
	candidates := []string{crtapp.AgentExe()}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), crtapp.ExeName("pathfinder-crt")), exe)
	}
	for _, p := range candidates {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return p
		}
	}
	return candidates[0]
}
