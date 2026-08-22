package halo

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

	"github.com/scottpeterman/pathfinderssh/internal/psasync"
)

// Client calls Halo PSA REST API (OAuth client credentials).
type Client struct {
	ClientID     string
	ClientSecret string
	Tenant       string // subdomain e.g. agenticops
	BaseURL      string
	HTTP         *http.Client
	accessToken  string
	tokenExpiry  time.Time
}

func New(clientID, secret, tenant, baseURL string) *Client {
	return &Client{
		ClientID:     strings.TrimSpace(clientID),
		ClientSecret: strings.TrimSpace(secret),
		Tenant:       strings.TrimSpace(tenant),
		BaseURL:      strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		HTTP:         &http.Client{Timeout: 90 * time.Second},
	}
}

func (c *Client) resolve() (clientID, secret, base string, err error) {
	clientID = c.ClientID
	if clientID == "" {
		clientID = os.Getenv("HALO_CLIENT_ID")
	}
	secret = c.ClientSecret
	if secret == "" {
		secret = os.Getenv("HALO_CLIENT_SECRET")
	}
	tenant := c.Tenant
	if tenant == "" {
		tenant = os.Getenv("HALO_TENANT")
	}
	base = strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		base = strings.TrimRight(strings.TrimSpace(os.Getenv("HALO_BASE_URL")), "/")
	}
	if base == "" && tenant != "" {
		base = "https://" + tenant + ".halopsa.com"
	}
	if clientID == "" || secret == "" || base == "" {
		return "", "", "", fmt.Errorf("Halo client id, secret, and tenant/base URL required")
	}
	return clientID, secret, base, nil
}

func (c *Client) token(ctx context.Context) (string, error) {
	if c.accessToken != "" && time.Now().Before(c.tokenExpiry) {
		return c.accessToken, nil
	}
	clientID, secret, base, err := c.resolve()
	if err != nil {
		return "", err
	}
	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", clientID)
	form.Set("client_secret", secret)
	form.Set("scope", "all")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/auth/token", strings.NewReader(form.Encode()))
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
		return "", fmt.Errorf("halo auth %s: %s", resp.Status, strings.TrimSpace(string(b)))
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
	var out struct {
		Clients []struct {
			ID int `json:"id"`
		} `json:"clients"`
	}
	return c.getJSON(ctx, "/api/Client?count=1", &out)
}

func (c *Client) ListClients(ctx context.Context) ([]psasync.Customer, error) {
	var out struct {
		Clients []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"clients"`
	}
	if err := c.getJSON(ctx, "/api/Client?count=500", &out); err != nil {
		return nil, err
	}
	var all []psasync.Customer
	for _, cl := range out.Clients {
		all = append(all, psasync.Customer{
			ExternalID: fmt.Sprintf("%d", cl.ID),
			Name:       cl.Name,
			Tags:       []string{"halo"},
		})
	}
	return all, nil
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
		return fmt.Errorf("halo API %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}

func (c *Client) http() *http.Client {
	if c != nil && c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

// PSA implements psasync.Source.
type PSA struct{ Client *Client }

func (p PSA) Name() string { return "halo" }

func (p PSA) ListCustomers(ctx context.Context) ([]psasync.Customer, error) {
	if p.Client == nil {
		return nil, fmt.Errorf("halo client nil")
	}
	return p.Client.ListClients(ctx)
}
