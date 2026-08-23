package autotask

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/psaticket"
)

// TicketBridge implements psaticket.Bridge for Datto Autotask.
type TicketBridge struct{ Client *Client }

func (b TicketBridge) Name() string { return "autotask" }

func (b TicketBridge) LookupTicket(ctx context.Context, raw string) (psaticket.TicketInfo, error) {
	if b.Client == nil {
		return psaticket.TicketInfo{}, fmt.Errorf("autotask client missing")
	}
	raw = strings.TrimSpace(raw)
	filter := fmt.Sprintf(`{"filter":[{"op":"eq","field":"id","value":%s}],"MaxRecords":1}`, raw)
	if !isDigits(raw) {
		filter = fmt.Sprintf(`{"filter":[{"op":"eq","field":"ticketNumber","value":"%s"}],"MaxRecords":1}`, raw)
	}
	var out struct {
		Items []struct {
			ID           int    `json:"id"`
			TicketNumber string `json:"ticketNumber"`
			Title        string `json:"title"`
			CompanyID    int    `json:"companyID"`
			CompanyName  string `json:"companyName"`
		} `json:"items"`
	}
	if err := b.Client.postJSON(ctx, "/Tickets/query", filter, &out); err != nil {
		return psaticket.TicketInfo{}, err
	}
	if len(out.Items) == 0 {
		return psaticket.TicketInfo{}, fmt.Errorf("Autotask ticket not found: %s", raw)
	}
	t := out.Items[0]
	return psaticket.TicketInfo{
		ID:           fmt.Sprintf("%d", t.ID),
		Number:       t.TicketNumber,
		Title:        t.Title,
		CustomerName: t.CompanyName,
		CompanyID:    fmt.Sprintf("%d", t.CompanyID),
	}, nil
}

func (b TicketBridge) PostDocument(ctx context.Context, req psaticket.DocumentRequest) error {
	if b.Client == nil {
		return fmt.Errorf("autotask client missing")
	}
	id := strings.TrimSpace(req.TicketID)
	body := map[string]interface{}{
		"ticketID": id,
		"title":    "Pathfinder engineer note",
		"description": req.Summary,
		"noteType": 1,
		"publish":  1,
	}
	data, _ := json.Marshal(body)
	var out struct{}
	if err := b.Client.postJSON(ctx, "/TicketNotes", string(data), &out); err != nil {
		return err
	}
	if len(req.FileBytes) > 0 && req.FileName != "" {
		if err := b.uploadAttachment(ctx, id, req.FileName, req.FileBytes); err != nil {
			return fmt.Errorf("note posted; attachment failed: %w", err)
		}
	}
	return nil
}

func (b TicketBridge) uploadAttachment(ctx context.Context, ticketID, filename string, data []byte) error {
	name := strings.TrimSpace(filename)
	if name == "" {
		name = "evidence.zip"
	}
	body := map[string]interface{}{
		"attachmentType": "FILE_ATTACHMENT",
		"fullPath":       name,
		"title":          "Pathfinder engineer evidence",
		"publish":        1,
		"data":           base64.StdEncoding.EncodeToString(data),
	}
	payload, _ := json.Marshal(body)
	path := fmt.Sprintf("/Tickets/%s/Attachments", ticketID)
	return b.Client.postJSON(ctx, path, string(payload), &struct{}{})
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(s) > 0
}
