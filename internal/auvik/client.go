// Package auvik is a thin client for Auvik inventory APIs.
//
// Auth: HTTP Basic — Auvik user email as username, API key as password.
// Region base URL: https://auvikapi.<region>.my.auvik.com (us1, eu1, …).
//
// Auvik does not expose device login passwords or a programmatic SSH/terminal
// session API. Inventory sync (IPs, names, tenants) is the integration surface.
package auvik

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

const defaultBase = "https://auvikapi.us1.my.auvik.com"

// Client calls Auvik REST v1.
type Client struct {
	Username string
	APIKey   string
	BaseURL  string
	HTTP     *http.Client
}

// ResolveCredentials returns explicit settings, else AUVIK_USERNAME + AUVIK_API_KEY.
func ResolveCredentials(username, apiKey string) (user, key string) {
	user = strings.TrimSpace(username)
	key = strings.TrimSpace(apiKey)
	if user == "" {
		user = strings.TrimSpace(os.Getenv("AUVIK_USERNAME"))
	}
	if key == "" {
		key = strings.TrimSpace(os.Getenv("AUVIK_API_KEY"))
	}
	return user, key
}

var (
	apiHostRE    = regexp.MustCompile(`(?i)^auvikapi\.([a-z]{2}\d+)\.my\.auvik\.com$`)
	regionHostRE = regexp.MustCompile(`(?i)(?:^|\.)([a-z]{2}\d+)\.my\.auvik\.com$`)
	regionOnlyRE = regexp.MustCompile(`(?i)^[a-z]{2}\d+$`)
)

// ResolveBaseURL returns a normalized API origin (no path, no trailing slash).
// Accepts dashboard URLs (including MSP vanity hosts like
// https://acme.us2.my.auvik.com/#...), API URLs, or a bare region code.
func ResolveBaseURL(base string) string {
	b := strings.TrimSpace(base)
	if b == "" {
		b = strings.TrimSpace(os.Getenv("AUVIK_BASE_URL"))
	}
	b = strings.TrimSpace(b)
	if b == "" {
		return defaultBase
	}

	if regionOnlyRE.MatchString(b) {
		return apiOrigin(strings.ToLower(b))
	}

	// Strip fragments/queries early so dashboard deep-links still parse.
	if i := strings.IndexAny(b, "#?"); i >= 0 {
		b = b[:i]
	}
	b = strings.TrimRight(b, "/")

	if regionOnlyRE.MatchString(b) {
		return apiOrigin(strings.ToLower(b))
	}

	u, err := url.Parse(b)
	if err != nil || u.Host == "" {
		// Allow host-only paste without scheme.
		u, err = url.Parse("https://" + strings.TrimPrefix(b, "//"))
	}
	if err == nil && u.Host != "" {
		host := strings.ToLower(u.Host)
		if m := apiHostRE.FindStringSubmatch(host); len(m) == 2 {
			return apiOrigin(m[1])
		}
		if m := regionHostRE.FindStringSubmatch(host); len(m) == 2 {
			return apiOrigin(m[1])
		}
	}

	// Last resort: find region token before .my.auvik.com in the raw string.
	if m := regexp.MustCompile(`(?i)([a-z]{2}\d+)\.my\.auvik\.com`).FindStringSubmatch(b); len(m) == 2 {
		return apiOrigin(m[1])
	}

	return strings.TrimRight(b, "/")
}

func apiOrigin(region string) string {
	return "https://auvikapi." + strings.ToLower(region) + ".my.auvik.com"
}

// New builds a client. Empty username/key fall back to env.
func New(username, apiKey, baseURL string) *Client {
	user, key := ResolveCredentials(username, apiKey)
	return &Client{
		Username: user,
		APIKey:   key,
		BaseURL:  ResolveBaseURL(baseURL),
		HTTP:     &http.Client{Timeout: 90 * time.Second},
	}
}

func (c *Client) httpClient() *http.Client {
	if c != nil && c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

func (c *Client) creds() (string, string, error) {
	user, key := ResolveCredentials(c.Username, c.APIKey)
	if user == "" || key == "" {
		return "", "", fmt.Errorf("Auvik credentials missing (set AUVIK_USERNAME and AUVIK_API_KEY or Settings → Integrations)")
	}
	return user, key, nil
}

func (c *Client) base() string {
	return ResolveBaseURL(c.BaseURL)
}

// Verify checks GET /v1/authentication/verify.
func (c *Client) Verify(ctx context.Context) error {
	return c.do(ctx, http.MethodGet, "/v1/authentication/verify", nil, nil)
}

// ListTenants returns client tenants (Auvik "tenant" = MSP client site).
func (c *Client) ListTenants(ctx context.Context) ([]Tenant, error) {
	var out tenantsEnvelope
	if err := c.do(ctx, http.MethodGet, "/v1/tenants", nil, &out); err != nil {
		return nil, err
	}
	tenants := make([]Tenant, 0, len(out.Data))
	for _, r := range out.Data {
		t := Tenant{ID: r.ID, Name: strings.TrimSpace(r.Attributes.DomainPrefix)}
		if t.Name == "" {
			t.Name = r.ID
		}
		tenants = append(tenants, t)
	}
	return tenants, nil
}

// ListDevices fetches inventory for one or more tenant IDs.
func (c *Client) ListDevices(ctx context.Context, tenantIDs []string, pageSize int) ([]Device, error) {
	if len(tenantIDs) == 0 {
		return nil, fmt.Errorf("tenant id required")
	}
	if pageSize <= 0 || pageSize > 300 {
		pageSize = 300
	}
	q := url.Values{}
	q.Set("tenants", strings.Join(tenantIDs, ","))
	q.Set("page[first]", fmt.Sprintf("%d", pageSize))
	// Auvik device/info only accepts include=deviceDetail (not discovery status).
	q.Set("include", "deviceDetail")

	var all []Device
	path := "/v1/inventory/device/info?" + q.Encode()
	for path != "" {
		var page devicesEnvelope
		if err := c.do(ctx, http.MethodGet, path, nil, &page); err != nil {
			return nil, err
		}
		for _, r := range page.Data {
			all = append(all, decodeDevice(r, page.Included))
		}
		path = nextPath(page.Links.Next)
	}
	return all, nil
}

func nextPath(link string) string {
	link = strings.TrimSpace(link)
	if link == "" {
		return ""
	}
	if strings.HasPrefix(link, "http") {
		u, err := url.Parse(link)
		if err != nil {
			return ""
		}
		return u.Path + "?" + u.RawQuery
	}
	if !strings.HasPrefix(link, "/") {
		return "/" + link
	}
	return link
}

func (c *Client) do(ctx context.Context, method, path string, body io.Reader, out any) error {
	user, key, err := c.creds()
	if err != nil {
		return err
	}
	full := c.base() + path
	req, err := http.NewRequestWithContext(ctx, method, full, body)
	if err != nil {
		return err
	}
	req.SetBasicAuth(user, key)
	req.Header.Set("Accept", "application/json")
	res, err := c.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return err
	}
	if res.StatusCode == http.StatusUnauthorized {
		return fmt.Errorf("Auvik HTTP 401 — check email, API key, and region (base URL must be https://auvikapi.<region>.my.auvik.com, not the dashboard)")
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("%s", formatHTTPError(res.StatusCode, full, data))
	}
	if out == nil || len(data) == 0 {
		return nil
	}
	trim := bytes.TrimSpace(data)
	if len(trim) == 0 {
		return nil
	}
	// Dashboard / wrong-host responses can be HTTP 200 with an HTML body.
	if trim[0] == '<' || looksLikeHTML(trim) {
		return fmt.Errorf("%s", formatHTTPError(http.StatusNotFound, full, data))
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("Auvik response from %s is not JSON (%v) — check API base URL is https://auvikapi.<region>.my.auvik.com", full, err)
	}
	return nil
}

func looksLikeHTML(data []byte) bool {
	s := strings.ToLower(string(data))
	return strings.Contains(s, "<html") || strings.Contains(s, "<!doctype")
}

func formatHTTPError(code int, fullURL string, data []byte) string {
	msg := strings.TrimSpace(string(data))
	looksHTML := strings.Contains(strings.ToLower(msg), "<html") || strings.Contains(strings.ToLower(msg), "<!doctype")
	if looksHTML || code == http.StatusNotFound {
		return fmt.Sprintf("Auvik HTTP %d for %s — use API base https://auvikapi.<region>.my.auvik.com (from your Auvik dashboard URL region, e.g. us1 → https://auvikapi.us1.my.auvik.com). Do not paste the dashboard or /v1 path.", code, fullURL)
	}
	if len(msg) > 240 {
		msg = msg[:240] + "…"
	}
	if msg == "" {
		msg = http.StatusText(code)
	}
	return fmt.Sprintf("Auvik HTTP %d: %s", code, msg)
}
