package passportal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/scottpeterman/pathfinderssh/internal/docvault"
)

const defaultBase = "https://api.passportal.com"

// Client calls N-able Passportal / Documentation API.
type Client struct {
	APIKey  string
	BaseURL string
	Tenant  string
	HTTP    *http.Client
}

func New(apiKey, tenant, baseURL string) *Client {
	return &Client{
		APIKey:  strings.TrimSpace(apiKey),
		Tenant:  strings.TrimSpace(tenant),
		BaseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		HTTP:    &http.Client{Timeout: 90 * time.Second},
	}
}

func (c *Client) creds() (key, tenant, base string, err error) {
	key = c.APIKey
	if key == "" {
		key = os.Getenv("PASSPORTAL_API_KEY")
	}
	tenant = c.Tenant
	if tenant == "" {
		tenant = os.Getenv("PASSPORTAL_TENANT")
	}
	base = strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		base = strings.TrimRight(strings.TrimSpace(os.Getenv("PASSPORTAL_BASE_URL")), "/")
	}
	if base == "" {
		base = defaultBase
	}
	if key == "" {
		return "", "", "", fmt.Errorf("Passportal API key required")
	}
	return key, tenant, base, nil
}

func (c *Client) Verify(ctx context.Context) error {
	var out struct {
		Items []struct {
			ID string `json:"id"`
		} `json:"items"`
	}
	return c.getJSON(ctx, "/v1/passwords?limit=1", &out)
}

func (c *Client) ListPasswords(ctx context.Context) ([]docvault.Password, error) {
	var out struct {
		Items []struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			Username string `json:"username"`
			Password string `json:"password"`
			URL      string `json:"url"`
		} `json:"items"`
	}
	if err := c.getJSON(ctx, "/v1/passwords?limit=500", &out); err != nil {
		return nil, err
	}
	var all []docvault.Password
	for _, p := range out.Items {
		all = append(all, docvault.Password{
			ID:       p.ID,
			Name:     p.Name,
			Username: p.Username,
			Password: p.Password,
			URL:      p.URL,
		})
	}
	return all, nil
}

func (c *Client) getJSON(ctx context.Context, path string, dest interface{}) error {
	key, tenant, base, err := c.creds()
	if err != nil {
		return err
	}
	u := base + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	if tenant != "" {
		req.Header.Set("X-Tenant", tenant)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.http().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("passportal API %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}

func (c *Client) http() *http.Client {
	if c != nil && c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}
