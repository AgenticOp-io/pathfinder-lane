package autotask

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/scottpeterman/pathfinderssh/internal/psasync"
)

// Client calls Datto Autotask REST API.
type Client struct {
	Username           string
	Secret             string
	APIIntegrationCode string
	BaseURL            string
	HTTP               *http.Client
}

func New(user, secret, code, baseURL string) *Client {
	return &Client{
		Username:           strings.TrimSpace(user),
		Secret:             strings.TrimSpace(secret),
		APIIntegrationCode: strings.TrimSpace(code),
		BaseURL:            strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		HTTP:               &http.Client{Timeout: 90 * time.Second},
	}
}

func (c *Client) headers() (map[string]string, string, error) {
	user := c.Username
	if user == "" {
		user = os.Getenv("AUTOTASK_USERNAME")
	}
	secret := c.Secret
	if secret == "" {
		secret = os.Getenv("AUTOTASK_SECRET")
	}
	code := c.APIIntegrationCode
	if code == "" {
		code = os.Getenv("AUTOTASK_API_INTEGRATION_CODE")
	}
	base := strings.TrimRight(c.BaseURL, "/")
	if base == "" {
		base = strings.TrimRight(strings.TrimSpace(os.Getenv("AUTOTASK_BASE_URL")), "/")
	}
	if user == "" || secret == "" || code == "" {
		return nil, "", fmt.Errorf("Autotask username, secret, and API integration code required")
	}
	if base == "" {
		base = "https://webservices.autotask.net/atservicesrest/v1.0"
	}
	return map[string]string{
		"Username":              user,
		"Secret":                secret,
		"ApiIntegrationCode":    code,
		"Accept":                "application/json",
		"Content-Type":          "application/json",
	}, base, nil
}

func (c *Client) Verify(ctx context.Context) error {
	var out struct {
		Items []struct {
			ID int `json:"id"`
		} `json:"items"`
	}
	body := `{"filter":[{"op":"exist","field":"id"}],"MaxRecords":1}`
	return c.postJSON(ctx, "/Companies/query", body, &out)
}

func (c *Client) ListCompanies(ctx context.Context) ([]psasync.Customer, error) {
	body := `{"filter":[{"op":"exist","field":"id"}],"MaxRecords":500}`
	var out struct {
		Items []struct {
			ID     int    `json:"id"`
			Name   string `json:"companyName"`
			Active bool   `json:"isActive"`
		} `json:"items"`
	}
	if err := c.postJSON(ctx, "/Companies/query", body, &out); err != nil {
		return nil, err
	}
	var all []psasync.Customer
	for _, co := range out.Items {
		if !co.Active {
			continue
		}
		all = append(all, psasync.Customer{
			ExternalID: fmt.Sprintf("%d", co.ID),
			Name:       co.Name,
			Tags:       []string{"autotask"},
		})
	}
	return all, nil
}

func (c *Client) postJSON(ctx context.Context, path, body string, dest interface{}) error {
	headers, base, err := c.headers()
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+path, strings.NewReader(body))
	if err != nil {
		return err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.http().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("autotask API %s: %s", resp.Status, strings.TrimSpace(string(b)))
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

func (p PSA) Name() string { return "autotask" }

func (p PSA) ListCustomers(ctx context.Context) ([]psasync.Customer, error) {
	if p.Client == nil {
		return nil, fmt.Errorf("autotask client nil")
	}
	return p.Client.ListCompanies(ctx)
}
