package domotz

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

const defaultBase = "https://api.domotz.com/public-api/v1"

// Client calls Domotz public API.
type Client struct {
	APIKey  string
	BaseURL string
	HTTP    *http.Client
}

func New(apiKey, baseURL string) *Client {
	return &Client{
		APIKey:  strings.TrimSpace(apiKey),
		BaseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		HTTP:    &http.Client{Timeout: 90 * time.Second},
	}
}

func (c *Client) key() (string, error) {
	k := c.APIKey
	if k == "" {
		k = os.Getenv("DOMOTZ_API_KEY")
	}
	if k == "" {
		return "", fmt.Errorf("Domotz API key missing")
	}
	return k, nil
}

func (c *Client) base() string {
	b := strings.TrimRight(c.BaseURL, "/")
	if b == "" {
		b = strings.TrimRight(strings.TrimSpace(os.Getenv("DOMOTZ_BASE_URL")), "/")
	}
	if b == "" {
		return defaultBase
	}
	return b
}

func (c *Client) Verify(ctx context.Context) error {
	var out []struct {
		ID int `json:"id"`
	}
	return c.getJSON(ctx, "/device?status=accepted&limit=1", &out)
}

func (c *Client) ListDevices(ctx context.Context) ([]invsync.Device, error) {
	var raw []struct {
		ID          int    `json:"id"`
		DisplayName string `json:"display_name"`
		IP          string `json:"ip"`
		Vendor      string `json:"vendor"`
		TypeName    string `json:"type_name"`
	}
	if err := c.getJSON(ctx, "/device?status=accepted&limit=500", &raw); err != nil {
		return nil, err
	}
	var out []invsync.Device
	for _, d := range raw {
		out = append(out, invsync.Device{
			ID:         fmt.Sprintf("%d", d.ID),
			Name:       d.DisplayName,
			Host:       d.IP,
			Vendor:     d.Vendor,
			DeviceType: d.TypeName,
		})
	}
	return out, nil
}

func (c *Client) getJSON(ctx context.Context, path string, dest interface{}) error {
	key, err := c.key()
	if err != nil {
		return err
	}
	u := c.base() + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("key", key)
	req.Header.Set("Accept", "application/json")
	resp, err := c.http().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("domotz API %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}

func (c *Client) http() *http.Client {
	if c != nil && c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}
