package hudu

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

	"github.com/scottpeterman/pathfinderssh/internal/docvault"
)

const defaultBase = "https://api.hudu.com"

// Client calls Hudu REST API (x-api-key).
type Client struct {
	APIKey  string
	BaseURL string
	HTTP    *http.Client
}

func ResolveAPIKey(key string) string {
	k := strings.TrimSpace(key)
	if k == "" {
		k = strings.TrimSpace(os.Getenv("HUDU_API_KEY"))
	}
	return k
}

func ResolveBaseURL(base string) string {
	b := strings.TrimRight(strings.TrimSpace(base), "/")
	if b == "" {
		b = strings.TrimRight(strings.TrimSpace(os.Getenv("HUDU_BASE_URL")), "/")
	}
	if b == "" {
		return defaultBase
	}
	return b
}

func New(apiKey, baseURL string) *Client {
	return &Client{
		APIKey:  ResolveAPIKey(apiKey),
		BaseURL: ResolveBaseURL(baseURL),
		HTTP:    &http.Client{Timeout: 90 * time.Second},
	}
}

type Company struct {
	ID   string
	Name string
}

func (c *Client) Verify(ctx context.Context) error {
	var out struct {
		Companies []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"companies"`
	}
	return c.getJSON(ctx, "/api/v1/companies?page=1", &out)
}

func (c *Client) ListCompanies(ctx context.Context) ([]Company, error) {
	var all []Company
	page := 1
	for {
		var out struct {
			Companies []struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			} `json:"companies"`
		}
		path := fmt.Sprintf("/api/v1/companies?page=%d", page)
		if err := c.getJSON(ctx, path, &out); err != nil {
			return nil, err
		}
		if len(out.Companies) == 0 {
			break
		}
		for _, co := range out.Companies {
			all = append(all, Company{ID: fmt.Sprintf("%d", co.ID), Name: co.Name})
		}
		page++
		if len(out.Companies) < 25 {
			break
		}
	}
	return all, nil
}

func (c *Client) ListPasswords(ctx context.Context, companyID string) ([]docvault.Password, error) {
	companyID = strings.TrimSpace(companyID)
	if companyID == "" {
		return nil, fmt.Errorf("company id required")
	}
	var out struct {
		Passwords []struct {
			ID       int    `json:"id"`
			Name     string `json:"name"`
			Username string `json:"username"`
			Password string `json:"password"`
			URL      string `json:"url"`
		} `json:"passwords"`
	}
	path := fmt.Sprintf("/api/v1/companies/%s/assets/passwords?show_password=true", url.PathEscape(companyID))
	if err := c.getJSON(ctx, path, &out); err != nil {
		return nil, err
	}
	var all []docvault.Password
	for _, p := range out.Passwords {
		all = append(all, docvault.Password{
			ID:       fmt.Sprintf("%d", p.ID),
			Name:     p.Name,
			Username: p.Username,
			Password: p.Password,
			URL:      p.URL,
		})
	}
	return all, nil
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
	req.Header.Set("x-api-key", key)
	req.Header.Set("Accept", "application/json")
	resp, err := c.http().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("hudu API %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}

func (c *Client) key() (string, error) {
	k := ResolveAPIKey(c.APIKey)
	if k == "" {
		return "", fmt.Errorf("Hudu API key missing (Settings → Tools or HUDU_API_KEY)")
	}
	return k, nil
}

func (c *Client) base() string {
	return ResolveBaseURL(c.BaseURL)
}

func (c *Client) http() *http.Client {
	if c != nil && c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}
