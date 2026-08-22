// Script editor — create and edit ~/.pathfinderssh/scripts.yaml without YAML.
package ui

import (
	"fmt"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/scottpeterman/pathfinderssh/internal/scripts"
)

// ScriptEditorOptions is everything the editor needs from the host.
type ScriptEditorOptions struct {
	// Path is shown and used for Save (usually scripts.Path(GetAppHome())).
	Path string
	// File is the starting set of scripts.
	File scripts.File
	// OnSaved is called after a successful write so the host can refresh menus.
	OnSaved func(scripts.File)
}

// ShowScriptEditor opens the visual script writer over w.
func ShowScriptEditor(w fyne.Window, opts ScriptEditorOptions) {
	if w == nil {
		return
	}
	ed := newScriptEditor(w, opts)
	ed.d.Resize(fyne.NewSize(820, 560))
	ed.d.Show()
}

type scriptEditor struct {
	w    fyne.Window
	opts ScriptEditorOptions
	file scripts.File

	scriptList *widget.List
	scriptSel  int

	nameEntry  *widget.Entry
	scopeSel   *widget.Select
	stepList   *widget.List
	stepSel    int
	sendEntry   *widget.Entry
	delayEntry  *widget.Entry
	waitEntry   *widget.Entry
	timeoutEntry *widget.Entry
	waitRegex   *widget.Check
	addEnter    *widget.Check
	status      *widget.Label

	d dialog.Dialog
}

func newScriptEditor(w fyne.Window, opts ScriptEditorOptions) *scriptEditor {
	ed := &scriptEditor{
		w:         w,
		opts:      opts,
		file:      opts.File,
		scriptSel: -1,
		stepSel:   -1,
	}
	if len(ed.file.Scripts) == 0 {
		ed.file = scripts.Defaults()
	}

	ed.nameEntry = widget.NewEntry()
	ed.nameEntry.SetPlaceHolder("Script name")
	ed.nameEntry.OnChanged = func(string) { ed.commitScriptMeta() }

	ed.scopeSel = widget.NewSelect([]string{"Active tab", "All tabs"}, func(string) {
		ed.commitScriptMeta()
	})
	ed.scopeSel.SetSelected("Active tab")

	ed.sendEntry = widget.NewMultiLineEntry()
	ed.sendEntry.SetMinRowsVisible(3)
	ed.sendEntry.SetPlaceHolder("Text to send (e.g. show version) — leave blank for wait-only")
	ed.sendEntry.OnChanged = func(string) { ed.commitStepFields() }

	ed.waitEntry = widget.NewEntry()
	ed.waitEntry.SetPlaceHolder("#  or  Password:  or  regex if checked")
	ed.waitEntry.OnChanged = func(string) { ed.commitStepFields() }

	ed.waitRegex = widget.NewCheck("Treat wait as regex", func(bool) {
		ed.commitStepFields()
	})

	ed.timeoutEntry = widget.NewEntry()
	ed.timeoutEntry.SetPlaceHolder("30000")
	ed.timeoutEntry.OnChanged = func(string) { ed.commitStepFields() }

	ed.delayEntry = widget.NewEntry()
	ed.delayEntry.SetPlaceHolder("300")
	ed.delayEntry.OnChanged = func(string) { ed.commitStepFields() }

	ed.addEnter = widget.NewCheck("Press Enter after send text", func(bool) {
		ed.commitStepFields()
	})
	ed.addEnter.SetChecked(true)

	ed.status = widget.NewLabel("")
	ed.status.Wrapping = fyne.TextWrapWord

	ed.scriptList = widget.NewList(
		func() int { return len(ed.file.Scripts) },
		func() fyne.CanvasObject { return widget.NewLabel("script") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			if i < 0 || i >= len(ed.file.Scripts) {
				return
			}
			sc := ed.file.Scripts[i]
			o.(*widget.Label).SetText(fmt.Sprintf("%s  (%d)", sc.Name, len(sc.Steps)))
		},
	)
	ed.scriptList.OnSelected = func(id widget.ListItemID) {
		ed.flushCurrent()
		ed.scriptSel = int(id)
		ed.stepSel = -1
		ed.loadScriptIntoForm()
	}

	ed.stepList = widget.NewList(
		func() int {
			if sc, ok := ed.currentScript(); ok {
				return len(sc.Steps)
			}
			return 0
		},
		func() fyne.CanvasObject { return widget.NewLabel("step") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			sc, ok := ed.currentScript()
			if !ok || i < 0 || i >= len(sc.Steps) {
				return
			}
			o.(*widget.Label).SetText(ed.stepSummary(i, sc.Steps[i]))
		},
	)
	ed.stepList.OnSelected = func(id widget.ListItemID) {
		ed.commitStepFields()
		ed.stepSel = int(id)
		ed.loadStepIntoForm()
	}

	leftBtns := container.NewGridWithColumns(2,
		widget.NewButtonWithIcon("New", theme.ContentAddIcon(), ed.newScript),
		widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), ed.deleteScript),
		widget.NewButtonWithIcon("Duplicate", theme.ContentCopyIcon(), ed.duplicateScript),
		widget.NewButtonWithIcon("Template…", theme.DocumentIcon(), ed.addTemplate),
	)
	left := container.NewBorder(
		widget.NewLabel("Scripts"),
		leftBtns,
		nil, nil,
		ed.scriptList,
	)

	stepBtns := container.NewHBox(
		widget.NewButtonWithIcon("Add step", theme.ContentAddIcon(), ed.addStep),
		widget.NewButtonWithIcon("Remove", theme.ContentRemoveIcon(), ed.removeStep),
		widget.NewButtonWithIcon("Up", theme.MoveUpIcon(), func() { ed.moveStep(-1) }),
		widget.NewButtonWithIcon("Down", theme.MoveDownIcon(), func() { ed.moveStep(1) }),
	)

	stepEditor := container.NewVBox(
		widget.NewLabel("Selected step"),
		ed.sendEntry,
		ed.addEnter,
		form(
			row("Wait for", ed.waitEntry),
			row("Wait timeout (ms)", ed.timeoutEntry),
			row("Wait after (ms)", ed.delayEntry),
		),
		ed.waitRegex,
	)

	right := container.NewBorder(
		container.NewVBox(
			form(
				row("Name", ed.nameEntry),
				row("Send to", ed.scopeSel),
			),
			widget.NewSeparator(),
			widget.NewLabel("Steps — each line is typed into the SSH session"),
			stepBtns,
		),
		stepEditor,
		nil, nil,
		ed.stepList,
	)

	split := container.NewHSplit(left, right)
	split.SetOffset(0.32)

	pathLbl := widget.NewLabel("File: " + opts.Path)
	pathLbl.Wrapping = fyne.TextWrapBreak
	hint := widget.NewLabel("Tip: after a command, set Wait for to the prompt (# or >) so the next step runs when the device is ready — not after a fixed sleep.")
	hint.Wrapping = fyne.TextWrapWord

	body := container.NewBorder(
		container.NewVBox(pathLbl, hint),
		container.NewVBox(ed.status, ed.footer()),
		nil, nil,
		split,
	)

	ed.d = dialog.NewCustomWithoutButtons("Script editor", body, w)

	if len(ed.file.Scripts) > 0 {
		ed.scriptList.Select(0)
	} else {
		ed.setEditorEnabled(false)
	}
	return ed
}

func (ed *scriptEditor) footer() fyne.CanvasObject {
	save := widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), ed.save)
	save.Importance = widget.HighImportance
	closeBtn := widget.NewButtonWithIcon("Close", theme.CancelIcon(), func() {
		ed.d.Hide()
	})
	return container.NewHBox(layout.NewSpacer(), closeBtn, save)
}

func (ed *scriptEditor) currentScript() (scripts.Script, bool) {
	if ed.scriptSel < 0 || ed.scriptSel >= len(ed.file.Scripts) {
		return scripts.Script{}, false
	}
	return ed.file.Scripts[ed.scriptSel], true
}

func (ed *scriptEditor) setEditorEnabled(on bool) {
	if on {
		ed.nameEntry.Enable()
		ed.scopeSel.Enable()
		ed.sendEntry.Enable()
		ed.delayEntry.Enable()
		ed.waitEntry.Enable()
		ed.timeoutEntry.Enable()
		ed.addEnter.Enable()
		ed.waitRegex.Enable()
	} else {
		ed.nameEntry.Disable()
		ed.scopeSel.Disable()
		ed.sendEntry.Disable()
		ed.delayEntry.Disable()
		ed.waitEntry.Disable()
		ed.timeoutEntry.Disable()
		ed.addEnter.Disable()
		ed.waitRegex.Disable()
	}
}

func (ed *scriptEditor) stepSummary(i int, st scripts.Step) string {
	parts := make([]string, 0, 3)
	text := strings.TrimSpace(strings.ReplaceAll(st.Send, "\n", "↵"))
	if text != "" {
		if len(text) > 36 {
			text = text[:33] + "…"
		}
		parts = append(parts, "send "+text)
	}
	if w := strings.TrimSpace(st.WaitFor); w != "" {
		parts = append(parts, "wait "+w)
	} else if w := strings.TrimSpace(st.WaitRegex); w != "" {
		parts = append(parts, "wait /"+w+"/")
	}
	if st.DelayMs > 0 {
		parts = append(parts, fmt.Sprintf("+%dms", st.DelayMs))
	}
	if len(parts) == 0 {
		parts = append(parts, "(empty)")
	}
	return fmt.Sprintf("%d. %s", i+1, strings.Join(parts, " · "))
}

func (ed *scriptEditor) flushCurrent() {
	ed.commitScriptMeta()
	ed.commitStepFields()
}

func (ed *scriptEditor) commitScriptMeta() {
	if ed.scriptSel < 0 || ed.scriptSel >= len(ed.file.Scripts) {
		return
	}
	name := strings.TrimSpace(ed.nameEntry.Text)
	if name == "" {
		name = "Untitled"
	}
	scope := "active"
	if ed.scopeSel.Selected == "All tabs" {
		scope = "all"
	}
	ed.file.Scripts[ed.scriptSel].Name = name
	ed.file.Scripts[ed.scriptSel].Scope = scope
	ed.scriptList.Refresh()
}

func (ed *scriptEditor) commitStepFields() {
	if ed.scriptSel < 0 || ed.scriptSel >= len(ed.file.Scripts) {
		return
	}
	if ed.stepSel < 0 || ed.stepSel >= len(ed.file.Scripts[ed.scriptSel].Steps) {
		return
	}
	send := ed.sendEntry.Text
	send = strings.ReplaceAll(send, "\r\n", "\n")
	send = strings.ReplaceAll(send, "\r", "\n")
	if ed.addEnter.Checked {
		trim := strings.TrimRight(send, "\n")
		if trim != "" {
			send = trim + "\n"
		} else if send == "" {
			// Wait-only / delay-only step: do not force a lone Enter.
			send = ""
		}
	} else {
		send = strings.TrimRight(send, "\n")
	}
	delay := parseNonNegInt(ed.delayEntry.Text, 0)
	timeout := parseNonNegInt(ed.timeoutEntry.Text, 0)
	wait := strings.TrimSpace(ed.waitEntry.Text)
	st := scripts.Step{Send: send, DelayMs: delay, TimeoutMs: timeout}
	if wait != "" {
		if ed.waitRegex.Checked {
			st.WaitRegex = wait
		} else {
			st.WaitFor = wait
		}
	}
	ed.file.Scripts[ed.scriptSel].Steps[ed.stepSel] = st
	ed.stepList.Refresh()
	ed.scriptList.Refresh()
}

func parseNonNegInt(text string, def int) int {
	t := strings.TrimSpace(text)
	if t == "" {
		return def
	}
	n, err := strconv.Atoi(t)
	if err != nil || n < 0 {
		return def
	}
	return n
}

func (ed *scriptEditor) loadScriptIntoForm() {
	sc, ok := ed.currentScript()
	if !ok {
		ed.setEditorEnabled(false)
		ed.nameEntry.SetText("")
		ed.stepList.Refresh()
		return
	}
	ed.setEditorEnabled(true)
	ed.nameEntry.SetText(sc.Name)
	if strings.EqualFold(sc.Scope, "all") {
		ed.scopeSel.SetSelected("All tabs")
	} else {
		ed.scopeSel.SetSelected("Active tab")
	}
	ed.stepList.UnselectAll()
	ed.stepList.Refresh()
	if len(sc.Steps) > 0 {
		ed.stepList.Select(0)
	} else {
		ed.stepSel = -1
		ed.sendEntry.SetText("")
		ed.waitEntry.SetText("")
		ed.delayEntry.SetText("300")
		ed.timeoutEntry.SetText("30000")
		ed.waitRegex.SetChecked(false)
		ed.addEnter.SetChecked(true)
	}
}

func (ed *scriptEditor) loadStepIntoForm() {
	sc, ok := ed.currentScript()
	if !ok || ed.stepSel < 0 || ed.stepSel >= len(sc.Steps) {
		ed.sendEntry.SetText("")
		ed.waitEntry.SetText("")
		ed.delayEntry.SetText("300")
		ed.timeoutEntry.SetText("30000")
		ed.waitRegex.SetChecked(false)
		return
	}
	st := sc.Steps[ed.stepSel]
	send := st.Send
	endsEnter := strings.HasSuffix(send, "\n")
	ed.addEnter.SetChecked(endsEnter && strings.TrimSpace(send) != "")
	if endsEnter {
		send = strings.TrimRight(send, "\n")
	}
	ed.sendEntry.SetText(send)
	if st.WaitRegex != "" {
		ed.waitEntry.SetText(st.WaitRegex)
		ed.waitRegex.SetChecked(true)
	} else {
		ed.waitEntry.SetText(st.WaitFor)
		ed.waitRegex.SetChecked(false)
	}
	if st.DelayMs > 0 {
		ed.delayEntry.SetText(strconv.Itoa(st.DelayMs))
	} else {
		ed.delayEntry.SetText("0")
	}
	if st.TimeoutMs > 0 {
		ed.timeoutEntry.SetText(strconv.Itoa(st.TimeoutMs))
	} else {
		ed.timeoutEntry.SetText(strconv.Itoa(scripts.DefaultWaitTimeoutMs))
	}
}

func (ed *scriptEditor) newScript() {
	ed.flushCurrent()
	ed.file.Scripts = append(ed.file.Scripts, scripts.Script{
		Name:  "New script",
		Scope: "active",
		Steps: []scripts.Step{{Send: "show version\n", WaitFor: "#", TimeoutMs: 15000, DelayMs: 100}},
	})
	ed.scriptList.Refresh()
	ed.scriptList.Select(len(ed.file.Scripts) - 1)
	ed.status.SetText("New script added — rename it and edit steps, then Save.")
}

func (ed *scriptEditor) deleteScript() {
	if ed.scriptSel < 0 || ed.scriptSel >= len(ed.file.Scripts) {
		return
	}
	name := ed.file.Scripts[ed.scriptSel].Name
	dialog.ShowConfirm("Delete script", "Delete “"+name+"”?", func(ok bool) {
		if !ok {
			return
		}
		i := ed.scriptSel
		ed.file.Scripts = append(ed.file.Scripts[:i], ed.file.Scripts[i+1:]...)
		ed.scriptSel = -1
		ed.stepSel = -1
		ed.scriptList.UnselectAll()
		ed.scriptList.Refresh()
		ed.stepList.Refresh()
		if len(ed.file.Scripts) > 0 {
			ed.scriptList.Select(0)
		} else {
			ed.setEditorEnabled(false)
			ed.nameEntry.SetText("")
			ed.sendEntry.SetText("")
		}
		ed.status.SetText("Deleted “" + name + "”. Save to write the file.")
	}, ed.w)
}

func (ed *scriptEditor) duplicateScript() {
	sc, ok := ed.currentScript()
	if !ok {
		return
	}
	ed.flushCurrent()
	sc, _ = ed.currentScript()
	copySteps := append([]scripts.Step(nil), sc.Steps...)
	ed.file.Scripts = append(ed.file.Scripts, scripts.Script{
		Name:  sc.Name + " copy",
		Scope: sc.Scope,
		Steps: copySteps,
	})
	ed.scriptList.Refresh()
	ed.scriptList.Select(len(ed.file.Scripts) - 1)
	ed.status.SetText("Duplicated — rename if you like, then Save.")
}

func (ed *scriptEditor) addTemplate() {
	templates := scripts.Defaults().Scripts
	names := make([]string, len(templates))
	for i, t := range templates {
		names[i] = t.Name
	}
	sel := widget.NewSelect(names, nil)
	sel.SetSelectedIndex(0)
	dialog.ShowForm("Add template", "Add", "Cancel", []*widget.FormItem{
		{Text: "Template", Widget: sel},
	}, func(ok bool) {
		if !ok {
			return
		}
		idx := sel.SelectedIndex()
		if idx < 0 || idx >= len(templates) {
			return
		}
		ed.flushCurrent()
		t := templates[idx]
		steps := append([]scripts.Step(nil), t.Steps...)
		ed.file.Scripts = append(ed.file.Scripts, scripts.Script{
			Name:  t.Name,
			Scope: t.Scope,
			Steps: steps,
		})
		ed.scriptList.Refresh()
		ed.scriptList.Select(len(ed.file.Scripts) - 1)
		ed.status.SetText("Added template “" + t.Name + "”. Save to keep it.")
	}, ed.w)
}

func (ed *scriptEditor) addStep() {
	if ed.scriptSel < 0 || ed.scriptSel >= len(ed.file.Scripts) {
		ed.status.SetText("Select or create a script first.")
		return
	}
	ed.flushCurrent()
	ed.file.Scripts[ed.scriptSel].Steps = append(ed.file.Scripts[ed.scriptSel].Steps, scripts.Step{
		Send:      "",
		WaitFor:   "#",
		TimeoutMs: scripts.DefaultWaitTimeoutMs,
		DelayMs:   0,
	})
	ed.stepList.Refresh()
	ed.scriptList.Refresh()
	ed.stepList.Select(len(ed.file.Scripts[ed.scriptSel].Steps) - 1)
	ed.status.SetText("Step added — set Send and/or Wait for (prompt marker), then Save.")
}

func (ed *scriptEditor) removeStep() {
	if ed.scriptSel < 0 || ed.stepSel < 0 {
		return
	}
	steps := ed.file.Scripts[ed.scriptSel].Steps
	if ed.stepSel >= len(steps) {
		return
	}
	ed.file.Scripts[ed.scriptSel].Steps = append(steps[:ed.stepSel], steps[ed.stepSel+1:]...)
	ed.stepSel = -1
	ed.stepList.UnselectAll()
	ed.stepList.Refresh()
	ed.scriptList.Refresh()
	ed.sendEntry.SetText("")
	if len(ed.file.Scripts[ed.scriptSel].Steps) > 0 {
		ed.stepList.Select(0)
	}
}

func (ed *scriptEditor) moveStep(delta int) {
	if ed.scriptSel < 0 || ed.stepSel < 0 {
		return
	}
	ed.commitStepFields()
	steps := ed.file.Scripts[ed.scriptSel].Steps
	j := ed.stepSel + delta
	if j < 0 || j >= len(steps) {
		return
	}
	steps[ed.stepSel], steps[j] = steps[j], steps[ed.stepSel]
	ed.file.Scripts[ed.scriptSel].Steps = steps
	ed.stepSel = j
	ed.stepList.Refresh()
	ed.stepList.Select(j)
}

func (ed *scriptEditor) save() {
	ed.flushCurrent()
	if ed.opts.Path == "" {
		ed.status.SetText("No scripts path configured.")
		return
	}
	clean := make([]scripts.Script, 0, len(ed.file.Scripts))
	for _, sc := range ed.file.Scripts {
		name := strings.TrimSpace(sc.Name)
		if name == "" {
			if len(sc.Steps) == 0 {
				continue
			}
			sc.Name = "Untitled"
		}
		clean = append(clean, sc)
	}
	ed.file.Scripts = clean
	ed.file.Version = 1
	if err := scripts.Save(ed.opts.Path, ed.file); err != nil {
		dialog.ShowError(err, ed.w)
		return
	}
	if loaded, err := scripts.Load(ed.opts.Path); err == nil {
		ed.file = loaded
	}
	ed.scriptList.Refresh()
	ed.stepList.Refresh()
	if ed.opts.OnSaved != nil {
		ed.opts.OnSaved(ed.file)
	}
	ed.status.SetText("Saved " + ed.opts.Path)
	dialog.ShowInformation("Scripts", "Saved "+strconv.Itoa(len(ed.file.Scripts))+" script(s).\nThey appear under the Scripts toolbar button.", ed.w)
}
