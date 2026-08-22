// Package cursorapi talks to Cursor account and Cloud Agents HTTP APIs.
//
// Auth: Basic (apiKey as username, empty password) or Bearer — both accepted
// by Cloud Agents. Prefer CURSOR_API_KEY, optionally overridden by Settings.
//
// Docs: https://cursor.com/docs/api · https://cursor.com/docs/cloud-agent/api/endpoints
package cursorapi

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
)

const DefaultBaseURL = "https://api.cursor.com"

// Client is a thin HTTP wrapper; it does not import Fyne.
type Client struct {
	APIKey  string
	BaseURL string
	HTTP    *http.Client
}

// ResolveKey returns explicit, else CURSOR_API_KEY, else empty.
func ResolveKey(explicit string) string {
	if k := strings.TrimSpace(explicit); k != "" {
		return k
	}
	return strings.TrimSpace(os.Getenv("CURSOR_API_KEY"))
}

// New builds a client. Empty apiKey falls back to CURSOR_API_KEY.
func New(apiKey string) *Client {
	return &Client{
		APIKey:  ResolveKey(apiKey),
		BaseURL: DefaultBaseURL,
		HTTP:    &http.Client{Timeout: 60 * time.Second},
	}
}

// Me is the Cloud Agents "API key info" payload (GET /v1/me).
type Me struct {
	APIKeyName string `json:"apiKeyName"`
	UserEmail  string `json:"userEmail"`
	// Extra fields vary by plan; keep raw-friendly via map if needed later.
}

// Model is one entry from GET /v1/models.
type Model struct {
	ID string `json:"id"`
}

// ModelsResponse wraps the models list.
type ModelsResponse struct {
	Models []Model `json:"models"`
}

// CreateAgentRequest is POST /v1/agents (subset of the OpenAPI body).
type CreateAgentRequest struct {
	Prompt CreatePrompt `json:"prompt"`
	Name   string       `json:"name,omitempty"`
	Model  *ModelRef    `json:"model,omitempty"`
	Repos  []RepoSpec   `json:"repos,omitempty"`
}

type CreatePrompt struct {
	Text string `json:"text"`
}

type ModelRef struct {
	ID string `json:"id"`
}

type RepoSpec struct {
	URL         string `json:"url"`
	StartingRef string `json:"startingRef,omitempty"`
}

// CreateAgentResponse is the durable agent + initial run.
type CreateAgentResponse struct {
	Agent Agent `json:"agent"`
	Run   Run   `json:"run"`
}

type Agent struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	URL    string `json:"url,omitempty"`
}

type Run struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

// ErrHTTP is a non-2xx API response.
type ErrHTTP struct {
	Status int
	Body   string
}

func (e *ErrHTTP) Error() string {
	if e == nil {
		return "cursor api error"
	}
	msg := strings.TrimSpace(e.Body)
	if len(msg) > 240 {
		msg = msg[:240] + "…"
	}
	return fmt.Sprintf("cursor api HTTP %d: %s", e.Status, msg)
}

func (c *Client) key() (string, error) {
	k := ResolveKey(c.APIKey)
	if k == "" {
		return "", fmt.Errorf("no Cursor API key (set CURSOR_API_KEY or Settings → Ops → Cursor API key)")
	}
	return k, nil
}

func (c *Client) base() string {
	b := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if b == "" {
		return DefaultBaseURL
	}
	return b
}

func (c *Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	key, err := c.key()
	if err != nil {
		return err
	}
	var rdr io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		rdr = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base()+path, rdr)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	// Bearer is accepted on Cloud Agents; Basic also works org-wide.
	req.Header.Set("Authorization", "Bearer "+key)
	req.SetBasicAuth(key, "")

	res, err := c.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return &ErrHTTP{Status: res.StatusCode, Body: string(data)}
	}
	if out == nil || len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, out)
}

// Me fetches the account / API key identity for the configured key.
func (c *Client) Me(ctx context.Context) (Me, error) {
	var me Me
	err := c.do(ctx, http.MethodGet, "/v1/me", nil, &me)
	return me, err
}

// ListModels returns cloud agent model IDs.
func (c *Client) ListModels(ctx context.Context) ([]Model, error) {
	var out ModelsResponse
	if err := c.do(ctx, http.MethodGet, "/v1/models", nil, &out); err != nil {
		return nil, err
	}
	return out.Models, nil
}

// CreateAgent starts a cloud agent (and its first run).
func (c *Client) CreateAgent(ctx context.Context, req CreateAgentRequest) (CreateAgentResponse, error) {
	var out CreateAgentResponse
	err := c.do(ctx, http.MethodPost, "/v1/agents", req, &out)
	return out, err
}

// GetAgent fetches one agent by id.
func (c *Client) GetAgent(ctx context.Context, id string) (Agent, error) {
	var out Agent
	err := c.do(ctx, http.MethodGet, "/v1/agents/"+strings.TrimSpace(id), nil, &out)
	return out, err
}
