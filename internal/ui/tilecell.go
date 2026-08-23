package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// tileCell wraps a docked terminal root in tiled layout with a focus ring and
// title strip so operators can see and pick the active pane.
type tileCell struct {
	shell   *Shell
	inst    *Instance
	border  *canvas.Rectangle
	header  *tileHeader
	content fyne.CanvasObject
	root    fyne.CanvasObject
	active  bool
}

func newTileCell(s *Shell, inst *Instance) *tileCell {
	tc := &tileCell{shell: s, inst: inst, content: inst.root}
	tc.border = canvas.NewRectangle(color.Transparent)
	tc.border.CornerRadius = 4
	tc.setActive(false)

	inner := container.NewStack(tc.border, inst.root)
	tc.header = newTileHeader(terminalTabTitle(inst, true), func() { s.activateTile(inst) })
	tc.root = container.NewBorder(tc.header, nil, nil, nil, inner)
	return tc
}

func (tc *tileCell) setActive(on bool) {
	if tc == nil {
		return
	}
	tc.active = on
	if on {
		tc.border.StrokeColor = theme.Color(theme.ColorNamePrimary)
		tc.border.StrokeWidth = 3
	} else {
		tc.border.StrokeColor = theme.Color(theme.ColorNameInputBorder)
		tc.border.StrokeWidth = 1
	}
	tc.border.FillColor = color.Transparent
	tc.border.Refresh()
	if tc.header != nil {
		tc.header.setActive(on)
	}
}

// tileHeader is a non-focusable title strip; tapping selects the pane.
type tileHeader struct {
	widget.BaseWidget
	title  string
	active bool
	onTap  func()
	hover  bool
}

func newTileHeader(title string, onTap func()) *tileHeader {
	h := &tileHeader{title: title, onTap: onTap}
	h.ExtendBaseWidget(h)
	return h
}

func (h *tileHeader) setActive(on bool) {
	h.active = on
	h.Refresh()
}

func (h *tileHeader) Tapped(*fyne.PointEvent) {
	if h.onTap != nil {
		h.onTap()
	}
}

func (h *tileHeader) Cursor() desktop.Cursor { return desktop.PointerCursor }

func (h *tileHeader) MouseIn(*desktop.MouseEvent) {
	h.hover = true
	h.Refresh()
}

func (h *tileHeader) MouseMoved(*desktop.MouseEvent) {}

func (h *tileHeader) MouseOut() {
	h.hover = false
	h.Refresh()
}

func (h *tileHeader) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(theme.Color(theme.ColorNameButton))
	txt := canvas.NewText(h.title, theme.Color(theme.ColorNameForeground))
	txt.TextSize = theme.TextSize()
	r := &tileHeaderRenderer{h: h, bg: bg, txt: txt, objs: []fyne.CanvasObject{bg, txt}}
	return r
}

type tileHeaderRenderer struct {
	h    *tileHeader
	bg   *canvas.Rectangle
	txt  *canvas.Text
	objs []fyne.CanvasObject
}

func (r *tileHeaderRenderer) Destroy() {}
func (r *tileHeaderRenderer) Objects() []fyne.CanvasObject { return r.objs }

func (r *tileHeaderRenderer) Layout(size fyne.Size) {
	r.bg.Resize(size)
	ts := r.txt.MinSize()
	r.txt.Move(fyne.NewPos(theme.Padding(), (size.Height-ts.Height)/2))
	r.txt.Resize(ts)
}

func (r *tileHeaderRenderer) MinSize() fyne.Size {
	ts := r.txt.MinSize()
	pad := theme.Padding()
	return fyne.NewSize(ts.Width+pad*3, fyne.Max(ts.Height+pad, 22))
}

func (r *tileHeaderRenderer) Refresh() {
	label := r.h.title
	if r.h.active {
		label = "● " + label
	}
	r.txt.Text = label
	r.txt.Color = theme.Color(theme.ColorNameForeground)
	r.txt.TextSize = theme.TextSize()
	if r.h.hover {
		r.bg.FillColor = theme.Color(theme.ColorNameHover)
	} else if r.h.active {
		r.bg.FillColor = theme.Color(theme.ColorNamePrimary)
		r.txt.Color = theme.Color(theme.ColorNameForegroundOnPrimary)
	} else {
		r.bg.FillColor = theme.Color(theme.ColorNameButton)
	}
	r.bg.Refresh()
	r.txt.Refresh()
}
