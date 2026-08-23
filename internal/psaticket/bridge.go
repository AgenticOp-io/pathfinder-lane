package psaticket

import (
	"context"
	"fmt"
	"strings"
)

// TicketInfo is resolved PSA ticket metadata for ops desk binding.
type TicketInfo struct {
	ID           string
	Number       string
	Title        string
	CustomerName string
	CompanyID    string
}

// DocumentRequest is engineer documentation for a PSA ticket.
type DocumentRequest struct {
	TicketID   string
	Summary    string
	FileName   string
	FileBytes  []byte
}

// Bridge posts notes and attachments to a PSA ticket system.
type Bridge interface {
	Name() string
	LookupTicket(ctx context.Context, raw string) (TicketInfo, error)
	PostDocument(ctx context.Context, req DocumentRequest) error
}

// PostDocument validates and posts documentation.
func PostDocument(ctx context.Context, b Bridge, req DocumentRequest) error {
	if b == nil {
		return fmt.Errorf("psa bridge not configured")
	}
	if strings.TrimSpace(req.TicketID) == "" {
		return fmt.Errorf("ticket id required")
	}
	if strings.TrimSpace(req.Summary) == "" {
		return fmt.Errorf("work summary is empty")
	}
	return b.PostDocument(ctx, req)
}
