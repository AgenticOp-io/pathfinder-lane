package incidentbridge

import (
	"context"
	"fmt"
	"strings"
)

// DocumentRequest is work documentation sent to an incident system.
type DocumentRequest struct {
	IncidentID string
	Summary    string
	FileName   string
	FileBytes  []byte
}

// Bridge posts engineer work notes to an on-call/incident platform.
type Bridge interface {
	Name() string
	Verify(ctx context.Context) error
	PostDocument(ctx context.Context, req DocumentRequest) error
}

// PostDocument sends summary text; attaches file when the provider supports it.
func PostDocument(ctx context.Context, b Bridge, req DocumentRequest) error {
	if b == nil {
		return fmt.Errorf("incident bridge not configured")
	}
	if strings.TrimSpace(req.IncidentID) == "" {
		return fmt.Errorf("incident id required")
	}
	if strings.TrimSpace(req.Summary) == "" {
		return fmt.Errorf("work summary is empty")
	}
	return b.PostDocument(ctx, req)
}
