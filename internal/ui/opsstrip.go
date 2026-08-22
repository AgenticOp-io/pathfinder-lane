// Connect bar + armable chat strip for MSP / SecureCRT-style multi-send.
package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

// OpsStripOptions wire quick-connect and send-chat into the host.
type OpsStripOptions struct {
	// OnConnect dials host[:port] with optional username (user@host).
	OnConnect func(host, user string, port int)
	// OnSendChat delivers the line; customer is set when mode is ChatSendCustomer.
	OnSendChat func(text string, mode ChatSendMode, customer string)
	Customers  []string
}

// ChatSendMode selects fan-out for the chat box.
type ChatSendMode int

const (
	ChatSendActive ChatSendMode = iota
	ChatSendAll
	ChatSendCustomer
)

// NewOpsStrip builds a compact connect entry plus an armable send line.
// Prefer NewBottomChrome for the main window; this remains for tests / embeds.
func NewOpsStrip(opts OpsStripOptions) fyne.CanvasObject {
	return container.NewVBox(
		newConnectPane(opts.OnConnect),
		newSendPane(opts.Customers, opts.OnSendChat),
	)
}

// ParseQuickConnect accepts host, host:port, user@host, user@host:port.
func ParseQuickConnect(raw string) (host, user string, port int) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", 0
	}
	if at := strings.LastIndex(raw, "@"); at >= 0 {
		user = strings.TrimSpace(raw[:at])
		raw = strings.TrimSpace(raw[at+1:])
	}
	host = raw
	if i := strings.LastIndex(raw, ":"); i > 0 {
		suf := raw[i+1:]
		if isAllDigits(suf) {
			host = raw[:i]
			port = atoiPort(suf)
		}
	}
	return host, user, port
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

func atoiPort(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		n = n*10 + int(s[i]-'0')
		if n > 65535 {
			return 0
		}
	}
	return n
}
