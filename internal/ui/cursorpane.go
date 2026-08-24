package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// CursorPaneHooks wires the Cursor side pane to live SSH sessions.
type CursorPaneHooks struct {
	APIKey           string
	ActiveContext    func() string
	GatherScrollback func(all bool) (string, error)
	AskCursor        func(prompt string) (summary string, err error)
}

// CursorAskPane is the right-side Cursor AI panel bound to the active SSH tab.
type CursorAskPane struct {
	hooks    CursorPaneHooks
	root     fyne.CanvasObject
	ctx      *widget.Label
	ask      *widget.Entry
	feedback *widget.Label
	askBtn   *widget.Button
	refresh  func()
}

// NewCursorPane builds the side pane (same as NewCursorAskPane).
func NewCursorPane(w fyne.Window, hooks CursorPaneHooks) *CursorAskPane {
	return NewCursorAskPane(w, hooks)
}

// NewCursorAskStrip keeps the old name for callers; returns the pane content only.
func NewCursorAskStrip(w fyne.Window, hooks CursorPaneHooks) fyne.CanvasObject {
	return NewCursorAskPane(w, hooks).Content()
}

// NewCursorAskPane builds Ask + feedback bound to ActiveTerminal scrollback.
func NewCursorAskPane(w fyne.Window, hooks CursorPaneHooks) *CursorAskPane {
	p := &CursorAskPane{hooks: hooks}

	p.ctx = widget.NewLabel("")
	p.ctx.Wrapping = fyne.TextWrapWord

	p.feedback = widget.NewLabel("Ask a question about the active SSH session.\nScrollback is sent as evidence to Cursor Cloud Agents.")
	p.feedback.Wrapping = fyne.TextWrapWord
	p.feedback.Importance = widget.LowImportance

	p.ask = widget.NewMultiLineEntry()
	p.ask.SetPlaceHolder("Ask about this session…")
	p.ask.SetMinRowsVisible(3)

	p.refresh = func() {
		text := ""
		if p.hooks.ActiveContext != nil {
			text = strings.TrimSpace(p.hooks.ActiveContext())
		}
		if text == "" {
			p.ctx.SetText("Not bound to a terminal.\nOpen or select an SSH tab.")
			p.ctx.Importance = widget.DangerImportance
			p.ask.Disable()
			if p.askBtn != nil {
				p.askBtn.Disable()
			}
		} else {
			p.ctx.SetText("Session: " + text)
			p.ctx.Importance = widget.MediumImportance
			p.ask.Enable()
			if p.askBtn != nil {
				p.askBtn.Enable()
			}
		}
		p.ctx.Refresh()
	}

	setBusy := func(busy bool) {
		if busy {
			p.ask.Disable()
			if p.askBtn != nil {
				p.askBtn.Disable()
			}
			return
		}
		p.refresh()
	}

	runAsk := func() {
		q := strings.TrimSpace(p.ask.Text)
		if q == "" {
			return
		}
		p.refresh()
		ctxText := ""
		if p.hooks.ActiveContext != nil {
			ctxText = strings.TrimSpace(p.hooks.ActiveContext())
		}
		if ctxText == "" {
			p.setFeedback("Not bound to a terminal — select an SSH tab first.", true)
			return
		}
		if p.hooks.AskCursor == nil {
			p.setFeedback("Cursor API not configured — add a key in Settings → Tools.", true)
			return
		}
		evidence := ""
		if p.hooks.GatherScrollback != nil {
			text, err := p.hooks.GatherScrollback(false)
			if err != nil {
				p.setFeedback("Cannot read terminal scrollback: "+err.Error(), true)
				return
			}
			evidence = text
		}
		prompt := buildCursorPrompt(q, ctxText, evidence)
		p.setFeedback(fmt.Sprintf("Asking Cursor… (%d chars of scrollback)", len(evidence)), false)
		setBusy(true)
		go func() {
			sum, err := p.hooks.AskCursor(prompt)
			fyne.Do(func() {
				setBusy(false)
				if err != nil {
					p.setFeedback("Error: "+err.Error(), true)
					return
				}
				if strings.TrimSpace(sum) == "" {
					p.setFeedback("Agent started — check the Cursor Agents dashboard.", false)
				} else {
					p.setFeedback(sum, false)
				}
			})
		}()
	}

	p.askBtn = widget.NewButtonWithIcon("Ask Cursor", theme.MailSendIcon(), runAsk)
	p.askBtn.Importance = widget.HighImportance

	title := widget.NewLabelWithStyle("Cursor AI", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	hint := widget.NewLabel("Uses the active SSH tab’s scrollback as evidence.")
	hint.Importance = widget.LowImportance
	hint.Wrapping = fyne.TextWrapWord

	fbScroll := container.NewVScroll(p.feedback)
	fbScroll.SetMinSize(fyne.NewSize(180, 160))

	p.root = container.NewBorder(
		container.NewVBox(title, p.ctx, hint, widget.NewSeparator()),
		container.NewVBox(p.ask, p.askBtn),
		nil, nil,
		container.NewPadded(fbScroll),
	)
	p.refresh()
	return p
}

func (p *CursorAskPane) setFeedback(text string, danger bool) {
	if p == nil || p.feedback == nil {
		return
	}
	p.feedback.SetText(text)
	if danger {
		p.feedback.Importance = widget.DangerImportance
	} else {
		p.feedback.Importance = widget.MediumImportance
	}
	p.feedback.Refresh()
}

// Content is the side-pane object for Shell.SetRight.
func (p *CursorAskPane) Content() fyne.CanvasObject {
	if p == nil {
		return nil
	}
	return p.root
}

// Refresh updates the bound-session line (call on tab change / connect).
func (p *CursorAskPane) Refresh() {
	if p != nil && p.refresh != nil {
		p.refresh()
	}
}

func buildCursorPrompt(question, context, evidence string) string {
	var b strings.Builder
	b.WriteString("PathfinderSSH MSP — network troubleshooting.\n")
	b.WriteString("You are advising an MSP engineer. Do NOT modify repository code unless asked.\n")
	b.WriteString("Answer with diagnosis steps and exact CLI commands when useful.\n\n")
	if strings.TrimSpace(context) != "" {
		b.WriteString("SESSION CONTEXT:\n")
		b.WriteString(context)
		b.WriteString("\n\n")
	}
	if strings.TrimSpace(evidence) != "" {
		b.WriteString("TERMINAL EVIDENCE:\n")
		b.WriteString(evidence)
		b.WriteString("\n\n")
	} else {
		b.WriteString("TERMINAL EVIDENCE: (empty scrollback)\n\n")
	}
	b.WriteString("OPERATOR QUESTION:\n")
	b.WriteString(question)
	return b.String()
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
		return ""
	}
	return strings.Join(parts, " · ")
}
