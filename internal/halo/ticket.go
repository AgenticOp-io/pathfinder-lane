package halo

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/psaticket"
)

// TicketBridge implements psaticket.Bridge for Halo PSA.
type TicketBridge struct{ Client *Client }

func (b TicketBridge) Name() string { return "halo" }

func (b TicketBridge) LookupTicket(ctx context.Context, raw string) (psaticket.TicketInfo, error) {
	if b.Client == nil {
		return psaticket.TicketInfo{}, fmt.Errorf("halo client missing")
	}
	raw = strings.TrimSpace(raw)
	path := fmt.Sprintf("/api/Tickets/%s", raw)
	var out struct {
		ID       int    `json:"id"`
		Summary  string `json:"summary"`
		ClientID int    `json:"client_id"`
		Client   struct {
			Name string `json:"name"`
		} `json:"client"`
	}
	if err := b.Client.getJSON(ctx, path, &out); err != nil {
		return psaticket.TicketInfo{}, err
	}
	return psaticket.TicketInfo{
		ID:           fmt.Sprintf("%d", out.ID),
		Number:       fmt.Sprintf("%d", out.ID),
		Title:        out.Summary,
		CustomerName: out.Client.Name,
		CompanyID:    fmt.Sprintf("%d", out.ClientID),
	}, nil
}

func (b TicketBridge) PostDocument(ctx context.Context, req psaticket.DocumentRequest) error {
	if b.Client == nil {
		return fmt.Errorf("halo client missing")
	}
	id := strings.TrimSpace(req.TicketID)
	body := map[string]interface{}{
		"ticket_id": id,
		"outcome":   "Pathfinder engineer work",
		"note":      req.Summary,
		"hidden":    false,
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return b.Client.postJSON(ctx, fmt.Sprintf("/api/Tickets/%s/Actions", id), data, nil)
}
