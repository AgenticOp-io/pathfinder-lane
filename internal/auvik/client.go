// Package auvik is a thin client for Auvik inventory APIs.
//
// Auth: HTTP Basic — Auvik user email as username, API key as password.
// Region base URL: https://auvikapi.<region>.my.auvik.com (us1, eu1, …).
//
// Auvik does not expose device login passwords or a programmatic SSH/terminal
// session API. Inventory sync (IPs, names, tenants) is the integration surface.
package auvik

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
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

// ResolveBaseURL returns explicit, env AUVIK_BASE_URL, or us1 default.
func ResolveBaseURL(base string) string {
	b := strings.TrimRight(strings.TrimSpace(base), "/")
	if b == "" {
		b = strings.TrimRight(strings.TrimSpace(os.Getenv("AUVIK_BASE_URL")), "/")
	}
	if b == "" {
		return defaultBase
	}
	return b
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
		return "", "", fmt.Errorf("Auvik credentials missing (set AUVIK_USERNAME and AUVIK_API_KEY or Settings → Ops)")
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
	// Discovery status helps skip devices Auvik cannot SSH anyway.
	q.Set("include", "deviceDiscoveryStatus")

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
		return fmt.Errorf("Auvik HTTP 401 — check username, API key, and region base URL")
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		msg := strings.TrimSpace(string(data))
		if len(msg) > 240 {
			msg = msg[:240] + "…"
		}
		return fmt.Errorf("Auvik HTTP %d: %s", res.StatusCode, msg)
	}
	if out == nil || len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, out)
}
