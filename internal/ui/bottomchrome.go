// Compact bottom chrome: one macros row + one command row (no stacked strips).
package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/scottpeterman/pathfinderssh/internal/buttons"
)

// BottomChromeOptions wire macros, quick-connect, and send into one dock.
type BottomChromeOptions struct {
	Buttons    []buttons.Button
	OnSend     func(b buttons.Button, all bool)
	OnEdit     func()
	Customers  []string
	OnConnect  func(host, user string, port int)
	OnSendChat func(text string, mode ChatSendMode, customer string)
}

// NewBottomChrome builds a CRT-style bottom dock:
//
//	macros row (when configured)
//	connect + multi-send on a single shared command row
//
// so the three features no longer stack three full-width bars.
func NewBottomChrome(opts BottomChromeOptions) fyne.CanvasObject {
	cmd := newCommandRow(opts.Customers, opts.OnConnect, opts.OnSendChat)
	if len(opts.Buttons) == 0 && opts.OnEdit == nil {
		return cmd
	}
	macros := NewButtonBar(ButtonBarOptions{
		Buttons: opts.Buttons,
		OnSend:  opts.OnSend,
		OnEdit:  opts.OnEdit,
	})
	return container.NewVBox(macros, cmd)
}

func newConnectPane(onConnect func(host, user string, port int)) fyne.CanvasObject {
	hostEntry := widget.NewEntry()
	hostEntry.SetPlaceHolder("user@host[:port]")
	btn := widget.NewButtonWithIcon("Connect", theme.LoginIcon(), func() {
		if onConnect == nil {
			return
		}
		h, u, p := ParseQuickConnect(hostEntry.Text)
		if h == "" {
			return
		}
		onConnect(h, u, p)
	})
	hostEntry.OnSubmitted = func(string) { btn.OnTapped() }
	hint := widget.NewLabel("Connect")
	hint.Importance = widget.LowImportance
	return container.NewBorder(nil, nil, hint, btn, hostEntry)
}

func newSendPane(customers []string, onSend func(string, ChatSendMode, string)) fyne.CanvasObject {
	chat := widget.NewEntry()
	chat.SetPlaceHolder("Command to send…")

	modeOpts := []string{"Active tab", "All tabs"}
	for _, c := range customers {
		c = strings.TrimSpace(c)
		if c != "" {
			modeOpts = append(modeOpts, "Customer: "+c)
		}
	}
	mode := widget.NewSelect(modeOpts, nil)
	mode.SetSelected("Active tab")
	mode.PlaceHolder = "Target"

	send := widget.NewButtonWithIcon("Send", theme.MailSendIcon(), func() {
		if onSend == nil {
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
		onSend(text, m, customer)
		chat.SetText("")
	})
	chat.OnSubmitted = func(string) { send.OnTapped() }
	return container.NewBorder(nil, nil, mode, send, chat)
}

// newCommandRow puts quick-connect and multi-send side by side on one row.
func newCommandRow(customers []string, onConnect func(host, user string, port int), onSend func(string, ChatSendMode, string)) fyne.CanvasObject {
	hostEntry := widget.NewEntry()
	hostEntry.SetPlaceHolder("user@host[:port]")
	connectBtn := widget.NewButtonWithIcon("", theme.LoginIcon(), func() {
		if onConnect == nil {
			return
		}
		h, u, p := ParseQuickConnect(hostEntry.Text)
		if h == "" {
			return
		}
		onConnect(h, u, p)
	})
	hostEntry.OnSubmitted = func(string) { connectBtn.OnTapped() }

	chat := widget.NewEntry()
	chat.SetPlaceHolder("Send command…")
	modeOpts := []string{"Active", "All"}
	for _, c := range customers {
		c = strings.TrimSpace(c)
		if c != "" {
			modeOpts = append(modeOpts, "Cust: "+c)
		}
	}
	mode := widget.NewSelect(modeOpts, nil)
	mode.SetSelected("Active")
	mode.PlaceHolder = "To"
	sendBtn := widget.NewButtonWithIcon("Send", theme.MailSendIcon(), func() {
		if onSend == nil {
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
		onSend(text, m, customer)
		chat.SetText("")
	})
	chat.OnSubmitted = func(string) { sendBtn.OnTapped() }

	left := container.NewBorder(nil, nil, nil, connectBtn, hostEntry)
	right := container.NewBorder(nil, nil, mode, sendBtn, chat)
	split := container.NewHSplit(left, right)
	split.SetOffset(0.38)
	return split
}

// chatModeFromSelect maps Send-tab labels (and legacy OpsStrip labels).
func chatModeFromSelect(sel string) (ChatSendMode, string) {
	switch {
	case sel == "All tabs" || sel == "All":
		return ChatSendAll, ""
	case strings.HasPrefix(sel, "Customer:"):
		return ChatSendCustomer, strings.TrimSpace(strings.TrimPrefix(sel, "Customer:"))
	case strings.HasPrefix(sel, "Cust:"):
		return ChatSendCustomer, strings.TrimSpace(strings.TrimPrefix(sel, "Cust:"))
	default:
		return ChatSendActive, ""
	}
}
