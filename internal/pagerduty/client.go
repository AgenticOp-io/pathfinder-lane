package pagerduty

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/scottpeterman/pathfinderssh/internal/incidentbridge"
	"github.com/scottpeterman/pathfinderssh/internal/workcontext"
)

const defaultBase = "https://api.pagerduty.com"

// Client calls PagerDuty REST API for incident notes.
type Client struct {
	APIKey  string
	BaseURL string
	HTTP    *http.Client
}

func New(apiKey, baseURL string) *Client {
	return &Client{
		APIKey:  strings.TrimSpace(apiKey),
		BaseURL: strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		HTTP:    &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *Client) key() (string, error) {
	k := c.APIKey
	if k == "" {
		k = strings.TrimSpace(os.Getenv("PAGERDUTY_API_KEY"))
	}
	if k == "" {
		return "", fmt.Errorf("PagerDuty API key missing")
	}
	return k, nil
}

func (c *Client) base() string {
	b := c.BaseURL
	if b == "" {
		b = strings.TrimRight(strings.TrimSpace(os.Getenv("PAGERDUTY_BASE_URL")), "/")
	}
	if b == "" {
		return defaultBase
	}
	return b
}

// Bridge implements incidentbridge.Bridge.
type Bridge struct{ Client *Client }

func (b Bridge) Name() string { return workcontext.ProviderPagerDuty }

func (b Bridge) Verify(ctx context.Context) error {
	if b.Client == nil {
		return fmt.Errorf("pagerduty client missing")
	}
	var out struct {
		Incidents []struct{ ID string `json:"id"` } `json:"incidents"`
	}
	return b.Client.getJSON(ctx, "/incidents?limit=1", &out)
}

func (b Bridge) PostDocument(ctx context.Context, req incidentbridge.DocumentRequest) error {
	if b.Client == nil {
		return fmt.Errorf("pagerduty client missing")
	}
	id := workcontext.NormalizeIncidentID(req.IncidentID)
	content := strings.TrimSpace(req.Summary)
	if len(req.FileBytes) > 0 {
		name := strings.TrimSpace(req.FileName)
		if name == "" {
			name = "evidence.zip"
		}
		content += fmt.Sprintf("\n\nEvidence pack: %s (%d bytes). "+
			"PagerDuty REST notes do not accept file uploads; "+
			"attach the zip in the PagerDuty UI or your PSA if required.\n", name, len(req.FileBytes))
	}
	if len(content) > 8000 {
		content = content[:8000] + "\n…(truncated)"
	}
	return b.Client.postNote(ctx, id, content)
}

func (c *Client) postNote(ctx context.Context, incidentID, content string) error {
	body := map[string]interface{}{
		"note": map[string]string{"content": content},
	}
	path := fmt.Sprintf("/incidents/%s/notes", incidentID)
	return c.postJSON(ctx, path, body, nil)
}

func (c *Client) getJSON(ctx context.Context, path string, dest interface{}) error {
	key, err := c.key()
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base()+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Token token="+key)
	req.Header.Set("Accept", "application/vnd.pagerduty+json")
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pagerduty %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	if dest != nil {
		return json.NewDecoder(resp.Body).Decode(dest)
	}
	return nil
}

func (c *Client) postJSON(ctx context.Context, path string, payload interface{}, dest interface{}) error {
	key, err := c.key()
	if err != nil {
		return err
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base()+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Token token="+key)
	req.Header.Set("Accept", "application/vnd.pagerduty+json")
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("pagerduty %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	if dest != nil {
		return json.NewDecoder(resp.Body).Decode(dest)
	}
	return nil
}
