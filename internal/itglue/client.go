package itglue

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultBase = "https://api.itglue.com"
	contentType = "application/vnd.api+json"
)

// Client calls IT Glue REST API (x-api-key token auth).
type Client struct {
	APIKey  string
	BaseURL string
	HTTP    *http.Client
}

// ResolveAPIKey returns explicit settings or ITGLUE_API_KEY env.
func ResolveAPIKey(key string) string {
	k := strings.TrimSpace(key)
	if k == "" {
		k = strings.TrimSpace(os.Getenv("ITGLUE_API_KEY"))
	}
	return k
}

// ResolveBaseURL returns explicit, ITGLUE_BASE_URL env, or US default.
func ResolveBaseURL(base string) string {
	b := strings.TrimRight(strings.TrimSpace(base), "/")
	if b == "" {
		b = strings.TrimRight(strings.TrimSpace(os.Getenv("ITGLUE_BASE_URL")), "/")
	}
	if b == "" {
		return defaultBase
	}
	return b
}

// New builds a client. API key falls back to ITGLUE_API_KEY.
func New(apiKey, baseURL string) *Client {
	return &Client{
		APIKey:  ResolveAPIKey(apiKey),
		BaseURL: ResolveBaseURL(baseURL),
		HTTP:    &http.Client{Timeout: 90 * time.Second},
	}
}

func (c *Client) httpClient() *http.Client {
	if c != nil && c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

func (c *Client) key() (string, error) {
	k := ResolveAPIKey(c.APIKey)
	if k == "" {
		return "", fmt.Errorf("IT Glue API key missing (Settings → Ops or ITGLUE_API_KEY)")
	}
	return k, nil
}

func (c *Client) base() string {
	return ResolveBaseURL(c.BaseURL)
}

// Verify checks API access with a minimal organizations request.
func (c *Client) Verify(ctx context.Context) error {
	var out listEnvelope
	return c.getJSON(ctx, "/organizations?page[size]=1&page[number]=1", &out)
}

// ListOrganizations returns IT Glue organizations (MSP clients).
func (c *Client) ListOrganizations(ctx context.Context) ([]Organization, error) {
	var all []Organization
	page := 1
	for {
		var out listEnvelope
		path := fmt.Sprintf("/organizations?page[size]=100&page[number]=%d&sort=name", page)
		if err := c.getJSON(ctx, path, &out); err != nil {
			return nil, err
		}
		for _, r := range out.Data {
			name := strings.TrimSpace(r.Attributes.Name)
			if name == "" {
				name = r.ID
			}
			all = append(all, Organization{ID: r.ID, Name: name})
		}
		if out.Meta.NextPage == 0 || out.Meta.NextPage <= page || len(out.Data) == 0 {
			break
		}
		if out.Meta.TotalPages > 0 && page >= out.Meta.TotalPages {
			break
		}
		page = out.Meta.NextPage
		if page <= 0 {
			page++
		}
	}
	return all, nil
}

// ListPasswords returns password metadata for an organization (no plaintext).
func (c *Client) ListPasswords(ctx context.Context, orgID string) ([]Password, error) {
	orgID = strings.TrimSpace(orgID)
	if orgID == "" {
		return nil, fmt.Errorf("organization id required")
	}
	var all []Password
	page := 1
	for {
		var out listEnvelope
		path := fmt.Sprintf("/organizations/%s/relationships/passwords?page[size]=100&page[number]=%d&sort=name",
			orgID, page)
		if err := c.getJSON(ctx, path, &out); err != nil {
			// Fallback route used by some accounts.
			path = fmt.Sprintf("/passwords?filter[organization_id]=%s&page[size]=100&page[number]=%d&sort=name",
				orgID, page)
			if err2 := c.getJSON(ctx, path, &out); err2 != nil {
				return nil, err
			}
		}
		for _, r := range out.Data {
			all = append(all, decodePassword(r))
		}
		if out.Meta.NextPage == 0 || out.Meta.NextPage <= page || len(out.Data) == 0 {
			break
		}
		if out.Meta.TotalPages > 0 && page >= out.Meta.TotalPages {
			break
		}
		page = out.Meta.NextPage
		if page <= 0 {
			page++
		}
	}
	return all, nil
}

// GetPassword fetches one password with plaintext (API key must allow Password Access).
func (c *Client) GetPassword(ctx context.Context, id string) (Password, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Password{}, fmt.Errorf("password id required")
	}
	var out singleEnvelope
	path := "/passwords/" + id + "?show_password=true"
	if err := c.getJSON(ctx, path, &out); err != nil {
		return Password{}, err
	}
	if out.Data.ID == "" {
		return Password{}, fmt.Errorf("password %s not found", id)
	}
	return decodePassword(out.Data), nil
}

// FetchPasswordSecrets loads plaintext for each listed password (show endpoint).
// Secrets are held in memory only — never logged.
func (c *Client) FetchPasswordSecrets(ctx context.Context, list []Password) ([]Password, error) {
	out := make([]Password, 0, len(list))
	for _, p := range list {
		if strings.TrimSpace(p.ID) == "" {
			continue
		}
		full, err := c.GetPassword(ctx, p.ID)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p.Name, err)
		}
		out = append(out, full)
	}
	return out, nil
}

type listEnvelope struct {
	Data []resource `json:"data"`
	Meta pageMeta   `json:"meta"`
}

type singleEnvelope struct {
	Data resource `json:"data"`
}

type pageMeta struct {
	CurrentPage int `json:"current-page"`
	NextPage    int `json:"next-page"`
	TotalPages  int `json:"total-pages"`
	TotalCount  int `json:"total-count"`
}

type resource struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	Attributes passwordAttrs   `json:"attributes"`
}

type passwordAttrs struct {
	Name             string `json:"name"`
	Username         string `json:"username"`
	Password         string `json:"password"`
	OrganizationID   int64  `json:"organization-id"`
	OrganizationName string `json:"organization-name"`
	URL              string `json:"url"`
	CategoryName     string `json:"password-category-name"`
	ResourceURL      string `json:"resource-url"`
}

func decodePassword(r resource) Password {
	a := r.Attributes
	return Password{
		ID:               r.ID,
		Name:             strings.TrimSpace(a.Name),
		Username:         strings.TrimSpace(a.Username),
		Password:         a.Password,
		OrganizationID:   strconv.FormatInt(a.OrganizationID, 10),
		OrganizationName: strings.TrimSpace(a.OrganizationName),
		URL:              strings.TrimSpace(a.URL),
		CategoryName:     strings.TrimSpace(a.CategoryName),
		ResourceURL:      strings.TrimSpace(a.ResourceURL),
	}
}

func (c *Client) getJSON(ctx context.Context, path string, out any) error {
	key, err := c.key()
	if err != nil {
		return err
	}
	full := c.base() + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, full, nil)
	if err != nil {
		return err
	}
	req.Header.Set("x-api-key", key)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", contentType)

	res, err := c.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 8<<20))
	if err != nil {
		return err
	}
	if res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden {
		return fmt.Errorf("IT Glue HTTP %d — check API key and Password Access permission", res.StatusCode)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		msg := strings.TrimSpace(string(data))
		if len(msg) > 240 {
			msg = msg[:240] + "…"
		}
		return fmt.Errorf("IT Glue HTTP %d: %s", res.StatusCode, msg)
	}
	if out == nil || len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, out)
}

// nextPath helper if links-based pagination is needed later.
func nextPath(link string) string {
	link = strings.TrimSpace(link)
	if link == "" {
		return ""
	}
	if strings.HasPrefix(link, "http") {
		u, err := url.Parse(link)
		if err != nil {
			return ""
		}
		return u.Path + "?" + u.RawQuery
	}
	if !strings.HasPrefix(link, "/") {
		return "/" + link
	}
	return link
}
