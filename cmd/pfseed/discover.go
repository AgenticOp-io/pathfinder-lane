package main

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Candidate is one host already named in the user's own SSH tooling.
type Candidate struct {
	Name   string `json:"name"`
	Host   string `json:"host"`
	Port   int    `json:"port"`
	User   string `json:"user,omitempty"`
	Source string `json:"source"`
}

func discoverAll() []Candidate {
	var out []Candidate
	seen := map[string]bool{}
	add := func(c Candidate) {
		c.Name = strings.TrimSpace(c.Name)
		c.Host = strings.TrimSpace(c.Host)
		if c.Port <= 0 {
			c.Port = 22
		}
		if c.Host == "" || skipHost(c.Host) || skipHost(c.Name) {
			return
		}
		if c.Name == "" {
			c.Name = c.Host
		}
		key := strings.ToLower(c.Host) + ":" + strconv.Itoa(c.Port)
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, c)
	}

	home, err := os.UserHomeDir()
	if err == nil {
		cfg := filepath.Join(home, ".ssh", "config")
		if data, err := os.ReadFile(cfg); err == nil {
			for _, c := range parseSSHConfig(string(data)) {
				c.Source = "ssh_config"
				add(c)
			}
		}
	}
	for _, c := range discoverPutty() {
		add(c)
	}
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
		"codeberg.org", "git.sr.ht", "gitlab.gnome.org",
	}
	for _, s := range skip {
		if h == s || strings.HasSuffix(h, "."+s) {
			return true
		}
	}
	return false
}

func parseSSHConfig(text string) []Candidate {
	var out []Candidate
	var names []string
	cur := Candidate{Port: 22}

	flush := func() {
		if len(names) == 0 {
			return
		}
		host := cur.Host
		port := cur.Port
		user := cur.User
		for _, n := range names {
			c := Candidate{Name: n, Host: host, Port: port, User: user}
			if c.Host == "" {
				c.Host = n
			}
			out = append(out, c)
		}
		names = nil
		cur = Candidate{Port: 22}
	}

	sc := bufio.NewScanner(strings.NewReader(text))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := splitSSHField(line)
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
				names = append(names, n)
			}
		case "hostname":
			cur.Host = val
		case "user":
			cur.User = val
		case "port":
			if p, err := strconv.Atoi(val); err == nil {
				cur.Port = p
			}
		}
	}
	flush()
	return out
}

func splitSSHField(line string) (key, val string, ok bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", "", false
	}
	i := strings.IndexAny(line, " \t=")
	if i < 0 {
		return line, "", true
	}
	key = line[:i]
	val = strings.TrimSpace(strings.TrimLeft(line[i:], " \t="))
	val = strings.Trim(val, `"'`)
	return key, val, true
}
