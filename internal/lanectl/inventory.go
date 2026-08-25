package lanectl

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/crtbridge"
	"github.com/scottpeterman/pathfinderssh/internal/sessions"
)

// Host is one SSH target the engineer already has (CRT, PuTTY, OpenSSH, or hosts.json).
type Host struct {
	Alias    string `json:"alias,omitempty"`
	Folder   string `json:"folder,omitempty"`
	Name     string `json:"name,omitempty"`
	Host     string `json:"host"`
	Port     int    `json:"port,omitempty"`
	User     string `json:"user,omitempty"`
	Source   string `json:"source,omitempty"`
	Rel      string `json:"rel,omitempty"`
	JumpHost string `json:"jump_host,omitempty"`
	JumpPort int    `json:"jump_port,omitempty"`
	JumpUser string `json:"jump_user,omitempty"`
}

func HostsFile(appHome string) string {
	return filepath.Join(appHome, "hosts.json")
}

func LoadHostsFile(appHome string) ([]Host, error) {
	raw, err := os.ReadFile(HostsFile(appHome))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Host
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func mappedFolders(cfg crtbridge.Settings) []string {
	seen := map[string]bool{}
	var out []string
	add := func(m map[string]string) {
		for k := range m {
			k = strings.TrimSpace(k)
			if k == "" || seen[strings.ToLower(k)] {
				continue
			}
			seen[strings.ToLower(k)] = true
			out = append(out, k)
		}
	}
	add(cfg.VPNTunnels)
	add(cfg.AuvikTenants)
	return out
}

// RelKey is the session path used for folder maps (never guessed as a tunnel name).
func (h Host) RelKey() string {
	if strings.TrimSpace(h.Rel) != "" {
		return filepath.ToSlash(h.Rel)
	}
	if strings.EqualFold(h.Source, "putty") {
		return "putty/" + strings.TrimSpace(h.Name)
	}
	name := strings.TrimSpace(h.Name)
	if name == "" {
		name = strings.TrimSpace(h.Alias)
	}
	if h.Folder != "" && name != "" {
		return filepath.ToSlash(h.Folder + "/" + strings.TrimSuffix(name, ".ini") + ".ini")
	}
	return name
}

// Mapped reports whether this host sits under an installer folder map.
func Mapped(cfg crtbridge.Settings, h Host) bool {
	rel := h.RelKey()
	return cfg.VPNTunnelForSession(rel, h.Folder) != "" || cfg.AuvikTenantForSession(rel, h.Folder) != ""
}

func FilterMapped(cfg crtbridge.Settings, hosts []Host) (mapped, skipped []Host) {
	for _, h := range hosts {
		if Mapped(cfg, h) {
			mapped = append(mapped, h)
		} else {
			skipped = append(skipped, h)
		}
	}
	return mapped, skipped
}

func Discover(cfg crtbridge.Settings, appHome, crtConfig string) []Host {
	st, _ := crtbridge.LoadState(appHome)
	var out []Host
	seen := map[string]bool{}
	add := func(h Host) {
		h.Host = strings.TrimSpace(h.Host)
		if skipHost(h.Host) || skipHost(h.Name) || skipHost(h.Alias) {
			return
		}
		if h.Host == "" {
			return
		}
		if h.Port <= 0 {
			h.Port = 22
		}
		if h.Name == "" {
			h.Name = h.Host
		}
		if h.Folder == "" {
			h.Folder = cfg.HostFolder(h.Alias, h.Name, h.Host)
			if h.Folder == "" {
				h.Folder = FolderOfName(h.Name, mappedFolders(cfg))
			}
			if h.Folder == "" {
				h.Folder = FolderOfName(h.Alias, mappedFolders(cfg))
			}
			if h.Folder == "" {
				h.Folder = FolderOfName(h.Rel, mappedFolders(cfg))
			}
		}
		if h.Alias == "" {
			h.Alias = SSHAlias(h.Folder, h.Name)
		}
		key := strings.ToLower(h.Host) + ":" + strconv.Itoa(h.Port)
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, h)
	}

	if dir := crtbridge.SessionsDir(crtConfig); dir != "" {
		for _, h := range discoverCRT(dir, cfg.CustomerRoot, st) {
			add(h)
		}
	}
	for _, h := range discoverPutty(st) {
		add(h)
	}
	if home, err := os.UserHomeDir(); err == nil {
		if raw, err := os.ReadFile(filepath.Join(home, ".ssh", "config")); err == nil {
			for _, h := range parseSSHConfigHosts(string(raw)) {
				h.Source = "ssh_config"
				add(h)
			}
		}
	}
	if extra, err := LoadHostsFile(appHome); err == nil {
		for _, h := range extra {
			h.Source = "hosts.json"
			add(h)
		}
	}
	for _, h := range discoverPathfinder() {
		add(h)
	}
	return out
}

func discoverCRT(sessionsDir, customerRoot string, st crtbridge.State) []Host {
	var out []Host
	_ = filepath.Walk(sessionsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.EqualFold(filepath.Ext(path), ".ini") {
			return nil
		}
		rel, err := filepath.Rel(sessionsDir, path)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		host, port, ok := crtbridge.ReadSSHHostPort(raw)
		if !ok {
			return nil
		}
		if prev, exists := st.Sessions[rel]; exists && prev.OriginalHost != "" {
			if host == "127.0.0.1" || host == "::1" {
				host = prev.OriginalHost
				if prev.OriginalPort > 0 {
					port = prev.OriginalPort
				}
			}
		}
		if host == "127.0.0.1" || host == "::1" {
			return nil
		}
		base := strings.TrimSuffix(filepath.Base(rel), ".ini")
		out = append(out, Host{
			Folder: crtbridge.CustomerOfRel(rel, customerRoot),
			Name:   base,
			Host:   host,
			Port:   port,
			Source: "securecrt",
			Rel:    rel,
		})
		return nil
	})
	return out
}

func skipHost(h string) bool {
	h = strings.ToLower(strings.TrimSpace(h))
	if h == "" || strings.ContainsAny(h, "*?!") {
		return true
	}
	skip := []string{
		"github.com", "gitlab.com", "bitbucket.org", "gist.github.com",
		"ssh.dev.azure.com", "vs-ssh.visualstudio.com", "sourceforge.net",
		"codeberg.org", "git.sr.ht",
	}
	for _, s := range skip {
		if h == s || strings.HasSuffix(h, "."+s) {
			return true
		}
	}
	return false
}

func parseSSHConfigHosts(text string) []Host {
	type cur struct {
		names []string
		host  string
		port  int
		user  string
		jump  string
	}
	var out []Host
	c := cur{port: 22}
	flush := func() {
		for _, n := range c.names {
			h := Host{Alias: n, Name: n, Host: c.host, Port: c.port, User: c.user}
			if h.Host == "" {
				h.Host = n
			}
			if ju, jh, jp := parseProxyJump(c.jump); jh != "" {
				h.JumpUser, h.JumpHost, h.JumpPort = ju, jh, jp
			}
			out = append(out, h)
		}
		c = cur{port: 22}
	}
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := cutSSH(line)
		if !ok {
			continue
		}
		switch strings.ToLower(key) {
		case "host":
			flush()
			for _, n := range strings.Fields(val) {
				if strings.ContainsAny(n, "*?") {
					continue
				}
				c.names = append(c.names, n)
			}
		case "hostname":
			c.host = val
		case "user":
			c.user = val
		case "port":
			if p, err := strconv.Atoi(val); err == nil && p > 0 {
				c.port = p
			}
		case "proxyjump":
			c.jump = val
		}
	}
	flush()
	return out
}

func cutSSH(line string) (key, val string, ok bool) {
	i := strings.IndexAny(line, " \t=")
	if i < 0 {
		return line, "", true
	}
	key = line[:i]
	val = strings.TrimSpace(strings.TrimLeft(line[i:], " \t="))
	val = strings.Trim(val, `"'`)
	return key, val, true
}

// parseProxyJump reads the first hop of an OpenSSH ProxyJump value.
func parseProxyJump(s string) (user, host string, port int) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", "", 0
	}
	if i := strings.IndexByte(s, ','); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	if i := strings.LastIndex(s, "@"); i >= 0 {
		user, s = s[:i], s[i+1:]
	}
	host, port = s, 22
	if h, p, err := net.SplitHostPort(s); err == nil {
		host = h
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			port = n
		}
	}
	return user, host, port
}

func discoverPathfinder() []Host {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	path := filepath.Join(home, ".pathfinderssh", "sessions.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	tree, err := sessions.UnmarshalTree(raw)
	if err != nil {
		return nil
	}
	var out []Host
	tree.WalkSessions(func(folder string, n sessions.Node) {
		if n.Transport != sessions.TransportSSH && n.Transport != "" {
			return
		}
		host := strings.TrimSpace(n.Host)
		if host == "" {
			return
		}
		cust := sessions.CustomerOfFolder(folder)
		port := n.Port
		if port <= 0 {
			port = 22
		}
		name := strings.TrimSpace(n.Name)
		if name == "" {
			name = host
		}
		out = append(out, Host{
			Folder:   cust,
			Name:     name,
			Host:     host,
			Port:     port,
			User:     n.Username,
			Source:   "pathfinder",
			Rel:      filepath.ToSlash(folder + "/" + name + ".ini"),
			JumpHost: strings.TrimSpace(n.Jump.Host),
			JumpPort: n.Jump.Port,
			JumpUser: strings.TrimSpace(n.Jump.Username),
		})
	})
	return out
}
