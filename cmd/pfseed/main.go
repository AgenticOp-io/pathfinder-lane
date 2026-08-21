// Command pfseed discovers SSH targets you already configured and writes them
// into PathfinderSSH's session tree as crawl seeds.
//
// Secrets are never accepted on the command line. Username is optional metadata
// on the session; the vault still holds credentials.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/sessions"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	cmd := os.Args[1]
	args := os.Args[2:]
	var err error
	switch cmd {
	case "status":
		err = cmdStatus(args)
	case "discover":
		err = cmdDiscover(args)
	case "apply":
		err = cmdApply(args)
	case "import-securecrt":
		err = cmdImportSecureCRT(args)
	case "import-csv":
		err = cmdImportCSV(args)
	case "-h", "-help", "--help", "help":
		fmt.Fprint(os.Stdout, usage)
		return
	default:
		fmt.Fprintf(os.Stderr, "pfseed: unknown command %q\n", cmd)
		fmt.Fprint(os.Stderr, usage)
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "pfseed: %v\n", err)
		os.Exit(1)
	}
}

const usage = `pfseed — create PathfinderSSH session seeds

  pfseed status
  pfseed discover
  pfseed apply -host HOST [-host HOST] [-folder NAME] [-user USER] [-sessions PATH]
  pfseed import-securecrt [-config DIR] [-sessions PATH] [-dry-run] [-preview]
  pfseed import-csv -file FILE [-sessions PATH] [-dry-run]

discover prints JSON of hosts from ~/.ssh/config and (on Windows) PuTTY.
apply merges those hosts into sessions.yaml without overwriting other folders.
import-securecrt reads VanDyke Sessions\**\*.ini (no passwords). Nested CRT
folders become Pathfinder folder names joined with " / ".
import-csv accepts VanDyke-style headers: session_name,hostname,protocol,folder,port,username,description.
`

func appHome() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "."
	}
	dir := filepath.Join(home, ".pathfinderssh")
	_ = os.MkdirAll(dir, 0o755)
	return dir
}

func defaultSessionsPath() string {
	return filepath.Join(appHome(), "sessions.yaml")
}

func cmdStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	path := fs.String("sessions", "", "session file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	p := *path
	if p == "" {
		p = defaultSessionsPath()
	}
	tr, err := sessions.LoadFile(p)
	if err != nil {
		return err
	}
	n := 0
	for _, f := range tr.Folders {
		n += len(f.Sessions)
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{
		"path":      p,
		"sessions":  n,
		"needSetup": n == 0,
	})
}

func cmdDiscover(args []string) error {
	fs := flag.NewFlagSet("discover", flag.ContinueOnError)
	if err := fs.Parse(args); err != nil {
		return err
	}
	found := discoverAll()
	if found == nil {
		found = []Candidate{}
	}
	return json.NewEncoder(os.Stdout).Encode(found)
}

type applyHosts []string

func (a *applyHosts) String() string     { return strings.Join(*a, ",") }
func (a *applyHosts) Set(v string) error { *a = append(*a, v); return nil }

func cmdApply(args []string) error {
	fs := flag.NewFlagSet("apply", flag.ContinueOnError)
	path := fs.String("sessions", "", "session file")
	folder := fs.String("folder", "Seeds", "folder name to create or extend")
	user := fs.String("user", "", "SSH username stored on each new session (not a password)")
	notes := fs.String("notes", "Created by Pathfinder setup", "session notes")
	var hosts applyHosts
	fs.Var(&hosts, "host", "seed: name=host[:port] or host[:port] (repeatable)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if len(hosts) == 0 {
		return fmt.Errorf("at least one -host is required")
	}
	p := *path
	if p == "" {
		p = defaultSessionsPath()
	}
	tr, err := sessions.LoadFile(p)
	if err != nil {
		return err
	}

	idx := -1
	for i, f := range tr.Folders {
		if strings.EqualFold(f.Name, *folder) {
			idx = i
			break
		}
	}
	if idx < 0 {
		tr.Folders = append(tr.Folders, sessions.Folder{Name: *folder})
		idx = len(tr.Folders) - 1
	}

	added := 0
	skipped := 0
	for _, raw := range hosts {
		n, err := nodeFromHost(raw, strings.TrimSpace(*user), *notes)
		if err != nil {
			return err
		}
		if sessionExists(tr, n) {
			skipped++
			continue
		}
		n.Name = uniqueName(tr, n.Name)
		tr.Folders[idx].Sessions = append(tr.Folders[idx].Sessions, n)
		added++
	}
	if err := sessions.SaveFile(p, tr); err != nil {
		return err
	}
	return json.NewEncoder(os.Stdout).Encode(map[string]any{
		"path":    p,
		"added":   added,
		"skipped": skipped,
		"folder":  tr.Folders[idx].Name,
	})
}

func nodeFromHost(raw, user, notes string) (sessions.Node, error) {
	raw = strings.TrimSpace(raw)
	name := ""
	if i := strings.IndexByte(raw, '='); i > 0 {
		name = strings.TrimSpace(raw[:i])
		raw = strings.TrimSpace(raw[i+1:])
	}
	host := raw
	port := 22
	if h, p, ok := splitHostPort(raw); ok {
		host, port = h, p
	}
	if host == "" {
		return sessions.Node{}, fmt.Errorf("empty host in %q", raw)
	}
	if name == "" {
		name = host
	}
	n := sessions.Defaults()
	n.Name = name
	n.Host = host
	n.Port = port
	n.Username = user
	n.Notes = notes
	n.AuthType = sessions.AuthPassword
	n.HostKeyPolicy = sessions.HostKeyTOFU
	return n.Normalize(), nil
}

func splitHostPort(s string) (string, int, bool) {
	// Avoid net.SplitHostPort on bare IPv6 without brackets.
	if strings.Count(s, ":") != 1 {
		return s, 0, false
	}
	i := strings.LastIndexByte(s, ':')
	p, err := strconv.Atoi(s[i+1:])
	if err != nil || p <= 0 || p > 65535 {
		return s, 0, false
	}
	return s[:i], p, true
}

func sessionExists(tr sessions.Tree, n sessions.Node) bool {
	wantHost := strings.ToLower(n.Host)
	for _, f := range tr.Folders {
		for _, s := range f.Sessions {
			if strings.EqualFold(s.Name, n.Name) {
				return true
			}
			if strings.ToLower(s.Host) == wantHost && s.Port == n.Port {
				return true
			}
		}
	}
	return false
}

func uniqueName(tr sessions.Tree, name string) string {
	taken := map[string]bool{}
	for _, f := range tr.Folders {
		for _, s := range f.Sessions {
			taken[strings.ToLower(s.Name)] = true
		}
	}
	if !taken[strings.ToLower(name)] {
		return name
	}
	for i := 2; i < 1000; i++ {
		cand := fmt.Sprintf("%s-%d", name, i)
		if !taken[strings.ToLower(cand)] {
			return cand
		}
	}
	return name
}
