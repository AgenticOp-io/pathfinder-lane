package ui

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/scottpeterman/pathfinderssh/internal/scripts"
)

// TroubleshootSession is one open terminal the addon can inspect.
type TroubleshootSession struct {
	Title    string
	Customer string
	Folder   string
	Target   string
	Active   bool
}

// TroubleshootHooks wire the modal into the host (sessions, scripts, Cursor).
type TroubleshootHooks struct {
	// Enabled must be true; otherwise the modal refuses to open.
	Enabled bool
	APIKey  string

	ListSessions func() []TroubleshootSession
	// GatherScrollback returns text for the active session, or all when all is true.
	GatherScrollback func(all bool) (string, error)
	// ScriptNames lists YAML script names for the Run script action.
	ScriptNames func() []string
	// RunScript runs a named YAML script into the active terminal.
	RunScript func(name string) error
	// LaunchCursor starts a Cloud Agent with the assembled pack + operator notes.
	LaunchCursor func(prompt, repo, ref, name string) (summary string, err error)
	// CheckCursor verifies API key / account.
	CheckCursor func() (summary string, err error)
}

// ShowTroubleshootAgent opens the MSP Troubleshoot addon modal.
func ShowTroubleshootAgent(w fyne.Window, hooks TroubleshootHooks) {
	if w == nil {
		return
	}
	if !hooks.Enabled {
		dialog.ShowInformation("Troubleshoot addon",
			"Enable it in Settings → Ops → Troubleshoot addon, then reopen this dialog.", w)
		return
	}

	log := widget.NewMultiLineEntry()
	log.SetPlaceHolder("Activity log…")
	log.Wrapping = fyne.TextWrapWord
	appendLog := func(line string) {
		ts := time.Now().Format("15:04:05")
		cur := log.Text
		if cur != "" && !strings.HasSuffix(cur, "\n") {
			cur += "\n"
		}
		log.SetText(cur + ts + "  " + line + "\n")
		log.CursorRow = strings.Count(log.Text, "\n")
	}

	ctxLabel := widget.NewLabel("No session context yet — Refresh.")
	ctxLabel.Wrapping = fyne.TextWrapWord

	notes := widget.NewMultiLineEntry()
	notes.SetPlaceHolder("What is broken? Ticket #, symptoms, recent change…")
	notes.SetMinRowsVisible(3)

	pack := widget.NewMultiLineEntry()
	pack.SetPlaceHolder("Evidence pack (scrollbacks, gather output) appears here")
	pack.SetMinRowsVisible(8)
	pack.Wrapping = fyne.TextWrapWord

	scriptNames := []string{}
	if hooks.ScriptNames != nil {
		scriptNames = hooks.ScriptNames()
	}
	scriptSel := widget.NewSelect(scriptNames, nil)
	if len(scriptNames) > 0 {
		scriptSel.SetSelected(scriptNames[0])
	}

	suggestScripts := func(list []TroubleshootSession) {
		if hooks.ScriptNames == nil {
			return
		}
		base := hooks.ScriptNames()
		var hints []string
		hints = append(hints, notes.Text)
		for _, s := range list {
			hints = append(hints, s.Customer, s.Folder, s.Title, s.Target)
			if s.Active {
				hints = append(hints, s.Customer, s.Folder, s.Title)
			}
		}
		ranked := scripts.RankNames(base, hints...)
		sel := scriptSel.Selected
		scriptSel.Options = ranked
		if len(ranked) == 0 {
			scriptSel.ClearSelected()
			return
		}
		keep := false
		for _, n := range ranked {
			if n == sel {
				keep = true
				break
			}
		}
		if keep {
			scriptSel.SetSelected(sel)
		} else {
			scriptSel.SetSelected(ranked[0])
		}
		appendLog("Suggested script: " + ranked[0])
	}

	refreshCtx := func() {
		if hooks.ListSessions == nil {
			ctxLabel.SetText("Session list not wired.")
			return
		}
		list := hooks.ListSessions()
		if len(list) == 0 {
			ctxLabel.SetText("No open terminals. Connect to a device first.")
			suggestScripts(nil)
			return
		}
		var b strings.Builder
		fmt.Fprintf(&b, "%d open terminal(s):\n", len(list))
		for _, s := range list {
			mark := " "
			if s.Active {
				mark = "*"
			}
			cust := s.Customer
			if cust == "" {
				cust = "—"
			}
			fmt.Fprintf(&b, "%s %s  customer=%s  %s\n", mark, s.Title, cust, s.Target)
		}
		ctxLabel.SetText(b.String())
		appendLog(fmt.Sprintf("Refreshed context (%d sessions)", len(list)))
		suggestScripts(list)
	}

	repo := widget.NewEntry()
	repo.SetPlaceHolder("GitHub repo for Cursor agent (optional)")
	ref := widget.NewEntry()
	ref.SetText("main")
	ref.SetPlaceHolder("branch / ref")

	gatherActive := widget.NewButtonWithIcon("Gather active scrollback", theme.DownloadIcon(), func() {
		if hooks.GatherScrollback == nil {
			return
		}
		text, err := hooks.GatherScrollback(false)
		if err != nil {
			appendLog("Gather failed: " + err.Error())
			dialog.ShowError(err, w)
			return
		}
		pack.SetText(mergePack(pack.Text, "=== ACTIVE SCROLLBACK ===\n"+text))
		appendLog(fmt.Sprintf("Gathered active scrollback (%d bytes)", len(text)))
	})
	gatherAll := widget.NewButtonWithIcon("Gather all tabs", theme.ListIcon(), func() {
		if hooks.GatherScrollback == nil {
			return
		}
		text, err := hooks.GatherScrollback(true)
		if err != nil {
			appendLog("Gather-all failed: " + err.Error())
			dialog.ShowError(err, w)
			return
		}
		pack.SetText(mergePack(pack.Text, "=== ALL TABS SCROLLBACK ===\n"+text))
		appendLog(fmt.Sprintf("Gathered all tabs (%d bytes)", len(text)))
	})
	runScript := widget.NewButtonWithIcon("Run diagnostic script", theme.MediaPlayIcon(), func() {
		name := scriptSel.Selected
		if name == "" || hooks.RunScript == nil {
			dialog.ShowInformation("Troubleshoot", "Pick a YAML script first.", w)
			return
		}
		appendLog("Running script: " + name)
		if err := hooks.RunScript(name); err != nil {
			appendLog("Script error: " + err.Error())
			dialog.ShowError(err, w)
			return
		}
		appendLog("Script started: " + name)
	})
	checkCursor := widget.NewButtonWithIcon("Check Cursor account", theme.ConfirmIcon(), func() {
		if hooks.CheckCursor == nil {
			return
		}
		appendLog("Checking Cursor account…")
		go func() {
			sum, err := hooks.CheckCursor()
			fyne.Do(func() {
				if err != nil {
					appendLog("Cursor check failed: " + err.Error())
					dialog.ShowError(err, w)
					return
				}
				appendLog(sum)
			})
		}()
	})
	var askCursor *widget.Button
	askCursor = widget.NewButtonWithIcon("Ask Cursor agent", theme.MailSendIcon(), func() {
		if hooks.LaunchCursor == nil {
			return
		}
		prompt := buildTroubleshootPrompt(notes.Text, ctxLabel.Text, pack.Text)
		if strings.TrimSpace(prompt) == "" {
			dialog.ShowInformation("Troubleshoot", "Add notes and/or gather evidence first.", w)
			return
		}
		appendLog("Launching Cursor cloud agent…")
		askCursor.Disable()
		go func() {
			sum, err := hooks.LaunchCursor(prompt, repo.Text, ref.Text, "pathfinder-troubleshoot")
			fyne.Do(func() {
				askCursor.Enable()
				if err != nil {
					appendLog("Cursor launch failed: " + err.Error())
					dialog.ShowError(err, w)
					return
				}
				appendLog(sum)
				dialog.ShowInformation("Cursor agent", sum, w)
			})
		}()
	})
	copyPack := widget.NewButtonWithIcon("Copy pack", theme.ContentCopyIcon(), func() {
		fyne.CurrentApp().Clipboard().SetContent(pack.Text)
		appendLog("Evidence pack copied to clipboard")
	})
	clearPack := widget.NewButton("Clear pack", func() {
		pack.SetText("")
		appendLog("Evidence pack cleared")
	})

	refreshBtn := widget.NewButtonWithIcon("Refresh sessions", theme.ViewRefreshIcon(), refreshCtx)

	actions := container.NewVBox(
		widget.NewLabelWithStyle("1 · Context", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		refreshBtn,
		ctxLabel,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("2 · Gather & run", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewGridWithColumns(2, gatherActive, gatherAll),
		container.NewBorder(nil, nil, nil, runScript, scriptSel),
		widget.NewSeparator(),
		widget.NewLabelWithStyle("3 · Notes", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		notes,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("4 · Cursor cloud agent", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		repo, ref,
		container.NewGridWithColumns(2, checkCursor, askCursor),
	)

	evidence := container.NewBorder(
		container.NewHBox(
			widget.NewLabelWithStyle("Evidence pack", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			copyPack, clearPack,
		),
		nil, nil, nil, pack,
	)

	body := container.NewBorder(
		nil,
		container.NewVBox(widget.NewLabel("Log"), container.NewScroll(log)),
		nil, nil,
		container.NewHSplit(container.NewVScroll(actions), evidence),
	)

	d := dialog.NewCustom("Troubleshoot agent", "Close", body, w)
	d.Resize(fyne.NewSize(920, 640))
	d.Show()
	refreshCtx()
	appendLog("Troubleshoot addon ready")
}

func mergePack(prev, next string) string {
	prev = strings.TrimSpace(prev)
	next = strings.TrimSpace(next)
	if prev == "" {
		return next
	}
	if next == "" {
		return prev
	}
	return prev + "\n\n" + next
}

func buildTroubleshootPrompt(notes, context, pack string) string {
	var b strings.Builder
	b.WriteString("You are assisting an MSP engineer using PathfinderSSH.\n")
	b.WriteString("Diagnose the issue, propose safe next CLI steps, and call out risk.\n\n")
	if n := strings.TrimSpace(notes); n != "" {
		b.WriteString("## Operator notes\n")
		b.WriteString(n)
		b.WriteString("\n\n")
	}
	if c := strings.TrimSpace(context); c != "" {
		b.WriteString("## Open sessions\n")
		b.WriteString(c)
		b.WriteString("\n")
	}
	if p := strings.TrimSpace(pack); p != "" {
		// Cap pack so we do not blow agent prompt limits.
		const max = 48_000
		if len(p) > max {
			p = p[len(p)-max:]
			p = "(truncated)\n" + p
		}
		b.WriteString("## Evidence pack\n```\n")
		b.WriteString(p)
		b.WriteString("\n```\n")
	}
	return strings.TrimSpace(b.String())
}
