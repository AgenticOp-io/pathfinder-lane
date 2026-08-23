// Button bar editor — create, edit, and delete bottom-bar macros without YAML.
package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/scottpeterman/pathfinderssh/internal/buttons"
)

// ButtonEditorOptions wires the button-bar editor to disk and the host.
type ButtonEditorOptions struct {
	Path string
	File buttons.File
	// ScriptNames populates the "Run script" picker (from scripts.yaml).
	ScriptNames []string
	// OnSaved is called after a successful write so the host can refresh the bar.
	OnSaved func(buttons.File)
	// OnManageScripts opens the script editor (optional).
	OnManageScripts func()
}

// ShowButtonEditor opens a visual editor for buttons.yaml.
func ShowButtonEditor(w fyne.Window, opts ButtonEditorOptions) {
	if w == nil {
		return
	}
	ed := newButtonEditor(w, opts)
	ed.d.Resize(fyne.NewSize(720, 480))
	ed.d.Show()
}

type buttonEditor struct {
	w    fyne.Window
	opts ButtonEditorOptions
	file buttons.File

	list   *widget.List
	sel    int
	label  *widget.Entry
	action *widget.RadioGroup // "Send text" | "Run script"
	send   *widget.Entry
	script *widget.Select
	scope  *widget.Select
	status *widget.Label
	d      dialog.Dialog
}

func newButtonEditor(w fyne.Window, opts ButtonEditorOptions) *buttonEditor {
	ed := &buttonEditor{
		w:    w,
		opts: opts,
		file: opts.File,
		sel:  -1,
	}
	if ed.file.Version == 0 {
		ed.file.Version = 1
	}

	ed.label = widget.NewEntry()
	ed.label.SetPlaceHolder("Button label")
	ed.label.OnChanged = func(string) { ed.commitForm() }

	ed.send = widget.NewMultiLineEntry()
	ed.send.SetPlaceHolder("Text to send (use Enter for Return)")
	ed.send.SetMinRowsVisible(4)
	ed.send.OnChanged = func(string) { ed.commitForm() }

	scriptOpts := append([]string{""}, opts.ScriptNames...)
	ed.script = widget.NewSelect(scriptOpts, func(string) { ed.commitForm() })
	ed.script.PlaceHolder = "(pick a script)"

	ed.scope = widget.NewSelect([]string{"Active tab", "All tabs", "Customer"}, func(string) { ed.commitForm() })
	ed.scope.SetSelected("Active tab")

	ed.action = widget.NewRadioGroup([]string{"Send text", "Run script"}, func(string) {
		ed.updateActionVisibility()
		ed.commitForm()
	})
	ed.action.SetSelected("Send text")
	ed.action.Required = true
	ed.action.Horizontal = true

	ed.status = widget.NewLabel("")
	ed.status.Importance = widget.LowImportance

	ed.list = widget.NewList(
		func() int { return len(ed.file.Buttons) },
		func() fyne.CanvasObject { return widget.NewLabel("button") },
		func(id widget.ListItemID, o fyne.CanvasObject) {
			i := int(id)
			if i < 0 || i >= len(ed.file.Buttons) {
				return
			}
			b := ed.file.Buttons[i]
			text := strings.TrimSpace(b.Label)
			if text == "" {
				text = "(unnamed)"
			}
			if b.Script != "" {
				text += "  → script:" + b.Script
			} else if s := strings.TrimSpace(b.Send); s != "" {
				one := strings.ReplaceAll(s, "\n", "↵")
				if len(one) > 36 {
					one = one[:36] + "…"
				}
				text += "  → " + one
			}
			o.(*widget.Label).SetText(text)
		},
	)
	ed.list.OnSelected = func(id widget.ListItemID) {
		ed.commitForm()
		ed.sel = int(id)
		ed.loadForm()
	}

	leftBtns := container.NewGridWithColumns(2,
		widget.NewButtonWithIcon("New", theme.ContentAddIcon(), ed.newButton),
		widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), ed.deleteButton),
		widget.NewButtonWithIcon("Up", theme.MoveUpIcon(), func() { ed.moveButton(-1) }),
		widget.NewButtonWithIcon("Down", theme.MoveDownIcon(), func() { ed.moveButton(1) }),
	)
	left := container.NewBorder(
		widget.NewLabel("Buttons"),
		leftBtns,
		nil, nil,
		ed.list,
	)

	form := container.NewVBox(
		widget.NewLabel("Selected button"),
		widget.NewForm(
			widget.NewFormItem("Label", ed.label),
			widget.NewFormItem("Send to", ed.scope),
		),
		widget.NewLabel("Action"),
		ed.action,
		widget.NewLabel("Send text"),
		ed.send,
		widget.NewLabel("Script"),
		ed.script,
	)

	var manageScripts fyne.CanvasObject
	if opts.OnManageScripts != nil {
		manageScripts = widget.NewButtonWithIcon("Manage scripts…", theme.DocumentIcon(), opts.OnManageScripts)
	} else {
		manageScripts = layout.NewSpacer()
	}

	pathLbl := widget.NewLabel("File: " + opts.Path)
	pathLbl.Wrapping = fyne.TextWrapBreak
	hint := widget.NewLabel("These chips appear on the bottom button bar when a session is connected. Save applies immediately — no restart.")
	hint.Wrapping = fyne.TextWrapWord

	split := container.NewHSplit(left, container.NewVScroll(form))
	split.SetOffset(0.38)

	save := widget.NewButtonWithIcon("Save", theme.DocumentSaveIcon(), ed.save)
	save.Importance = widget.HighImportance
	closeBtn := widget.NewButtonWithIcon("Close", theme.CancelIcon(), func() { ed.d.Hide() })

	body := container.NewBorder(
		container.NewVBox(pathLbl, hint),
		container.NewVBox(
			ed.status,
			container.NewHBox(manageScripts, layout.NewSpacer(), closeBtn, save),
		),
		nil, nil,
		split,
	)
	ed.d = dialog.NewCustomWithoutButtons("Button bar", body, w)

	if len(ed.file.Buttons) > 0 {
		ed.list.Select(0)
	} else {
		ed.setFormEnabled(false)
	}
	ed.updateActionVisibility()
	return ed
}

func (ed *buttonEditor) setFormEnabled(on bool) {
	if on {
		ed.label.Enable()
		ed.send.Enable()
		ed.script.Enable()
		ed.scope.Enable()
		ed.action.Enable()
	} else {
		ed.label.Disable()
		ed.send.Disable()
		ed.script.Disable()
		ed.scope.Disable()
		ed.action.Disable()
	}
}

func (ed *buttonEditor) updateActionVisibility() {
	if ed.action.Selected == "Run script" {
		ed.send.Disable()
		ed.script.Enable()
		return
	}
	ed.send.Enable()
	ed.script.Disable()
}

func (ed *buttonEditor) loadForm() {
	if ed.sel < 0 || ed.sel >= len(ed.file.Buttons) {
		ed.setFormEnabled(false)
		ed.label.SetText("")
		ed.send.SetText("")
		ed.script.SetSelected("")
		return
	}
	ed.setFormEnabled(true)
	b := ed.file.Buttons[ed.sel]
	ed.label.SetText(b.Label)
	ed.send.SetText(b.Send)
	if b.Script != "" {
		ed.action.SetSelected("Run script")
		ed.script.SetSelected(b.Script)
	} else {
		ed.action.SetSelected("Send text")
		ed.script.SetSelected("")
	}
	switch strings.ToLower(strings.TrimSpace(b.Scope)) {
	case "all":
		ed.scope.SetSelected("All tabs")
	case "customer":
		ed.scope.SetSelected("Customer")
	default:
		ed.scope.SetSelected("Active tab")
	}
	ed.updateActionVisibility()
}

func (ed *buttonEditor) commitForm() {
	if ed.sel < 0 || ed.sel >= len(ed.file.Buttons) {
		return
	}
	b := ed.file.Buttons[ed.sel]
	b.Label = strings.TrimSpace(ed.label.Text)
	switch ed.scope.Selected {
	case "All tabs":
		b.Scope = "all"
	case "Customer":
		b.Scope = "customer"
	default:
		b.Scope = "active"
	}
	if ed.action.Selected == "Run script" {
		b.Script = strings.TrimSpace(ed.script.Selected)
		b.Send = ""
	} else {
		b.Script = ""
		b.Send = ed.send.Text
		if b.Send != "" && !strings.HasSuffix(b.Send, "\n") && !strings.HasSuffix(b.Send, "\r") {
			// Keep as typed; operators often want a trailing newline for Enter.
		}
	}
	ed.file.Buttons[ed.sel] = b
	ed.list.Refresh()
}

func (ed *buttonEditor) newButton() {
	ed.commitForm()
	ed.file.Buttons = append(ed.file.Buttons, buttons.Button{
		Label: "New button",
		Send:  "\n",
		Scope: "active",
	})
	ed.list.Refresh()
	ed.list.Select(len(ed.file.Buttons) - 1)
	ed.status.SetText("Added button — set label and action, then Save.")
}

func (ed *buttonEditor) deleteButton() {
	if ed.sel < 0 || ed.sel >= len(ed.file.Buttons) {
		return
	}
	name := ed.file.Buttons[ed.sel].Label
	if name == "" {
		name = "this button"
	}
	dialog.ShowConfirm("Delete button", "Delete \""+name+"\"?", func(ok bool) {
		if !ok {
			return
		}
		i := ed.sel
		ed.file.Buttons = append(ed.file.Buttons[:i], ed.file.Buttons[i+1:]...)
		ed.sel = -1
		ed.list.UnselectAll()
		ed.list.Refresh()
		if len(ed.file.Buttons) == 0 {
			ed.setFormEnabled(false)
			ed.label.SetText("")
			ed.send.SetText("")
			ed.status.SetText("No buttons left — New to add one, then Save.")
			return
		}
		if i >= len(ed.file.Buttons) {
			i = len(ed.file.Buttons) - 1
		}
		ed.list.Select(i)
		ed.status.SetText("Deleted — Save to apply.")
	}, ed.w)
}

func (ed *buttonEditor) moveButton(delta int) {
	ed.commitForm()
	i := ed.sel
	j := i + delta
	if i < 0 || j < 0 || i >= len(ed.file.Buttons) || j >= len(ed.file.Buttons) {
		return
	}
	ed.file.Buttons[i], ed.file.Buttons[j] = ed.file.Buttons[j], ed.file.Buttons[i]
	ed.list.Refresh()
	ed.list.Select(j)
}

func (ed *buttonEditor) save() {
	ed.commitForm()
	clean := make([]buttons.Button, 0, len(ed.file.Buttons))
	for _, b := range ed.file.Buttons {
		b.Label = strings.TrimSpace(b.Label)
		b.Script = strings.TrimSpace(b.Script)
		if b.Label == "" && b.Send == "" && b.Script == "" {
			continue
		}
		if b.Label == "" {
			if b.Script != "" {
				b.Label = b.Script
			} else {
				b.Label = "Button"
			}
		}
		if b.Script != "" {
			b.Send = ""
		}
		if b.Scope == "" {
			b.Scope = "active"
		}
		clean = append(clean, b)
	}
	ed.file.Buttons = clean
	ed.file.Version = 1
	if err := buttons.Save(ed.opts.Path, ed.file); err != nil {
		dialog.ShowError(fmt.Errorf("save buttons: %w", err), ed.w)
		return
	}
	if loaded, err := buttons.Load(ed.opts.Path); err == nil {
		ed.file = loaded
	}
	ed.list.Refresh()
	if ed.sel >= 0 && ed.sel < len(ed.file.Buttons) {
		ed.loadForm()
	} else if len(ed.file.Buttons) > 0 {
		ed.list.Select(0)
	}
	ed.status.SetText(fmt.Sprintf("Saved %d button(s).", len(ed.file.Buttons)))
	if ed.opts.OnSaved != nil {
		ed.opts.OnSaved(ed.file)
	}
}
