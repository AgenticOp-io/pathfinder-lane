// Package crtimport reads VanDyke SecureCRT session .ini files.
// Secrets (passwords) are never parsed or returned.
package crtimport

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/sessions"
)

// Session is one SecureCRT .ini session (no secrets).
type Session struct {
	Name     string
	Folder   string // path relative to Sessions, levels joined with " / "
	Host     string
	Port     int
	User     string
	Protocol string // ssh, telnet, serial, unsupported
	Serial   string
	Notes    string
	Skipped  string
}

// Import walks Config/Sessions (or a Sessions directory) and returns sessions.
func Import(configRoot string) ([]Session, error) {
	sessionsDir := filepath.Join(configRoot, "Sessions")
	if st, err := os.Stat(sessionsDir); err != nil || !st.IsDir() {
		if st, err2 := os.Stat(configRoot); err2 == nil && st.IsDir() &&
			strings.EqualFold(filepath.Base(configRoot), "Sessions") {
			sessionsDir = configRoot
		} else {
			return nil, fmt.Errorf("no Sessions folder under %s", configRoot)
		}
	}

	var out []Session
	err := filepath.WalkDir(sessionsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		base := d.Name()
		if !strings.EqualFold(filepath.Ext(base), ".ini") {
			return nil
		}
		if strings.EqualFold(base, "__FolderData__.ini") {
			return nil
		}
		rel, err := filepath.Rel(sessionsDir, path)
		if err != nil {
			return err
		}
		cs, err := ParseSessionINI(path, rel)
		if err != nil {
			return nil
		}
		out = append(out, cs)
		return nil
	})
	return out, err
}

// ParseSessionINI reads one session file.
func ParseSessionINI(path, rel string) (Session, error) {
	f, err := os.Open(path)
	if err != nil {
		return Session{}, err
	}
	defer f.Close()

	cs := Session{
		Name: strings.TrimSuffix(filepath.Base(rel), filepath.Ext(rel)),
		Port: 22,
	}
	dir := filepath.Dir(rel)
	if dir != "." && dir != "" {
		parts := strings.Split(dir, string(filepath.Separator))
		cs.Folder = strings.Join(parts, " / ")
	}

	sc := bufio.NewScanner(f)
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 1024*1024)

	proto := ""
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := splitField(line)
		if !ok {
			continue
		}
		switch key {
		case `S:"Hostname"`:
			cs.Host = val
		case `S:"Username"`:
			cs.User = val
		case `S:"Protocol Name"`:
			proto = val
		case `D:"[SSH2] Port"`, `D:"Port"`:
			if p, err := strconv.ParseInt(val, 16, 32); err == nil && p > 0 {
				cs.Port = int(p)
			} else if p, err := strconv.Atoi(val); err == nil && p > 0 {
				cs.Port = p
			}
		case `S:"Com Port"`:
			cs.Serial = val
		case `S:"Description"`:
			cs.Notes = strings.ReplaceAll(val, `\r`, "\n")
		}
	}
	if err := sc.Err(); err != nil {
		return Session{}, err
	}

	switch strings.ToLower(proto) {
	case "ssh2", "ssh1", "ssh":
		cs.Protocol = "ssh"
		if cs.Port == 0 {
			cs.Port = 22
		}
	case "telnet":
		cs.Protocol = "telnet"
		if cs.Port == 0 {
			cs.Port = 23
		}
	case "serial":
		cs.Protocol = "serial"
	case "rdp", "raw", "rlogin", "":
		if cs.Host != "" && proto == "" {
			cs.Protocol = "ssh"
		} else {
			cs.Protocol = "unsupported"
			cs.Skipped = proto
			if cs.Skipped == "" {
				cs.Skipped = "unknown"
			}
		}
	default:
		cs.Protocol = "unsupported"
		cs.Skipped = proto
	}

	if cs.Protocol == "ssh" || cs.Protocol == "telnet" {
		if cs.Host == "" {
			cs.Protocol = "unsupported"
			cs.Skipped = "no hostname"
		}
	}
	if cs.Protocol == "serial" && cs.Serial == "" {
		cs.Protocol = "unsupported"
		cs.Skipped = "no com port"
	}
	return cs, nil
}

func splitField(line string) (key, val string, ok bool) {
	eq := strings.IndexByte(line, '=')
	if eq < 0 {
		return "", "", false
	}
	return line[:eq], line[eq+1:], true
}

// ToNode converts a supported CRT session into a Pathfinder node.
func ToNode(cs Session) sessions.Node {
	n := sessions.Defaults()
	n.Name = cs.Name
	n.Notes = "Imported from SecureCRT"
	if cs.Notes != "" {
		n.Notes = cs.Notes
	}
	n.Username = cs.User
	switch cs.Protocol {
	case "telnet":
		n.Transport = sessions.TransportTelnet
		n.Host = cs.Host
		n.Port = cs.Port
	case "serial":
		n.Transport = sessions.TransportSerial
		n.SerialPort = cs.Serial
	default:
		n.Transport = sessions.TransportSSH
		n.Host = cs.Host
		n.Port = cs.Port
		n.AuthType = sessions.AuthPassword
		n.HostKeyPolicy = sessions.HostKeyTOFU
	}
	return n.Normalize()
}

// Folders groups supported sessions into Pathfinder folders (path-encoded names).
func Folders(list []Session) (folders []sessions.Folder, supported, skipped int) {
	by := map[string][]sessions.Node{}
	for _, cs := range list {
		if cs.Protocol == "unsupported" {
			skipped++
			continue
		}
		folder := cs.Folder
		if folder == "" {
			folder = "SecureCRT"
		}
		by[folder] = append(by[folder], ToNode(cs))
		supported++
	}
	for name, nodes := range by {
		folders = append(folders, sessions.Folder{Name: name, Sessions: nodes})
	}
	return folders, supported, skipped
}

// DefaultConfig returns the usual Windows VanDyke Config path, or "".
func DefaultConfig() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	p := filepath.Join(home, "AppData", "Roaming", "VanDyke", "Config")
	if st, err := os.Stat(p); err == nil && st.IsDir() {
		return p
	}
	return ""
}
