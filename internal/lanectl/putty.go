package lanectl

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/crtbridge"
)

type puttyEntry struct {
	Key         string
	Name        string
	Host        string
	Port        int
	User        string
	Protocol    string
	ProxyMethod uint32
	ProxyCmd    string
}

func puttySSH(e puttyEntry) bool {
	p := strings.ToLower(strings.TrimSpace(e.Protocol))
	return p == "" || p == "ssh" || p == "ssh2"
}

func discoverPutty(st crtbridge.State) []Host {
	var out []Host
	for _, e := range listPutty() {
		if !puttySSH(e) {
			continue
		}
		host, port := e.Host, e.Port
		if host == "127.0.0.1" || host == "::1" {
			if prev, ok := st.OtherSessions["putty/"+e.Name]; ok && prev.OriginalHost != "" {
				host = prev.OriginalHost
				if prev.OriginalPort > 0 {
					port = prev.OriginalPort
				}
			} else {
				continue
			}
		}
		out = append(out, Host{
			Name:   e.Name,
			Host:   host,
			Port:   port,
			User:   e.User,
			Source: "putty",
		})
	}
	return out
}

// puttyOrig is stored so we can restore HostName / proxy settings.
type puttyOrig struct {
	Host        string `json:"host"`
	Port        int    `json:"port"`
	User        string `json:"user,omitempty"`
	ProxyMethod uint32 `json:"proxy_method"`
	ProxyCmd    string `json:"proxy_cmd,omitempty"`
}

func patchPuttyText(raw, host string, port int, proxyMethod uint32, proxyCmd string) string {
	set := map[string]string{
		"hostname":           host,
		"portnumber":         strconv.Itoa(port),
		"proxymethod":        strconv.FormatUint(uint64(proxyMethod), 10),
		"proxytelnetcommand": proxyCmd,
	}
	seen := map[string]bool{}
	var b strings.Builder
	for _, line := range strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n") {
		if line == "" && b.Len() == 0 {
			continue
		}
		key, _, ok := strings.Cut(line, "=")
		if ok {
			lk := strings.ToLower(strings.TrimSpace(key))
			if val, hit := set[lk]; hit {
				b.WriteString(key)
				b.WriteByte('=')
				b.WriteString(val)
				b.WriteByte('\n')
				seen[lk] = true
				continue
			}
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	order := []string{"hostname", "portnumber", "proxymethod", "proxytelnetcommand"}
	names := map[string]string{
		"hostname": "HostName", "portnumber": "PortNumber",
		"proxymethod": "ProxyMethod", "proxytelnetcommand": "ProxyTelnetCommand",
	}
	for _, k := range order {
		if seen[k] {
			continue
		}
		b.WriteString(names[k])
		b.WriteByte('=')
		b.WriteString(set[k])
		b.WriteByte('\n')
	}
	return b.String()
}

func parsePuttyText(name, raw string) puttyEntry {
	e := puttyEntry{Key: name, Name: name, Port: 22}
	for _, line := range strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n") {
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(key)) {
		case "hostname":
			e.Host = strings.TrimSpace(val)
		case "username":
			e.User = strings.TrimSpace(val)
		case "protocol":
			e.Protocol = strings.TrimSpace(val)
		case "proxytelnetcommand":
			e.ProxyCmd = val
		case "portnumber":
			if p, err := strconv.Atoi(strings.TrimSpace(val)); err == nil && p > 0 {
				e.Port = p
			}
		case "proxymethod":
			if p, err := strconv.ParseUint(strings.TrimSpace(val), 10, 32); err == nil {
				e.ProxyMethod = uint32(p)
			}
		}
	}
	if e.Host == "" {
		e.Host = name
	}
	return e
}

func puttyProxyMethodLocal() uint32 { return 5 }

func fmtPuttyErr(name string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("putty %s: %w", name, err)
}

func puttyOrigFile(appHome string) string {
	return filepath.Join(appHome, "putty-orig.json")
}

func loadPuttyOrig(appHome string) map[string]puttyOrig {
	out := map[string]puttyOrig{}
	raw, err := os.ReadFile(puttyOrigFile(appHome))
	if err != nil {
		return out
	}
	_ = json.Unmarshal(raw, &out)
	if out == nil {
		return map[string]puttyOrig{}
	}
	return out
}

func savePuttyOrig(appHome string, m map[string]puttyOrig) error {
	if err := os.MkdirAll(appHome, 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(puttyOrigFile(appHome), raw, 0o600)
}

func rememberPuttyOrig(store map[string]puttyOrig, e puttyEntry) {
	if _, ok := store[e.Name]; ok {
		return
	}
	if e.Host == "127.0.0.1" || e.Host == "::1" {
		return
	}
	if e.ProxyMethod == puttyProxyMethodLocal() && strings.Contains(e.ProxyCmd, "pflane") {
		return
	}
	store[e.Name] = puttyOrig{
		Host:        e.Host,
		Port:        e.Port,
		User:        e.User,
		ProxyMethod: e.ProxyMethod,
		ProxyCmd:    e.ProxyCmd,
	}
}

func origFor(store map[string]puttyOrig, e puttyEntry) puttyOrig {
	if o, ok := store[e.Name]; ok && o.Host != "" {
		return o
	}
	return puttyOrig{Host: e.Host, Port: e.Port, User: e.User, ProxyMethod: e.ProxyMethod, ProxyCmd: e.ProxyCmd}
}
