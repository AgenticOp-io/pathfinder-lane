// Package pfbridge exposes a localhost HTTP API so Cursor IDE (via MCP) can
// list Pathfinder SSH sessions, read scrollback, and optionally send commands.
package pfbridge

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	DefaultPort       = 19790
	DefaultBind       = "127.0.0.1"
	StateFileName     = "msp-bridge.json"
	legacyStateFile   = "mcp-bridge.json"
	maxScrollback     = 200_000
	defaultScrollback = 24_000
)

// Session is one open terminal tab visible to Cursor.
type Session struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Customer string `json:"customer,omitempty"`
	Folder   string `json:"folder,omitempty"`
	Target   string `json:"target,omitempty"`
	Active   bool   `json:"active"`
	Kind     string `json:"kind"`
}

// Backend is implemented by the Pathfinder host (UI thread marshaling inside).
type Backend interface {
	ListSessions() []Session
	ActiveSession() (Session, bool)
	Scrollback(id string, maxChars int) (string, error)
	Send(id string, text string) error
}

// Config controls the localhost listener.
type Config struct {
	Enabled   bool
	Bind      string
	Port      int
	Token     string
	AllowSend bool
	AppHome   string // where msp-bridge.json is written
}

// StateFile is what pathfinder-msp reads to find the running app.
type StateFile struct {
	URL       string `json:"url"`
	Token     string `json:"token"`
	AllowSend bool   `json:"allow_send"`
	PID       int    `json:"pid"`
	Updated   string `json:"updated"`
}

// Server is the localhost bridge.
type Server struct {
	cfg Config
	be  Backend

	mu   sync.Mutex
	http *http.Server
	ln   net.Listener
}

// New builds a server (not started).
func New(cfg Config, be Backend) *Server {
	if cfg.Bind == "" {
		cfg.Bind = DefaultBind
	}
	if cfg.Port <= 0 {
		cfg.Port = DefaultPort
	}
	return &Server{cfg: cfg, be: be}
}

// EnsureToken returns cfg.Token or a new random token.
func EnsureToken(existing string) string {
	if strings.TrimSpace(existing) != "" {
		return strings.TrimSpace(existing)
	}
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return hex.EncodeToString([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	}
	return hex.EncodeToString(b[:])
}

// DefaultAppHome is ~/.pathfinderssh (same as the desktop app).
func DefaultAppHome() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "."
	}
	dir := filepath.Join(home, ".pathfinderssh")
	_ = os.MkdirAll(dir, 0o700)
	return dir
}

// StatePath is AppHome/msp-bridge.json.
func StatePath(appHome string) string {
	return filepath.Join(appHome, StateFileName)
}

// WriteState publishes connection details for pathfinder-msp.
func WriteState(appHome string, st StateFile) error {
	if err := os.MkdirAll(appHome, 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(StatePath(appHome), raw, 0o600)
}

// ClearState removes the discovery file (best-effort).
func ClearState(appHome string) {
	_ = os.Remove(StatePath(appHome))
	_ = os.Remove(filepath.Join(appHome, legacyStateFile))
}

// LoadState reads msp-bridge.json (falls back to legacy mcp-bridge.json).
func LoadState(appHome string) (StateFile, error) {
	raw, err := os.ReadFile(StatePath(appHome))
	if err != nil {
		raw, err = os.ReadFile(filepath.Join(appHome, legacyStateFile))
		if err != nil {
			return StateFile{}, err
		}
	}
	var st StateFile
	if err := json.Unmarshal(raw, &st); err != nil {
		return StateFile{}, err
	}
	return st, nil
}

// Start listens on 127.0.0.1 and serves the API.
func (s *Server) Start() error {
	if s == nil || s.be == nil {
		return fmt.Errorf("pfbridge: nil server")
	}
	if !s.cfg.Enabled {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.http != nil {
		return nil
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.withAuth(s.handleHealth))
	mux.HandleFunc("/v1/health", s.withAuth(s.handleHealth))
	mux.HandleFunc("/v1/sessions", s.withAuth(s.handleSessions))
	mux.HandleFunc("/v1/sessions/active", s.withAuth(s.handleActive))
	mux.HandleFunc("/v1/sessions/", s.withAuth(s.handleSessionSub))
	mux.HandleFunc("/v1/send", s.withAuth(s.handleSend))

	addr := net.JoinHostPort(s.cfg.Bind, strconv.Itoa(s.cfg.Port))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("pfbridge listen %s: %w", addr, err)
	}
	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	s.ln = ln
	s.http = srv

	if s.cfg.AppHome != "" {
		_ = WriteState(s.cfg.AppHome, StateFile{
			URL:       "http://" + addr,
			Token:     s.cfg.Token,
			AllowSend: s.cfg.AllowSend,
			PID:       os.Getpid(),
			Updated:   time.Now().UTC().Format(time.RFC3339),
		})
	}

	go func() {
		_ = srv.Serve(ln)
	}()
	return nil
}

// Stop closes the listener and clears the state file.
func (s *Server) Stop() {
	if s == nil {
		return
	}
	s.mu.Lock()
	srv := s.http
	s.http = nil
	s.ln = nil
	home := s.cfg.AppHome
	s.mu.Unlock()
	if srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		_ = srv.Shutdown(ctx)
		cancel()
	}
	if home != "" {
		ClearState(home)
	}
}

// Addr returns the bound address, or empty when not listening.
func (s *Server) Addr() string {
	if s == nil || s.ln == nil {
		return ""
	}
	return s.ln.Addr().String()
}

func (s *Server) withAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			if host != "127.0.0.1" && host != "::1" && host != "localhost" {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
		}
		tok := strings.TrimSpace(s.cfg.Token)
		if tok != "" {
			got := strings.TrimSpace(r.Header.Get("Authorization"))
			if strings.HasPrefix(strings.ToLower(got), "bearer ") {
				got = strings.TrimSpace(got[7:])
			} else {
				got = strings.TrimSpace(r.Header.Get("X-Pathfinder-Token"))
			}
			if got != tok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}
		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":         true,
		"service":    "pathfinder-msp-bridge",
		"allow_send": s.cfg.AllowSend,
	})
}

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"sessions": s.be.ListSessions()})
}

func (s *Server) handleActive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	sess, ok := s.be.ActiveSession()
	if !ok {
		writeJSON(w, http.StatusOK, map[string]any{"session": nil})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"session": sess})
}

func (s *Server) handleSessionSub(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/sessions/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	id := parts[0]
	if len(parts) == 1 && parts[0] != "active" {
		http.NotFound(w, r)
		return
	}
	if len(parts) >= 2 && parts[1] == "scrollback" {
		if r.Method != http.MethodGet {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		max := defaultScrollback
		if q := r.URL.Query().Get("max"); q != "" {
			if n, err := strconv.Atoi(q); err == nil && n > 0 {
				max = n
			}
		}
		if max > maxScrollback {
			max = maxScrollback
		}
		text, err := s.be.Scrollback(id, max)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"id":         id,
			"scrollback": text,
			"chars":      len(text),
		})
		return
	}
	if len(parts) >= 2 && parts[1] == "send" {
		s.handleSendTo(w, r, id)
		return
	}
	http.NotFound(w, r)
}

type sendBody struct {
	Text      string `json:"text"`
	SessionID string `json:"session_id"`
	Confirm   bool   `json:"confirm"`
}

func (s *Server) handleSend(w http.ResponseWriter, r *http.Request) {
	s.handleSendTo(w, r, "")
}

func (s *Server) handleSendTo(w http.ResponseWriter, r *http.Request, id string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	if !s.cfg.AllowSend {
		http.Error(w, "send disabled (enable Cursor allow-send in PathfinderSSH MSP settings)", http.StatusForbidden)
		return
	}
	var body sendBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(body.Text) == "" {
		http.Error(w, "text required", http.StatusBadRequest)
		return
	}
	if !body.Confirm {
		http.Error(w, "confirm=true required to send to a live SSH session", http.StatusBadRequest)
		return
	}
	if id == "" {
		id = strings.TrimSpace(body.SessionID)
	}
	if err := s.be.Send(id, body.Text); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "session_id": id})
}
