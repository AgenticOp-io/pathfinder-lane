package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// macroChip is a text button that does NOT take keyboard focus.
// Fyne's widget.Button is Focusable, so tapping a macro stole focus from the
// SSH terminal (laggy settle/refocus, white paint glitches, dead keyboard).
type macroChip struct {
	widget.BaseWidget
	label   string
	onTap   func()
	hovered bool
}

func newMacroChip(label string, onTap func()) *macroChip {
	c := &macroChip{label: label, onTap: onTap}
	c.ExtendBaseWidget(c)
	return c
}

func (c *macroChip) setLabel(label string) {
	c.label = label
	c.Refresh()
}

func (c *macroChip) Tapped(*fyne.PointEvent) {
	if c.onTap != nil {
		c.onTap()
	}
}

func (c *macroChip) Cursor() desktop.Cursor { return desktop.PointerCursor }

func (c *macroChip) MouseIn(*desktop.MouseEvent) {
	c.hovered = true
	c.Refresh()
}

func (c *macroChip) MouseMoved(*desktop.MouseEvent) {}

func (c *macroChip) MouseOut() {
	c.hovered = false
	c.Refresh()
}

func (c *macroChip) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(color.Transparent)
	bg.CornerRadius = 4
	txt := canvas.NewText(c.label, theme.Color(theme.ColorNameForeground))
	txt.TextSize = theme.TextSize()
	r := &macroChipRenderer{c: c, bg: bg, txt: txt, objs: []fyne.CanvasObject{bg, txt}}
	return r
}

type macroChipRenderer struct {
	c    *macroChip
	bg   *canvas.Rectangle
	txt  *canvas.Text
	objs []fyne.CanvasObject
}

func (r *macroChipRenderer) Destroy() {}
func (r *macroChipRenderer) Objects() []fyne.CanvasObject { return r.objs }

func (r *macroChipRenderer) Layout(size fyne.Size) {
	pad := theme.Padding()
	r.bg.Resize(size)
	ts := r.txt.MinSize()
	r.txt.Move(fyne.NewPos((size.Width-ts.Width)/2, (size.Height-ts.Height)/2))
	r.txt.Resize(ts)
	_ = pad
}

func (r *macroChipRenderer) MinSize() fyne.Size {
	pad := theme.Padding()
	ts := r.txt.MinSize()
	// Tall enough to hit easily; matches former button row height.
	return fyne.NewSize(ts.Width+pad*4, fyne.Max(ts.Height+pad*2.5, 32))
}

func (r *macroChipRenderer) Refresh() {
	r.txt.Text = r.c.label
	r.txt.Color = theme.Color(theme.ColorNameForeground)
	r.txt.TextSize = theme.TextSize()
	if r.c.hovered {
		r.bg.FillColor = theme.Color(theme.ColorNameHover)
	} else {
		r.bg.FillColor = theme.Color(theme.ColorNameButton)
	}
	r.bg.StrokeColor = theme.Color(theme.ColorNameInputBorder)
	r.bg.StrokeWidth = 1
	r.bg.Refresh()
	r.txt.Refresh()
}
