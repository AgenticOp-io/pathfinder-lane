package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/scottpeterman/pathfinderssh/internal/mspsecurity"
)

// SecurityPolicyForm collects org security defaults for engineer workstations.
type SecurityPolicyForm struct {
	readOnly   *widget.Check
	breakGlass *widget.Check
	capture    *widget.Check
	start      *widget.Entry
	end        *widget.Entry
	content    fyne.CanvasObject
}

// NewSecurityPolicyForm builds security policy fields.
func NewSecurityPolicyForm(preset mspsecurity.Policy) *SecurityPolicyForm {
	f := &SecurityPolicyForm{}
	f.readOnly = widget.NewCheck("Read-only mode (block interactive sends outside change window)", nil)
	f.breakGlass = widget.NewCheck("Vault break-glass (ops desk may use any vault credential)", nil)
	f.capture = widget.NewCheck("Capture configs by default on new sessions", nil)
	f.start = widget.NewEntry()
	f.start.SetPlaceHolder("09:00 (optional change window start)")
	f.end = widget.NewEntry()
	f.end.SetPlaceHolder("17:00 (optional change window end)")

	f.readOnly.SetChecked(preset.ReadOnlyMode)
	f.breakGlass.SetChecked(preset.VaultBreakGlass)
	f.capture.SetChecked(preset.CaptureByDefault)
	f.start.SetText(preset.ChangeWindowStart)
	f.end.SetText(preset.ChangeWindowEnd)

	f.content = container.NewVBox(
		widget.NewLabel("These settings ship to engineer PCs. Engineers cannot change org policy from the standalone installer."),
		widget.NewForm(
			widget.NewFormItem("Read-only mode", f.readOnly),
			widget.NewFormItem("Change window start", f.start),
			widget.NewFormItem("Change window end", f.end),
			widget.NewFormItem("Vault break-glass", f.breakGlass),
			widget.NewFormItem("Capture by default", f.capture),
		),
	)
	return f
}

func (f *SecurityPolicyForm) Content() fyne.CanvasObject {
	if f == nil {
		return widget.NewLabel("")
	}
	return f.content
}

func (f *SecurityPolicyForm) Policy() mspsecurity.Policy {
	if f == nil {
		return mspsecurity.Policy{}
	}
	return mspsecurity.Policy{
		ReadOnlyMode:      f.readOnly.Checked,
		ChangeWindowStart: f.start.Text,
		ChangeWindowEnd:   f.end.Text,
		VaultBreakGlass:   f.breakGlass.Checked,
		CaptureByDefault:  f.capture.Checked,
	}
}

func (f *SecurityPolicyForm) Save() error {
	return mspsecurity.Save(f.Policy())
}

// ApplyPolicyToSettings merges a security policy into settings.
func ApplyPolicyToSettings(base Settings, p mspsecurity.Policy) Settings {
	base.ReadOnlyMode = p.ReadOnlyMode
	base.ChangeWindowStart = p.ChangeWindowStart
	base.ChangeWindowEnd = p.ChangeWindowEnd
	base.VaultBreakGlass = p.VaultBreakGlass
	base.CaptureByDefault = p.CaptureByDefault
	return base
}
