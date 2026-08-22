// Persistent SecureCRT-style button strip at the bottom of the shell.
package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/scottpeterman/pathfinderssh/internal/buttons"
)

// ButtonBarOptions wire the strip into the host.
type ButtonBarOptions struct {
	Buttons []buttons.Button
	// OnSend is called with the button and whether the person armed "all sessions".
	OnSend func(b buttons.Button, all bool)
	// OnEdit opens the yaml help / editor hint.
	OnEdit func()
}

// NewButtonBar builds a scrollable row of macro buttons plus an "All" arm toggle.
func NewButtonBar(opts ButtonBarOptions) fyne.CanvasObject {
	all := widget.NewCheck("All tabs", nil)

	row := container.NewHBox()
	for _, b := range opts.Buttons {
		b := b
		label := b.Label
		if label == "" {
			label = "•"
		}
		tip := "Send to active terminal"
		if strings.EqualFold(b.Scope, "all") {
			tip = "Always sends to all open terminals"
		}
		lb := TipButtonLabeled(label, theme.MailSendIcon(), func() {
			if opts.OnSend == nil {
				return
			}
			forceAll := all.Checked || strings.EqualFold(b.Scope, "all")
			opts.OnSend(b, forceAll)
		})
		lb.Importance = widget.LowImportance
		lb.SetToolTip(tip)
		row.Add(lb)
	}
	if opts.OnEdit != nil {
		row.Add(TipIconButtonLow("Edit buttons.yaml", theme.DocumentCreateIcon(), opts.OnEdit))
	}
	scroll := container.NewHScroll(row)
	scroll.SetMinSize(fyne.NewSize(200, 36))
	return container.NewBorder(nil, nil, all, nil, scroll)
}
