package dattormm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/scottpeterman/pathfinderssh/internal/invsync"
)

const defaultBase = "https://zinfandel-api.centrastage.net/api/v2"

// Client calls Datto RMM API.
type Client struct {
	APIKey  string
	Secret  string
	BaseURL string
	HTTP    *http.Client
}

func New(apiKey, secret, baseURL string) *Client {
	return &Client{
		APIKey:  strings.TrimSpace(apiKey),
		Secret:  strings.TrimSpace(secret),
		BaseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		HTTP:    &http.Client{Timeout: 90 * time.Second},
	}
}

func (c *Client) creds() (key, secret, base string, err error) {
	key = c.APIKey
	if key == "" {
		key = os.Getenv("DATTO_API_KEY")
	}
	secret = c.Secret
	if secret == "" {
		secret = os.Getenv("DATTO_SECRET")
	}
	base = strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		base = strings.TrimRight(strings.TrimSpace(os.Getenv("DATTO_BASE_URL")), "/")
	}
	if base == "" {
		base = defaultBase
	}
	if key == "" || secret == "" {
		return "", "", "", fmt.Errorf("Datto RMM API key and secret required")
	}
	return key, secret, base, nil
}

func (c *Client) Verify(ctx context.Context) error {
	var out struct {
		Sites []struct {
			UID string `json:"uid"`
		} `json:"sites"`
	}
	return c.getJSON(ctx, "/account/sites?pageSize=1", &out)
}

func (c *Client) ListDevices(ctx context.Context, siteUID string) ([]invsync.Device, error) {
	path := "/account/devices"
	if strings.TrimSpace(siteUID) != "" {
		path = fmt.Sprintf("/account/sites/%s/devices", strings.TrimSpace(siteUID))
	}
	var raw []struct {
		UID  string `json:"uid"`
		Name string `json:"hostname"`
		IP   string `json:"intIpAddress"`
	}
	if err := c.getJSON(ctx, path, &raw); err != nil {
		return nil, err
	}
	var out []invsync.Device
	for _, d := range raw {
		out = append(out, invsync.Device{
			ID:   d.UID,
			Name: d.Name,
			Host: d.IP,
		})
	}
	return out, nil
}

func (c *Client) ListSites(ctx context.Context) ([]struct{ ID, Name string }, error) {
	var out struct {
		Sites []struct {
			UID  string `json:"uid"`
			Name string `json:"name"`
		} `json:"sites"`
	}
	if err := c.getJSON(ctx, "/account/sites?pageSize=250", &out); err != nil {
		return nil, err
	}
	var sites []struct{ ID, Name string }
	for _, s := range out.Sites {
		sites = append(sites, struct{ ID, Name string }{ID: s.UID, Name: s.Name})
	}
	return sites, nil
}

func (c *Client) getJSON(ctx context.Context, path string, dest interface{}) error {
	key, secret, base, err := c.creds()
	if err != nil {
		return err
	}
	u := base + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("ApiKey", key)
	req.Header.Set("Secret", secret)
	req.Header.Set("Accept", "application/json")
	resp, err := c.http().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("datto API %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}

func (c *Client) http() *http.Client {
	if c != nil && c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}
