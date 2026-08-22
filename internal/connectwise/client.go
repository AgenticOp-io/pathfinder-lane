package connectwise

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

// Client calls ConnectWise Manage REST API.
type Client struct {
	CompanyID  string
	PublicKey  string
	PrivateKey string
	ClientID   string
	BaseURL    string
	HTTP       *http.Client
}

func New(companyID, publicKey, privateKey, clientID, baseURL string) *Client {
	return &Client{
		CompanyID:  strings.TrimSpace(companyID),
		PublicKey:  strings.TrimSpace(publicKey),
		PrivateKey: strings.TrimSpace(privateKey),
		ClientID:   strings.TrimSpace(clientID),
		BaseURL:    strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		HTTP:       &http.Client{Timeout: 90 * time.Second},
	}
}

func (c *Client) envOr(field, env string) string {
	if strings.TrimSpace(field) != "" {
		return strings.TrimSpace(field)
	}
	return strings.TrimSpace(os.Getenv(env))
}

func (c *Client) credentials() (companyID, publicKey, privateKey, clientID, base string, err error) {
	companyID = c.envOr(c.CompanyID, "CONNECTWISE_COMPANY_ID")
	publicKey = c.envOr(c.PublicKey, "CONNECTWISE_PUBLIC_KEY")
	privateKey = c.envOr(c.PrivateKey, "CONNECTWISE_PRIVATE_KEY")
	clientID = c.envOr(c.ClientID, "CONNECTWISE_CLIENT_ID")
	base = strings.TrimRight(c.envOr(c.BaseURL, "CONNECTWISE_BASE_URL"), "/")
	if companyID == "" || publicKey == "" || privateKey == "" {
		return "", "", "", "", "", fmt.Errorf("ConnectWise company id, public key, and private key required")
	}
	if base == "" {
		return "", "", "", "", "", fmt.Errorf("ConnectWise API base URL required")
	}
	return companyID, publicKey, privateKey, clientID, base, nil
}

func (c *Client) Verify(ctx context.Context) error {
	var out []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	return c.getJSON(ctx, "/company/companies?pageSize=1", &out)
}

func (c *Client) ListCompanies(ctx context.Context) ([]psasync.Customer, error) {
	var all []psasync.Customer
	page := 1
	for {
		var out []struct {
			ID     int    `json:"id"`
			Name   string `json:"name"`
			Status string `json:"status"`
		}
		path := fmt.Sprintf("/company/companies?page=%d&pageSize=250&orderBy=name", page)
		if err := c.getJSON(ctx, path, &out); err != nil {
			return nil, err
		}
		if len(out) == 0 {
			break
		}
		for _, co := range out {
			if strings.EqualFold(co.Status, "Deleted") {
				continue
			}
			all = append(all, psasync.Customer{
				ExternalID: fmt.Sprintf("%d", co.ID),
				Name:       co.Name,
				Tags:       []string{"connectwise"},
			})
		}
		page++
		if len(out) < 250 {
			break
		}
	}
	return all, nil
}

func (c *Client) getJSON(ctx context.Context, path string, dest interface{}) error {
	companyID, publicKey, privateKey, clientID, base, err := c.credentials()
	if err != nil {
		return err
	}
	u := base + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(companyID+"+"+publicKey, privateKey)
	if clientID != "" {
		req.Header.Set("clientId", clientID)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.http().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("connectwise API %s: %s", resp.Status, strings.TrimSpace(string(b)))
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
type PSA struct {
	Client *Client
}

func (p PSA) Name() string { return "connectwise" }

func (p PSA) ListCustomers(ctx context.Context) ([]psasync.Customer, error) {
	if p.Client == nil {
		return nil, fmt.Errorf("connectwise client nil")
	}
	return p.Client.ListCompanies(ctx)
}
