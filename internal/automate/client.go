package automate

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

// Client calls ConnectWise Automate REST API.
type Client struct {
	Username  string
	Password  string
	ServerURL string
	HTTP      *http.Client
}

func New(user, pass, serverURL string) *Client {
	return &Client{
		Username:  strings.TrimSpace(user),
		Password:  strings.TrimSpace(pass),
		ServerURL: strings.TrimRight(strings.TrimSpace(serverURL), "/"),
		HTTP:      &http.Client{Timeout: 90 * time.Second},
	}
}

func (c *Client) creds() (user, pass, base string, err error) {
	user = c.Username
	if user == "" {
		user = os.Getenv("AUTOMATE_USERNAME")
	}
	pass = c.Password
	if pass == "" {
		pass = os.Getenv("AUTOMATE_PASSWORD")
	}
	base = strings.TrimRight(c.ServerURL, "/")
	if base == "" {
		base = strings.TrimRight(strings.TrimSpace(os.Getenv("AUTOMATE_SERVER_URL")), "/")
	}
	if user == "" || pass == "" || base == "" {
		return "", "", "", fmt.Errorf("Automate username, password, and server URL required")
	}
	return user, pass, base, nil
}

func (c *Client) Verify(ctx context.Context) error {
	var out []struct {
		ID int `json:"Id"`
	}
	return c.getJSON(ctx, "/cwa/api/v1/Computers?condition=Id>0&pagesize=1", &out)
}

func (c *Client) ListDevices(ctx context.Context) ([]invsync.Device, error) {
	var raw []struct {
		ID       int    `json:"Id"`
		Name     string `json:"ComputerName"`
		LocalIP  string `json:"LocalIPAddress"`
		ClientID int    `json:"ClientId"`
	}
	if err := c.getJSON(ctx, "/cwa/api/v1/Computers?pagesize=500", &raw); err != nil {
		return nil, err
	}
	var out []invsync.Device
	for _, d := range raw {
		out = append(out, invsync.Device{
			ID:   fmt.Sprintf("%d", d.ID),
			Name: d.Name,
			Host: d.LocalIP,
		})
	}
	return out, nil
}

func (c *Client) getJSON(ctx context.Context, path string, dest interface{}) error {
	user, pass, base, err := c.creds()
	if err != nil {
		return err
	}
	u := base + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(user, pass)
	req.Header.Set("Accept", "application/json")
	resp, err := c.http().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("automate API %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}

func (c *Client) http() *http.Client {
	if c != nil && c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}
