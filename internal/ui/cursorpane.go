package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// CursorPaneHooks wires the side-panel AI assistant to live SSH sessions.
type CursorPaneHooks struct {
	APIKey string
	// ActiveContext describes the focused terminal session.
	ActiveContext func() string
	GatherScrollback func(all bool) (string, error)
	SendToActive     func(command string) error
	RunScript        func(name string) error
	ScriptNames      func() []string
	AskCursor          func(prompt string) (summary string, err error)
}

// NewCursorPane builds the right-side Cursor troubleshooting panel.
func NewCursorPane(w fyne.Window, hooks CursorPaneHooks) fyne.CanvasObject {
	ctx := widget.NewLabel("No active SSH session")
	ctx.Wrapping = fyne.TextWrapWord

	chat := widget.NewMultiLineEntry()
	chat.SetPlaceHolder("Ask about the active session…")
	chat.Wrapping = fyne.TextWrapWord
	chat.SetMinRowsVisible(4)

	evidence := widget.NewMultiLineEntry()
	evidence.SetPlaceHolder("Session scrollback / evidence")
	evidence.Wrapping = fyne.TextWrapWord
	evidence.SetMinRowsVisible(6)

	log := widget.NewLabel("")
	log.Wrapping = fyne.TextWrapWord

	appendLog := func(msg string) {
		if strings.TrimSpace(msg) == "" {
			return
		}
		cur := strings.TrimSpace(log.Text)
		if cur != "" {
			cur += "\n"
		}
		log.SetText(cur + msg)
	}

	refreshCtx := func() {
		if hooks.ActiveContext != nil {
			ctx.SetText(hooks.ActiveContext())
		}
	}

	refreshCtxBtn := widget.NewButtonWithIcon("Refresh context", theme.ViewRefreshIcon(), func() {
		refreshCtx()
		appendLog("Context refreshed")
	})

	gatherBtn := widget.NewButtonWithIcon("Gather scrollback", theme.DocumentIcon(), func() {
		if hooks.GatherScrollback == nil {
			return
		}
		text, err := hooks.GatherScrollback(false)
		if err != nil {
			appendLog("Gather failed: " + err.Error())
			return
		}
		evidence.SetText(text)
		appendLog("Gathered active session scrollback")
	})

	sendBtn := widget.NewButtonWithIcon("Send to SSH", theme.MailSendIcon(), func() {
		cmd := strings.TrimSpace(chat.Text)
		if cmd == "" {
			return
		}
		if hooks.SendToActive == nil {
			appendLog("Send not available")
			return
		}
		if !strings.HasSuffix(cmd, "\n") {
			cmd += "\n"
		}
		if err := hooks.SendToActive(cmd); err != nil {
			appendLog("Send failed: " + err.Error())
			return
		}
		appendLog("Sent command to active session")
		chat.SetText("")
	})

	askBtn := widget.NewButtonWithIcon("Ask Cursor", theme.MailSendIcon(), func() {
		if hooks.AskCursor == nil {
			appendLog("Cursor API not configured")
			return
		}
		q := strings.TrimSpace(chat.Text)
		if q == "" {
			dialog.ShowInformation("Cursor", "Type a question first.", w)
			return
		}
		prompt := buildCursorPrompt(q, ctx.Text, evidence.Text)
		appendLog("Asking Cursor…")
		go func() {
			sum, err := hooks.AskCursor(prompt)
			fyne.Do(func() {
				if err != nil {
					appendLog("Cursor error: " + err.Error())
					return
				}
				appendLog("Cursor agent started")
				if sum != "" {
					evidence.SetText(mergeEvidence(evidence.Text, "=== CURSOR ===\n"+sum))
				}
			})
		}()
	})

	scriptNames := []string{}
	if hooks.ScriptNames != nil {
		scriptNames = hooks.ScriptNames()
	}
	scriptSel := widget.NewSelect(scriptNames, nil)
	if len(scriptNames) > 0 {
		scriptSel.SetSelected(scriptNames[0])
	}
	runScript := widget.NewButtonWithIcon("Run script", theme.MediaPlayIcon(), func() {
		if hooks.RunScript == nil || scriptSel.Selected == "" {
			return
		}
		if err := hooks.RunScript(scriptSel.Selected); err != nil {
			appendLog("Script failed: " + err.Error())
			return
		}
		appendLog("Ran script: " + scriptSel.Selected)
	})

	header := widget.NewLabelWithStyle("Cursor AI", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	sub := widget.NewLabel("Uses active SSH scrollback. Suggested commands can be sent back to the session.")
	sub.Wrapping = fyne.TextWrapWord

	tools := container.NewHBox(refreshCtxBtn, gatherBtn, runScript, scriptSel, sendBtn, askBtn)

	body := container.NewVBox(
		header,
		sub,
		widget.NewLabel("Active session"),
		ctx,
		tools,
		widget.NewLabel("Evidence"),
		evidence,
		widget.NewLabel("Question / command"),
		chat,
		widget.NewLabel("Activity"),
		log,
	)
	return container.NewVScroll(body)
}

func buildCursorPrompt(question, context, evidence string) string {
	var b strings.Builder
	b.WriteString("PathfinderSSH MSP — network troubleshooting.\n\n")
	if strings.TrimSpace(context) != "" {
		b.WriteString("SESSION CONTEXT:\n")
		b.WriteString(context)
		b.WriteString("\n\n")
	}
	if strings.TrimSpace(evidence) != "" {
		b.WriteString("TERMINAL EVIDENCE:\n")
		b.WriteString(evidence)
		b.WriteString("\n\n")
	}
	b.WriteString("OPERATOR QUESTION:\n")
	b.WriteString(question)
	return b.String()
}

func mergeEvidence(prev, block string) string {
	prev = strings.TrimSpace(prev)
	block = strings.TrimSpace(block)
	if prev == "" {
		return block
	}
	if block == "" {
		return prev
	}
	return prev + "\n\n" + block
}

// CursorPaneTitle is the splitter tab label when embedded elsewhere.
func CursorPaneTitle() string { return "Cursor AI" }

// FormatActiveContext builds a one-line session summary for the pane header.
func FormatActiveContext(title, customer, folder, target string, active bool) string {
	parts := []string{}
	if title != "" {
		parts = append(parts, title)
	}
	if target != "" {
		parts = append(parts, target)
	}
	if customer != "" {
		parts = append(parts, "customer: "+customer)
	}
	if folder != "" {
		parts = append(parts, folder)
	}
	if active {
		parts = append(parts, "(active)")
	}
	if len(parts) == 0 {
		return "No active SSH session"
	}
	return strings.Join(parts, " · ")
}
