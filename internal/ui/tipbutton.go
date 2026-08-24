// Icon / toolbar buttons with mouseover tooltips (Fyne has none built in).
package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"
)

// TipButton is an icon/text button that can show a mouseover tip.
type TipButton = ttwidget.Button

// WithTooltips used to wrap the window in fyne-tooltip's overlay layer.
// That layer plus any Fyne dialog (session form, Connecting, errors) was
// aborting Pathfinder on Windows ARM64 with no Go panic — Auvik sessions
// never even reached connect(). Tips still set on buttons are ignored until
// a safer hover path exists.
func WithTooltips(content fyne.CanvasObject, c fyne.Canvas) fyne.CanvasObject {
	_ = c
	return content
}

// TipIconButton is an icon-only button that shows tip on mouseover.
func TipIconButton(tip string, icon fyne.Resource, tapped func()) *TipButton {
	b := ttwidget.NewButtonWithIcon("", icon, tapped)
	if tip != "" {
		b.SetToolTip(tip)
	}
	return b
}

// TipButtonLabeled is a labelled toolbar button with the same text as its tip.
func TipButtonLabeled(label string, icon fyne.Resource, tapped func()) *TipButton {
	b := ttwidget.NewButtonWithIcon(label, icon, tapped)
	if label != "" {
		b.SetToolTip(label)
	}
	return b
}

// TipButtonText is a text-only button with an optional tooltip.
func TipButtonText(label, tip string, tapped func()) *TipButton {
	b := ttwidget.NewButton(label, tapped)
	if tip != "" {
		b.SetToolTip(tip)
	} else if label != "" {
		b.SetToolTip(label)
	}
	return b
}

// TipIconButtonLow is TipIconButton with LowImportance chrome.
func TipIconButtonLow(tip string, icon fyne.Resource, tapped func()) *TipButton {
	b := TipIconButton(tip, icon, tapped)
	b.Importance = widget.LowImportance
	return b
}
