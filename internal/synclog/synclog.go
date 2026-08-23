// Package synclog persists MSP inventory sync events for troubleshooting.
package synclog

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const fileName = "msp-sync.log"

var mu sync.Mutex

// Entry is one sync event line (JSONL on disk).
type Entry struct {
	Time    time.Time `json:"time"`
	Source  string    `json:"source"` // auvik, ninja, …
	Level   string    `json:"level"`  // info, warn, error
	Client  string    `json:"client,omitempty"`
	Message string    `json:"message"`
	Detail  string    `json:"detail,omitempty"`
}

// Dir is ~/.pathfinderssh/logs (created on write).
func Dir(appHome string) string {
	return filepath.Join(appHome, "logs")
}

// Path is the sync log file under app home.
func Path(appHome string) string {
	return filepath.Join(Dir(appHome), fileName)
}

// Append writes one entry.
func Append(appHome string, e Entry) error {
	if strings.TrimSpace(appHome) == "" {
		return fmt.Errorf("app home required")
	}
	if e.Time.IsZero() {
		e.Time = time.Now()
	}
	if e.Level == "" {
		e.Level = "info"
	}
	if e.Source == "" {
		e.Source = "msp"
	}
	raw, err := json.Marshal(e)
	if err != nil {
		return err
	}
	mu.Lock()
	defer mu.Unlock()
	if err := os.MkdirAll(Dir(appHome), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(Path(appHome), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(raw, '\n'))
	return err
}

// Info / Warn / Error helpers.
func Info(appHome, source, client, message, detail string) {
	_ = Append(appHome, Entry{Source: source, Level: "info", Client: client, Message: message, Detail: detail})
}

func Warn(appHome, source, client, message, detail string) {
	_ = Append(appHome, Entry{Source: source, Level: "warn", Client: client, Message: message, Detail: detail})
}

func Error(appHome, source, client, message, detail string) {
	_ = Append(appHome, Entry{Source: source, Level: "error", Client: client, Message: message, Detail: detail})
}

// ReadTail returns the last maxBytes of the log as plain text (newest last).
func ReadTail(appHome string, maxBytes int) (string, error) {
	if maxBytes <= 0 {
		maxBytes = 256 * 1024
	}
	path := Path(appHome)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "(no sync log yet — run an Auvik or inventory sync first)\n", nil
		}
		return "", err
	}
	if len(raw) > maxBytes {
		raw = raw[len(raw)-maxBytes:]
		if i := strings.IndexByte(string(raw), '\n'); i >= 0 && i+1 < len(raw) {
			raw = raw[i+1:]
		}
	}
	var b strings.Builder
	sc := bufio.NewScanner(strings.NewReader(string(raw)))
	buf := make([]byte, 0, 64*1024)
	sc.Buffer(buf, 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var e Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			b.WriteString(line)
			b.WriteByte('\n')
			continue
		}
		b.WriteString(formatEntry(e))
		b.WriteByte('\n')
	}
	if err := sc.Err(); err != nil {
		return b.String(), err
	}
	if b.Len() == 0 {
		return "(sync log is empty)\n", nil
	}
	return b.String(), nil
}

func formatEntry(e Entry) string {
	ts := e.Time.Local().Format("2006-01-02 15:04:05")
	lvl := strings.ToUpper(e.Level)
	src := e.Source
	if e.Client != "" {
		src += "/" + e.Client
	}
	msg := fmt.Sprintf("[%s] %-5s %s — %s", ts, lvl, src, e.Message)
	if d := strings.TrimSpace(e.Detail); d != "" {
		msg += "\n    " + strings.ReplaceAll(d, "\n", "\n    ")
	}
	return msg
}

// Clear truncates the sync log.
func Clear(appHome string) error {
	mu.Lock()
	defer mu.Unlock()
	path := Path(appHome)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// AuvikHint returns a short remediation tip for common Auvik API errors.
func AuvikHint(errText string) string {
	s := strings.ToLower(errText)
	switch {
	case strings.Contains(s, "include") && strings.Contains(s, "devicedetail"):
		return "API include parameter rejected — update Pathfinder (device list must use include=deviceDetail)."
	case strings.Contains(s, "403") || strings.Contains(s, "do not have permission"):
		return "No permission on that Auvik tenant (often the MSP root). Uncheck it on Sync selected, or grant the API user access in Auvik."
	case strings.Contains(s, "401") || strings.Contains(s, "unauthorized"):
		return "Check Auvik username + API key under Settings → Integrations."
	case strings.Contains(s, "html") || strings.Contains(s, "invalid character '<'"):
		return "Wrong base URL (dashboard vs API). Use https://auvikapi.<region>.my.auvik.com — not the my.auvik.com UI URL."
	case strings.Contains(s, "404"):
		return "Check Auvik API base URL / region under Settings → Integrations."
	case strings.Contains(s, "timeout") || strings.Contains(s, "deadline"):
		return "Sync timed out — retry Sync selected for fewer clients, or try again later."
	case strings.Contains(s, "auviktunnel not found") || strings.Contains(s, "binary not found"):
		return "Install AuvikTunnel from Auvik, then set path under Settings → Integrations."
	case strings.Contains(s, "no client domain"):
		return "Re-run Sync from Auvik so tenant domain is stored on sessions."
	case strings.Contains(s, "did not open local port"):
		return "AuvikTunnel failed to listen — open logs/auvik-tunnel.log; ensure you are signed into Auvik."
	default:
		return ""
	}
}
