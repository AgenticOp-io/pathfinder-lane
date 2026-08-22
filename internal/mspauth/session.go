package mspauth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/scottpeterman/pathfinderssh/internal/idp"
	"github.com/zalando/go-keyring"
)

const (
	sessionFileName = "auth-session.json"
	keyringService  = "PathfinderSSH-MSP"
)

// UserSession is the signed-in engineer on this Windows profile.
type UserSession struct {
	Provider idp.Provider `json:"provider"`
	Subject  string   `json:"subject,omitempty"`
	Email    string   `json:"email,omitempty"`
	Name     string   `json:"name,omitempty"`
	Roles    []string `json:"roles,omitempty"`
	// AuthenticatedAt is when this profile last completed sign-in.
	AuthenticatedAt time.Time `json:"authenticated_at,omitempty"`
	// ExpiresAt is when refresh should be required (best-effort).
	ExpiresAt time.Time `json:"expires_at,omitempty"`
}

func sessionPath(home string) string {
	return filepath.Join(home, sessionFileName)
}

// LoadUserSession reads the per-user session file.
func LoadUserSession(home string) (UserSession, bool, error) {
	path := sessionPath(home)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return UserSession{}, false, nil
		}
		return UserSession{}, false, err
	}
	var s UserSession
	if err := json.Unmarshal(raw, &s); err != nil {
		return UserSession{}, false, fmt.Errorf("parse auth session: %w", err)
	}
	s.Provider = s.Provider.Normalize()
	return s, true, nil
}

// SaveUserSession persists session metadata (refresh token goes to keyring).
func SaveUserSession(home string, s UserSession) error {
	path := sessionPath(home)
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// ClearUserSession removes session file and keyring refresh token.
func ClearUserSession(home string, enroll Enrollment) error {
	_ = os.Remove(sessionPath(home))
	if enroll.ClientID != "" {
		_ = keyring.Delete(keyringService, refreshKey(enroll))
	}
	return nil
}

func refreshKey(enroll Enrollment) string {
	return "refresh:" + string(enroll.Provider.Normalize()) + ":" + enroll.ClientID
}

// StoreRefreshToken saves OAuth refresh token in OS keyring.
func StoreRefreshToken(enroll Enrollment, token string) error {
	if strings.TrimSpace(token) == "" {
		return nil
	}
	return keyring.Set(keyringService, refreshKey(enroll), token)
}

// LoadRefreshToken reads refresh token from keyring.
func LoadRefreshToken(enroll Enrollment) (string, error) {
	return keyring.Get(keyringService, refreshKey(enroll))
}

// SessionValid reports whether the user session is usable for cloud enrollment.
func SessionValid(enroll Enrollment, sess UserSession, now time.Time) bool {
	if enroll.Provider.Normalize() == ProviderLocal {
		return true
	}
	if strings.TrimSpace(sess.Subject) == "" && strings.TrimSpace(sess.Email) == "" {
		return false
	}
	if sess.Provider.Normalize() != enroll.Provider.Normalize() {
		return false
	}
	if !sess.ExpiresAt.IsZero() && now.After(sess.ExpiresAt) {
		return false
	}
	return true
}

// LoginRequired reports whether engineer must sign in before using the app.
func LoginRequired(enroll Enrollment, sess UserSession, found bool) bool {
	if !enroll.Provider.RequiresCloudLogin() {
		return false
	}
	if enroll.AllowLocalFallback {
		return false
	}
	if !found {
		return true
	}
	return !SessionValid(enroll, sess, time.Now())
}
