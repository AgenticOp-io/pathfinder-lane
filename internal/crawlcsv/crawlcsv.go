// Package crawlcsv is the MSP “new customer crawl” seed spreadsheet format.
//
// A short CSV of starting devices: import creates sessions under
// Customers/<customer>/… and those hosts become crawl seeds.
package crawlcsv

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/sessions"
)

// TemplateFileName is the suggested download name.
const TemplateFileName = "pathfinder-customer-crawl-seeds.csv"

// Template is a starter CSV operators can fill in Excel / Sheets / Notepad.
const Template = `host,name,protocol,port,username,folder,notes
192.0.2.1,core-sw1,ssh,22,,seeds,Primary seed — replace with real gear
192.0.2.2,edge-rtr,ssh,22,admin,seeds,
example.com,jump-host,ssh,22,,,
`

// Row is one parsed seed line.
type Row struct {
	Host     string
	Name     string
	Protocol string // ssh | telnet
	Port     int
	Username string
	Folder   string // relative under the customer (optional)
	Notes    string
}

// Parse reads CSV with a header row. Required column: host (alias: hostname).
func Parse(r io.Reader) ([]Row, error) {
	cr := csv.NewReader(r)
	cr.TrimLeadingSpace = true
	cr.ReuseRecord = false
	rows, err := cr.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("csv: %w", err)
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("csv needs a header row and at least one data row")
	}
	idx := headerIndex(rows[0])
	if _, ok := idx["host"]; !ok {
		return nil, fmt.Errorf("csv header must include host (or hostname)")
	}

	out := make([]Row, 0, len(rows)-1)
	for i, raw := range rows[1:] {
		line := i + 2
		host := cell(raw, idx, "host")
		if host == "" {
			continue
		}
		name := cell(raw, idx, "name")
		if name == "" {
			name = host
		}
		proto := strings.ToLower(cell(raw, idx, "protocol"))
		if proto == "" {
			proto = "ssh"
		}
		port := 0
		if ps := cell(raw, idx, "port"); ps != "" {
			p, err := strconv.Atoi(ps)
			if err != nil || p <= 0 || p > 65535 {
				return nil, fmt.Errorf("line %d: invalid port %q", line, ps)
			}
			port = p
		}
		folder := cell(raw, idx, "folder")
		folder = strings.ReplaceAll(folder, `\`, "/")
		folder = strings.Trim(folder, "/")
		out = append(out, Row{
			Host:     host,
			Name:     name,
			Protocol: proto,
			Port:     port,
			Username: cell(raw, idx, "username"),
			Folder:   folder,
			Notes:    cell(raw, idx, "notes"),
		})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("csv has no rows with a host")
	}
	return out, nil
}

// ParseBytes parses Template-shaped CSV bytes.
func ParseBytes(data []byte) ([]Row, error) {
	return Parse(bytes.NewReader(data))
}

// Nodes converts rows to session nodes (defaults filled).
func Nodes(rows []Row) []sessions.Node {
	out := make([]sessions.Node, 0, len(rows))
	for _, r := range rows {
		n := sessions.Defaults()
		n.Name = r.Name
		n.Host = r.Host
		n.Username = r.Username
		n.Notes = r.Notes
		if n.Notes == "" {
			n.Notes = "Imported for customer crawl"
		}
		switch {
		case strings.Contains(r.Protocol, "telnet"):
			n.Transport = sessions.TransportTelnet
			n.Port = 23
		default:
			n.Transport = sessions.TransportSSH
			n.Port = 22
			n.AuthType = sessions.AuthPassword
		}
		if r.Port > 0 {
			n.Port = r.Port
		}
		out = append(out, n)
	}
	return out
}

// GroupByFolder buckets nodes by relative folder ("" → "seeds").
func GroupByFolder(rows []Row) map[string][]sessions.Node {
	grouped := map[string][]sessions.Node{}
	for _, r := range rows {
		folder := r.Folder
		if folder == "" {
			folder = "seeds"
		}
		n := Nodes([]Row{r})[0]
		grouped[folder] = append(grouped[folder], n)
	}
	return grouped
}

func headerIndex(header []string) map[string]int {
	idx := map[string]int{}
	for i, h := range header {
		k := strings.ToLower(strings.TrimSpace(h))
		switch k {
		case "hostname":
			k = "host"
		case "session_name", "session", "label", "device":
			k = "name"
		case "description", "note", "comment":
			k = "notes"
		case "user", "user_name":
			k = "username"
		case "proto":
			k = "protocol"
		}
		idx[k] = i
	}
	return idx
}

func cell(row []string, idx map[string]int, name string) string {
	i, ok := idx[name]
	if !ok || i < 0 || i >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[i])
}
