// Package workcontext tracks the engineer's active incident/work item locally.
// It augments PagerDuty/Opsgenie — Pathfinder does not own incident workflow.
package workcontext

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Provider names an incident system (pagerduty, opsgenie, …).
const (
	ProviderPagerDuty = "pagerduty"
)

// Context is the engineer's active work binding for this desktop session.
type Context struct {
	Provider      string    `json:"provider,omitempty"`
	IncidentID    string    `json:"incident_id,omitempty"`
	IncidentURL   string    `json:"incident_url,omitempty"`
	Title         string    `json:"title,omitempty"`
	CustomerName  string    `json:"customer_name,omitempty"`
	StartedAt     time.Time `json:"started_at,omitempty"`
	LinkedHosts   []string  `json:"linked_hosts,omitempty"`
	EngineerNotes string    `json:"engineer_notes,omitempty"`
}

func (c Context) Active() bool {
	return strings.TrimSpace(c.IncidentID) != ""
}

func (c Context) DisplayLabel() string {
	id := strings.TrimSpace(c.IncidentID)
	if id == "" {
		return ""
	}
	if c := strings.TrimSpace(c.CustomerName); c != "" {
		return "Incident " + id + " · " + c
	}
	return "Incident " + id
}

// Path returns the default persistence file under app home.
func Path(home string) string {
	return filepath.Join(home, "work-context.json")
}

// Load reads persisted work context. Missing file yields empty context.
func Load(path string) (Context, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Context{}, nil
		}
		return Context{}, err
	}
	var c Context
	if err := json.Unmarshal(data, &c); err != nil {
		return Context{}, err
	}
	return c, nil
}

// Save writes work context to disk.
func Save(path string, c Context) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// Clear removes the persisted file.
func Clear(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// RecordHost appends a host to linked hosts (deduped).
func (c *Context) RecordHost(host string) {
	host = strings.TrimSpace(host)
	if host == "" {
		return
	}
	for _, h := range c.LinkedHosts {
		if strings.EqualFold(h, host) {
			return
		}
	}
	c.LinkedHosts = append(c.LinkedHosts, host)
}

// NormalizeIncidentID extracts an id from a PagerDuty URL or raw paste.
func NormalizeIncidentID(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.Contains(raw, "/incidents/") {
		parts := strings.Split(raw, "/incidents/")
		if len(parts) > 1 {
			id := strings.Split(parts[1], "/")[0]
			id = strings.Split(id, "?")[0]
			return strings.TrimSpace(id)
		}
	}
	return raw
}

// Bind starts or replaces active work context.
func Bind(provider, incidentRaw, customer, title, notes string) Context {
	id := NormalizeIncidentID(incidentRaw)
	c := Context{
		Provider:      strings.TrimSpace(provider),
		IncidentID:    id,
		IncidentURL:   strings.TrimSpace(incidentRaw),
		CustomerName:  strings.TrimSpace(customer),
		Title:         strings.TrimSpace(title),
		EngineerNotes: strings.TrimSpace(notes),
		StartedAt:     time.Now(),
	}
	if c.IncidentURL == "" && id != "" {
		c.IncidentURL = id
	}
	return c
}

// SummaryInput feeds engineer work summary generation.
type SummaryInput struct {
	Context     Context
	SessionHost string
	SessionName string
	OpenTabs    []TabInfo
	EngineerNote string
}

// TabInfo describes one open terminal for documentation.
type TabInfo struct {
	Title string
	Host  string
}

// BuildSummary returns a plaintext engineer work note (no secrets).
func BuildSummary(in SummaryInput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "PathfinderSSH MSP — engineer work note\n")
	fmt.Fprintf(&b, "Time: %s\n", time.Now().Format(time.RFC3339))
	if in.Context.IncidentID != "" {
		fmt.Fprintf(&b, "Incident: %s\n", in.Context.IncidentID)
	}
	if in.Context.CustomerName != "" {
		fmt.Fprintf(&b, "Customer: %s\n", in.Context.CustomerName)
	}
	if in.Context.Title != "" {
		fmt.Fprintf(&b, "Title: %s\n", in.Context.Title)
	}
	if in.SessionHost != "" {
		fmt.Fprintf(&b, "Session host: %s\n", in.SessionHost)
	}
	if in.SessionName != "" {
		fmt.Fprintf(&b, "Session: %s\n", in.SessionName)
	}
	if len(in.Context.LinkedHosts) > 0 {
		fmt.Fprintf(&b, "Hosts touched: %s\n", strings.Join(in.Context.LinkedHosts, ", "))
	}
	if len(in.OpenTabs) > 0 {
		b.WriteString("Open terminals:\n")
		for _, t := range in.OpenTabs {
			line := strings.TrimSpace(t.Title)
			if h := strings.TrimSpace(t.Host); h != "" {
				line += " (" + h + ")"
			}
			if line != "" {
				fmt.Fprintf(&b, "  - %s\n", line)
			}
		}
	}
	note := strings.TrimSpace(in.EngineerNote)
	if note == "" {
		note = strings.TrimSpace(in.Context.EngineerNotes)
	}
	if note != "" {
		fmt.Fprintf(&b, "\nEngineer notes:\n%s\n", note)
	}
	b.WriteString("\nEvidence: scrollback capture attached or saved locally by Pathfinder.\n")
	return b.String()
}
