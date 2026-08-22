// Connect bar + armable chat strip for MSP / SecureCRT-style multi-send.
package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
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
func NewOpsStrip(opts OpsStripOptions) fyne.CanvasObject {
	hostEntry := widget.NewEntry()
	hostEntry.SetPlaceHolder("user@host[:port] — Quick Connect")
	connectBtn := widget.NewButtonWithIcon("Connect", theme.LoginIcon(), func() {
		if opts.OnConnect == nil {
			return
		}
		h, u, p := ParseQuickConnect(hostEntry.Text)
		if h == "" {
			return
		}
		opts.OnConnect(h, u, p)
	})
	hostEntry.OnSubmitted = func(string) { connectBtn.OnTapped() }

	chat := widget.NewEntry()
	chat.SetPlaceHolder("Type and Send — arm All or a customer to fan out")

	modeOpts := []string{"Active", "All tabs"}
	for _, c := range opts.Customers {
		c = strings.TrimSpace(c)
		if c != "" {
			modeOpts = append(modeOpts, "Customer: "+c)
		}
	}
	mode := widget.NewSelect(modeOpts, nil)
	mode.SetSelected("Active")

	send := widget.NewButtonWithIcon("Send", theme.MailSendIcon(), func() {
		if opts.OnSendChat == nil {
			return
		}
		text := chat.Text
		if text == "" {
			return
		}
		if !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		m, customer := chatModeFromSelect(mode.Selected)
		opts.OnSendChat(text, m, customer)
		chat.SetText("")
	})
	chat.OnSubmitted = func(string) { send.OnTapped() }

	connectRow := container.NewBorder(nil, nil, nil, connectBtn, hostEntry)
	chatRow := container.NewBorder(nil, nil, mode, send, chat)
	return container.NewVBox(connectRow, chatRow)
}

func chatModeFromSelect(sel string) (ChatSendMode, string) {
	switch {
	case sel == "All tabs":
		return ChatSendAll, ""
	case strings.HasPrefix(sel, "Customer:"):
		return ChatSendCustomer, strings.TrimSpace(strings.TrimPrefix(sel, "Customer:"))
	default:
		return ChatSendActive, ""
	}
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
