package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// CursorConfigForm edits Cursor AI troubleshoot settings.
type CursorConfigForm struct {
	apiKey *widget.Entry
	addon  *widget.Check
}

// NewCursorConfigForm builds Cursor API key + troubleshoot addon controls.
func NewCursorConfigForm() *CursorConfigForm {
	f := &CursorConfigForm{
		apiKey: widget.NewEntry(),
		addon:  widget.NewCheck("Enable Troubleshoot addon (Cursor AI pane + Ops agent)", nil),
	}
	f.apiKey.SetPlaceHolder("crsr_… or leave blank to use CURSOR_API_KEY")
	f.apiKey.Password = true
	f.addon.SetChecked(true)
	return f
}

func (f *CursorConfigForm) load(v SettingsFields) {
	f.apiKey.SetText(v.CursorAPIKey)
	f.addon.SetChecked(v.TroubleshootAddon)
}

func (f *CursorConfigForm) apply(base SettingsFields) SettingsFields {
	base.CursorAPIKey = f.apiKey.Text
	base.TroubleshootAddon = f.addon.Checked
	return base
}

// Content returns the form body for embedding in setup wizards.
func (f *CursorConfigForm) Content() fyne.CanvasObject {
	return container.NewVBox(
		widget.NewLabelWithStyle("Cursor AI", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Gather scrollback, ask Cursor Cloud Agents, and send commands to SSH.\n"+
			"Prefer CURSOR_API_KEY in the environment; this field is an optional override.\n"+
			"Create a key at cursor.com/dashboard → API Keys."),
		widget.NewForm(
			widget.NewFormItem("Cursor API key", f.apiKey),
			widget.NewFormItem("", f.addon),
		),
	)
}
