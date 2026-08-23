package connectwise

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/psaticket"
)

// TicketBridge implements psaticket.Bridge for ConnectWise Manage.
type TicketBridge struct{ Client *Client }

func (b TicketBridge) Name() string { return "connectwise" }

func (b TicketBridge) LookupTicket(ctx context.Context, raw string) (psaticket.TicketInfo, error) {
	if b.Client == nil {
		return psaticket.TicketInfo{}, fmt.Errorf("connectwise client missing")
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return psaticket.TicketInfo{}, fmt.Errorf("ticket id or number required")
	}
	cond := fmt.Sprintf("id=%s", raw)
	if !strings.Contains(raw, "-") && len(raw) < 12 {
		cond = fmt.Sprintf("ticketNumber=%s", raw)
	}
	path := fmt.Sprintf("/service/tickets?conditions=%s&pageSize=1", cond)
	var out []struct {
		ID           int    `json:"id"`
		Summary      string `json:"summary"`
		TicketNumber int    `json:"ticketNumber"`
		Company      struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"company"`
	}
	if err := b.Client.getJSON(ctx, path, &out); err != nil {
		return psaticket.TicketInfo{}, err
	}
	if len(out) == 0 {
		return psaticket.TicketInfo{}, fmt.Errorf("ConnectWise ticket not found: %s", raw)
	}
	t := out[0]
	return psaticket.TicketInfo{
		ID:           fmt.Sprintf("%d", t.ID),
		Number:       fmt.Sprintf("%d", t.TicketNumber),
		Title:        t.Summary,
		CustomerName: t.Company.Name,
		CompanyID:    fmt.Sprintf("%d", t.Company.ID),
	}, nil
}

func (b TicketBridge) PostDocument(ctx context.Context, req psaticket.DocumentRequest) error {
	if b.Client == nil {
		return fmt.Errorf("connectwise client missing")
	}
	id := strings.TrimSpace(req.TicketID)
	if err := b.postNote(ctx, id, req.Summary); err != nil {
		return err
	}
	if len(req.FileBytes) > 0 {
		name := strings.TrimSpace(req.FileName)
		if name == "" {
			name = "evidence.zip"
		}
		if err := b.uploadDocument(ctx, id, name, req.FileBytes); err != nil {
			return fmt.Errorf("note posted; attachment failed: %w", err)
		}
	}
	return nil
}

func (b TicketBridge) postNote(ctx context.Context, ticketID, text string) error {
	body := map[string]interface{}{
		"text":                  text,
		"detailDescriptionFlag": true,
	}
	path := fmt.Sprintf("/service/tickets/%s/notes", ticketID)
	return b.Client.postJSON(ctx, path, body)
}

func (b TicketBridge) uploadDocument(ctx context.Context, ticketID, filename string, data []byte) error {
	companyID, publicKey, privateKey, clientID, base, err := b.Client.credentials()
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("recordType", "Ticket")
	_ = w.WriteField("recordId", ticketID)
	_ = w.WriteField("title", filename)
	part, err := w.CreateFormFile("file", filename)
	if err != nil {
		return err
	}
	if _, err := part.Write(data); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	u := base + "/system/documents"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, &buf)
	if err != nil {
		return err
	}
	req.SetBasicAuth(companyID+"+"+publicKey, privateKey)
	if clientID != "" {
		req.Header.Set("clientId", clientID)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := b.Client.http().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("connectwise document %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	return nil
}
