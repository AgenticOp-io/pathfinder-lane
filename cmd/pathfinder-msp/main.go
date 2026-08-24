package main

// pathfinder-msp is the Cursor IDE stdio bridge to a running PathfinderSSH MSP
// window (localhost HTTP API written to ~/.pathfinderssh/msp-bridge.json).
//
// Configure in Cursor → Tools/MCP, e.g. %USERPROFILE%\.cursor\mcp.json:
//
//	{
//	  "mcpServers": {
//	    "pathfinder-msp": {
//	      "command": "C:\\Users\\…\\PathfinderSSH-MSP\\bin\\pathfinder-msp.exe"
//	    }
//	  }
//	}
//
// Tools: pathfinder_list_sessions, pathfinder_active_session,
// pathfinder_read_scrollback, pathfinder_send_command.

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/scottpeterman/pathfinderssh/internal/pfbridge"
)

func main() {
	if err := run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "pathfinder-msp: %v\n", err)
		os.Exit(1)
	}
}

type rpcReq struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcResp struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id,omitempty"`
	Result  any    `json:"result,omitempty"`
	Error   *rpcErr `json:"error,omitempty"`
}

type rpcErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type client struct {
	http  *http.Client
	base  string
	token string
}

func run(in io.Reader, out io.Writer) error {
	st, err := loadBridgeState()
	if err != nil {
		// Still speak MCP so Cursor can show the error via tools/call.
		st = pfbridge.StateFile{}
	}
	c := &client{
		http:  &http.Client{Timeout: 30 * time.Second},
		base:  strings.TrimRight(st.URL, "/"),
		token: st.Token,
	}
	_ = st.AllowSend

	sc := bufio.NewScanner(in)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	enc := json.NewEncoder(out)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var req rpcReq
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			continue
		}
		if req.Method == "" {
			continue
		}
		// Notifications have no id (or null) and get no response.
		if req.ID == nil && !strings.HasPrefix(req.Method, "tools/") && req.Method != "initialize" && req.Method != "ping" {
			if req.Method == "notifications/initialized" || strings.HasPrefix(req.Method, "notifications/") {
				continue
			}
		}
		resp := c.dispatch(req)
		if resp == nil {
			continue
		}
		if err := enc.Encode(resp); err != nil {
			return err
		}
	}
	return sc.Err()
}

func loadBridgeState() (pfbridge.StateFile, error) {
	u := strings.TrimSpace(os.Getenv("PATHFINDER_MSP_URL"))
	if u == "" {
		u = strings.TrimSpace(os.Getenv("PATHFINDER_MCP_URL")) // legacy
	}
	if u != "" {
		tok := strings.TrimSpace(os.Getenv("PATHFINDER_MSP_TOKEN"))
		if tok == "" {
			tok = strings.TrimSpace(os.Getenv("PATHFINDER_MCP_TOKEN"))
		}
		return pfbridge.StateFile{URL: u, Token: tok, AllowSend: true}, nil
	}
	home := pfbridge.DefaultAppHome()
	if env := strings.TrimSpace(os.Getenv("PATHFINDER_HOME")); env != "" {
		home = env
	}
	return pfbridge.LoadState(home)
}

func (c *client) dispatch(req rpcReq) *rpcResp {
	switch req.Method {
	case "initialize":
		return &rpcResp{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "pathfinder-msp", "version": "1.0.0"},
		}}
	case "ping":
		return &rpcResp{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{}}
	case "tools/list":
		return &rpcResp{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{"tools": toolDefs()}}
	case "tools/call":
		return c.callTool(req)
	case "notifications/initialized":
		return nil
	default:
		if req.ID == nil {
			return nil
		}
		return &rpcResp{JSONRPC: "2.0", ID: req.ID, Error: &rpcErr{Code: -32601, Message: "method not found: " + req.Method}}
	}
}

func toolDefs() []map[string]any {
	return []map[string]any{
		{
			"name":        "pathfinder_list_sessions",
			"description": "List open PathfinderSSH terminal tabs (SSH sessions currently connected in the desktop app).",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}},
		},
		{
			"name":        "pathfinder_active_session",
			"description": "Return the PathfinderSSH tab that currently receives keyboard / macro traffic.",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}},
		},
		{
			"name":        "pathfinder_read_scrollback",
			"description": "Read terminal scrollback from a PathfinderSSH session for troubleshooting context.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"session_id": map[string]any{"type": "string", "description": "Session id from list_sessions; omit for active tab"},
					"max_chars":  map[string]any{"type": "integer", "description": "Max characters (default 24000)"},
				},
			},
		},
		{
			"name":        "pathfinder_send_command",
			"description": "Send text/commands into a live PathfinderSSH SSH tab. Requires confirm=true. Prefer read-only diagnosis first.",
			"inputSchema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"text":       map[string]any{"type": "string", "description": "Exact bytes/text to type (include \\n for Enter)"},
					"session_id": map[string]any{"type": "string", "description": "Optional session id; default active"},
					"confirm":    map[string]any{"type": "boolean", "description": "Must be true to send"},
				},
				"required": []string{"text", "confirm"},
			},
		},
		{
			"name":        "pathfinder_health",
			"description": "Check whether PathfinderSSH MSP is running and the Cursor bridge is reachable.",
			"inputSchema": map[string]any{"type": "object", "properties": map[string]any{}},
		},
	}
}

func (c *client) callTool(req rpcReq) *rpcResp {
	var p struct {
		Name      string         `json:"name"`
		Arguments map[string]any `json:"arguments"`
	}
	_ = json.Unmarshal(req.Params, &p)
	if p.Arguments == nil {
		p.Arguments = map[string]any{}
	}
	text, isErr := c.execTool(p.Name, p.Arguments)
	return &rpcResp{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{
		"content": []map[string]any{{"type": "text", "text": text}},
		"isError": isErr,
	}}
}

func (c *client) execTool(name string, args map[string]any) (string, bool) {
	// Reload discovery each call so Cursor works if Pathfinder started later.
	if st, err := loadBridgeState(); err == nil && strings.TrimSpace(st.URL) != "" {
		c.base = strings.TrimRight(st.URL, "/")
		c.token = st.Token
	}
	if c.base == "" {
		home := pfbridge.DefaultAppHome()
		return fmt.Sprintf("PathfinderSSH MSP Cursor bridge is not running.\nStart PathfinderSSH MSP (Settings → Tools → Cursor IDE bridge), then retry.\nExpected state file: %s", filepath.Join(home, pfbridge.StateFileName)), true
	}
	switch name {
	case "pathfinder_health":
		body, code, err := c.get("/v1/health")
		if err != nil {
			return "health error: " + err.Error(), true
		}
		return fmt.Sprintf("HTTP %d\n%s", code, body), code >= 300
	case "pathfinder_list_sessions":
		body, code, err := c.get("/v1/sessions")
		if err != nil {
			return err.Error(), true
		}
		return body, code >= 300
	case "pathfinder_active_session":
		body, code, err := c.get("/v1/sessions/active")
		if err != nil {
			return err.Error(), true
		}
		return body, code >= 300
	case "pathfinder_read_scrollback":
		id, _ := args["session_id"].(string)
		id = strings.TrimSpace(id)
		if id == "" {
			active, _, err := c.get("/v1/sessions/active")
			if err != nil {
				return err.Error(), true
			}
			var wrap struct {
				Session *struct {
					ID string `json:"id"`
				} `json:"session"`
			}
			_ = json.Unmarshal([]byte(active), &wrap)
			if wrap.Session == nil || wrap.Session.ID == "" {
				return "no active PathfinderSSH session", true
			}
			id = wrap.Session.ID
		}
		max := 24000
		switch v := args["max_chars"].(type) {
		case float64:
			max = int(v)
		case int:
			max = v
		}
		path := fmt.Sprintf("/v1/sessions/%s/scrollback?max=%d", id, max)
		body, code, err := c.get(path)
		if err != nil {
			return err.Error(), true
		}
		return body, code >= 300
	case "pathfinder_send_command":
		text, _ := args["text"].(string)
		id, _ := args["session_id"].(string)
		confirm, _ := args["confirm"].(bool)
		payload, _ := json.Marshal(map[string]any{
			"text":       text,
			"session_id": id,
			"confirm":    confirm,
		})
		body, code, err := c.post("/v1/send", payload)
		if err != nil {
			return err.Error(), true
		}
		return body, code >= 300
	default:
		return "unknown tool: " + name, true
	}
}

func (c *client) get(path string) (string, int, error) {
	req, err := http.NewRequest(http.MethodGet, c.base+path, nil)
	if err != nil {
		return "", 0, err
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	res, err := c.http.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("PathfinderSSH not reachable at %s (%v). Is the app open?", c.base, err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	return string(b), res.StatusCode, nil
}

func (c *client) post(path string, payload []byte) (string, int, error) {
	req, err := http.NewRequest(http.MethodPost, c.base+path, bytes.NewReader(payload))
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	res, err := c.http.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("PathfinderSSH not reachable at %s (%v). Is the app open?", c.base, err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	return string(b), res.StatusCode, nil
}
