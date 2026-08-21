// Icon / toolbar buttons with mouseover tooltips (Fyne has none built in).
package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	fynetooltip "github.com/dweymouth/fyne-tooltip"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"
)

// TipButton is an icon/text button that can show a mouseover tip.
type TipButton = ttwidget.Button

// WithTooltips wraps window content so hover tips from TipIconButton render.
func WithTooltips(content fyne.CanvasObject, c fyne.Canvas) fyne.CanvasObject {
	if content == nil || c == nil {
		return content
	}
	return fynetooltip.AddWindowToolTipLayer(content, c)
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

// TipIconButtonLow is TipIconButton with LowImportance chrome.
func TipIconButtonLow(tip string, icon fyne.Resource, tapped func()) *TipButton {
	b := TipIconButton(tip, icon, tapped)
	b.Importance = widget.LowImportance
	return b
}
