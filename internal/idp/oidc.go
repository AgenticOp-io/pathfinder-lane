package idp

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// LoginInteractive runs OIDC authorization code + PKCE in the system browser.
func LoginInteractive(ctx context.Context, cfg LoginConfig, redirectPort int) (SessionClaims, string, int, error) {
	cfg.Provider = cfg.Provider.Normalize()
	switch cfg.Provider {
	case ProviderEntra, ProviderGoogle:
	default:
		return SessionClaims{}, "", 0, fmt.Errorf("provider %s does not use interactive login", cfg.Provider)
	}
	if err := ValidateLoginConfig(cfg); err != nil {
		return SessionClaims{}, "", 0, err
	}

	verifier, challenge, err := newPKCE()
	if err != nil {
		return SessionClaims{}, "", 0, err
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return SessionClaims{}, "", 0, err
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port
	if redirectPort > 0 {
		ln.Close()
		addr := fmt.Sprintf("127.0.0.1:%d", redirectPort)
		ln, err = net.Listen("tcp", addr)
		if err != nil {
			return SessionClaims{}, "", 0, fmt.Errorf("listen %s: %w", addr, err)
		}
		defer ln.Close()
		port = redirectPort
	}
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", port)

	authURL, tokenURL, scopes := oauthEndpoints(cfg)
	q := url.Values{}
	q.Set("client_id", cfg.ClientID)
	q.Set("response_type", "code")
	q.Set("redirect_uri", redirectURI)
	q.Set("response_mode", "query")
	q.Set("scope", strings.Join(scopes, " "))
	q.Set("code_challenge", challenge)
	q.Set("code_challenge_method", "S256")
	q.Set("prompt", "select_account")

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/callback" {
				http.NotFound(w, r)
				return
			}
			if errMsg := r.URL.Query().Get("error"); errMsg != "" {
				errCh <- fmt.Errorf("identity provider: %s", errMsg)
				return
			}
			code := r.URL.Query().Get("code")
			if code == "" {
				errCh <- fmt.Errorf("identity provider returned no authorization code")
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			io.WriteString(w, "<html><body><p>Signed in. You can close this tab and return to PathfinderSSH MSP.</p></body></html>")
			codeCh <- code
		}),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	defer func() {
		ctxShut, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctxShut)
	}()

	fullAuth := authURL + "?" + q.Encode()
	if err := openBrowser(fullAuth); err != nil {
		return SessionClaims{}, "", 0, fmt.Errorf("open browser: %w (visit %s)", err, fullAuth)
	}

	var code string
	select {
	case <-ctx.Done():
		return SessionClaims{}, "", 0, ctx.Err()
	case err := <-errCh:
		return SessionClaims{}, "", 0, err
	case code = <-codeCh:
	case <-time.After(120 * time.Second):
		return SessionClaims{}, "", 0, fmt.Errorf("sign-in timed out after 120s")
	}

	body := url.Values{}
	body.Set("grant_type", "authorization_code")
	body.Set("client_id", cfg.ClientID)
	body.Set("code", code)
	body.Set("redirect_uri", redirectURI)
	body.Set("code_verifier", verifier)

	tokenRaw, err := postForm(ctx, tokenURL, body)
	if err != nil {
		return SessionClaims{}, "", 0, err
	}
	var tok tokenResponse
	if err := json.Unmarshal(tokenRaw, &tok); err != nil {
		return SessionClaims{}, "", 0, fmt.Errorf("parse token response: %w", err)
	}
	if tok.Error != "" {
		return SessionClaims{}, "", 0, fmt.Errorf("token error: %s", tok.ErrorDescription)
	}
	claims, err := ParseIDTokenClaims(tok.IDToken)
	if err != nil {
		return SessionClaims{}, "", 0, err
	}
	if err := claimsMatchConfig(cfg, claims); err != nil {
		return SessionClaims{}, "", 0, err
	}

	out := SessionClaims{
		Provider: cfg.Provider,
		Subject:  claims.Subject,
		Email:    firstNonEmpty(claims.Email, claims.UPN),
		Name:     claims.Name,
		Roles:    claims.Roles,
	}
	expiresIn := tok.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 8 * 3600
	}
	return out, tok.RefreshToken, expiresIn, nil
}

func ValidateLoginConfig(cfg LoginConfig) error {
	cfg.Provider = cfg.Provider.Normalize()
	switch cfg.Provider {
	case ProviderLocal:
		return nil
	case ProviderEntra:
		if strings.TrimSpace(cfg.ClientID) == "" {
			return fmt.Errorf("Entra application (client) ID is required")
		}
		if strings.TrimSpace(cfg.TenantID) == "" {
			return fmt.Errorf("Entra directory (tenant) ID is required")
		}
		return nil
	case ProviderGoogle:
		if strings.TrimSpace(cfg.ClientID) == "" {
			return fmt.Errorf("Google OAuth client ID is required")
		}
		return nil
	default:
		return fmt.Errorf("unknown provider %q", cfg.Provider)
	}
}

type tokenResponse struct {
	AccessToken      string `json:"access_token"`
	IDToken          string `json:"id_token"`
	RefreshToken     string `json:"refresh_token"`
	ExpiresIn        int    `json:"expires_in"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func oauthEndpoints(cfg LoginConfig) (authURL, tokenURL string, scopes []string) {
	scopes = []string{"openid", "profile", "email", "offline_access"}
	switch cfg.Provider {
	case ProviderEntra:
		return entraEndpoints(cfg)
	case ProviderGoogle:
		return googleEndpoints()
	default:
		return "", "", scopes
	}
}

func postForm(ctx context.Context, tokenURL string, body url.Values) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(body.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("token HTTP %d: %s", res.StatusCode, strings.TrimSpace(string(data)))
	}
	return data, nil
}

func newPKCE() (verifier, challenge string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	verifier = base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge, nil
}

func openBrowser(u string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", u).Start()
	case "darwin":
		return exec.Command("open", u).Start()
	default:
		return exec.Command("xdg-open", u).Start()
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
