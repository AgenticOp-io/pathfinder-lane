package ninja

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

	"github.com/scottpeterman/pathfinderssh/internal/invsync"
)

const defaultBase = "https://api.ninjarmm.com/v2"

// Client calls NinjaOne API (OAuth client credentials).
type Client struct {
	ClientID     string
	ClientSecret string
	BaseURL      string
	HTTP         *http.Client
	accessToken  string
	tokenExpiry  time.Time
}

func New(clientID, secret, baseURL string) *Client {
	return &Client{
		ClientID:     strings.TrimSpace(clientID),
		ClientSecret: strings.TrimSpace(secret),
		BaseURL:      strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		HTTP:         &http.Client{Timeout: 90 * time.Second},
	}
}

func (c *Client) resolve() (id, secret, base string, err error) {
	id = c.ClientID
	if id == "" {
		id = os.Getenv("NINJA_CLIENT_ID")
	}
	secret = c.ClientSecret
	if secret == "" {
		secret = os.Getenv("NINJA_CLIENT_SECRET")
	}
	base = strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		base = strings.TrimRight(strings.TrimSpace(os.Getenv("NINJA_BASE_URL")), "/")
	}
	if base == "" {
		base = defaultBase
	}
	if id == "" || secret == "" {
		return "", "", "", fmt.Errorf("NinjaOne client id and secret required")
	}
	return id, secret, base, nil
}

func (c *Client) token(ctx context.Context) (string, error) {
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		return c.accessToken, nil
	}
	id, secret, base, err := c.resolve()
	if err != nil {
		return "", err
	}
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", id)
	form.Set("client_secret", secret)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.http().Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("ninja auth %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	var out struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	c.accessToken = out.AccessToken
	c.tokenExpiry = time.Now().Add(time.Duration(out.ExpiresIn-60) * time.Second)
	return c.accessToken, nil
}

func (c *Client) Verify(ctx context.Context) error {
	var out []struct {
		ID int `json:"id"`
	}
	return c.getJSON(ctx, "/organizations", &out)
}

func (c *Client) ListDevices(ctx context.Context, orgID string) ([]invsync.Device, error) {
	path := "/devices"
	if strings.TrimSpace(orgID) != "" {
		path = fmt.Sprintf("/organization/%s/devices", strings.TrimSpace(orgID))
	}
	var raw []struct {
		ID   int    `json:"id"`
		Name string `json:"systemName"`
		IP   string `json:"lastIpAddress"`
	}
	if err := c.getJSON(ctx, path, &raw); err != nil {
		return nil, err
	}
	var out []invsync.Device
	for _, d := range raw {
		out = append(out, invsync.Device{
			ID:   fmt.Sprintf("%d", d.ID),
			Name: d.Name,
			Host: d.IP,
		})
	}
	return out, nil
}

func (c *Client) ListOrganizations(ctx context.Context) ([]struct{ ID, Name string }, error) {
	var raw []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	if err := c.getJSON(ctx, "/organizations", &raw); err != nil {
		return nil, err
	}
	var out []struct{ ID, Name string }
	for _, o := range raw {
		out = append(out, struct{ ID, Name string }{ID: fmt.Sprintf("%d", o.ID), Name: o.Name})
	}
	return out, nil
}

func (c *Client) getJSON(ctx context.Context, path string, dest interface{}) error {
	_, _, base, err := c.resolve()
	if err != nil {
		return err
	}
	tok, err := c.token(ctx)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Accept", "application/json")
	resp, err := c.http().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("ninja API %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}

func (c *Client) http() *http.Client {
	if c != nil && c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}
