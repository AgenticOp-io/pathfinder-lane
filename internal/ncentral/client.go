package ncentral

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

// Client calls N-able N-central REST API (JWT).
type Client struct {
	JWT       string
	ServerURL string
	HTTP      *http.Client
}

func New(jwt, serverURL string) *Client {
	return &Client{
		JWT:       strings.TrimSpace(jwt),
		ServerURL: strings.TrimRight(strings.TrimSpace(serverURL), "/"),
		HTTP:      &http.Client{Timeout: 90 * time.Second},
	}
}

func (c *Client) creds() (jwt, base string, err error) {
	jwt = c.JWT
	if jwt == "" {
		jwt = os.Getenv("NCENTRAL_JWT")
	}
	base = strings.TrimRight(c.ServerURL, "/")
	if base == "" {
		base = strings.TrimRight(strings.TrimSpace(os.Getenv("NCENTRAL_SERVER_URL")), "/")
	}
	if jwt == "" || base == "" {
		return "", "", fmt.Errorf("N-central JWT and server URL required")
	}
	return jwt, base, nil
}

func (c *Client) Verify(ctx context.Context) error {
	var out struct {
		Data []struct {
			ID int `json:"deviceId"`
		} `json:"data"`
	}
	return c.getJSON(ctx, "/api/devices?limit=1", &out)
}

func (c *Client) ListDevices(ctx context.Context) ([]invsync.Device, error) {
	var out struct {
		Data []struct {
			ID     int    `json:"deviceId"`
			Name   string `json:"longName"`
			IP     string `json:"ipAddress"`
		} `json:"data"`
	}
	if err := c.getJSON(ctx, "/api/devices?limit=500", &out); err != nil {
		return nil, err
	}
	var devices []invsync.Device
	for _, d := range out.Data {
		devices = append(devices, invsync.Device{
			ID:   fmt.Sprintf("%d", d.ID),
			Name: d.Name,
			Host: d.IP,
		})
	}
	return devices, nil
}

func (c *Client) getJSON(ctx context.Context, path string, dest interface{}) error {
	jwt, base, err := c.creds()
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Accept", "application/json")
	resp, err := c.http().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("ncentral API %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}

func (c *Client) http() *http.Client {
	if c != nil && c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}
