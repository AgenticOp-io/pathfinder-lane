package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const (
	sideDrawerRailW float32 = 40
	sideDrawerOpenW float32 = 220
)

// SideDrawer is a pin/hover left rail: collapsed to a thin strip by default,
// expands on hover or when pinned.
type SideDrawer struct {
	widget.BaseWidget

	inner     fyne.CanvasObject
	pinned    bool
	hoverOpen bool

	rail      *fyne.Container
	panel     *fyne.Container
	root      *fyne.Container
	pinBtn    *widget.Button
	onChange  func()
	onPinned  func(pinned bool)
}

// NewSideDrawer wraps inventory (or any) content in a collapsible rail.
func NewSideDrawer(inner fyne.CanvasObject, pinned bool, onChange func(), onPinned func(bool)) *SideDrawer {
	d := &SideDrawer{
		inner:    inner,
		pinned:   pinned,
		onChange: onChange,
		onPinned: onPinned,
	}
	d.ExtendBaseWidget(d)

	openBtn := widget.NewButtonWithIcon("", theme.ListIcon(), func() {
		d.SetPinned(true)
	})
	openBtn.Importance = widget.LowImportance
	d.rail = container.NewVBox(openBtn)

	d.pinBtn = widget.NewButtonWithIcon("", theme.NavigateBackIcon(), nil)
	d.pinBtn.Importance = widget.LowImportance
	d.pinBtn.OnTapped = func() {
		d.SetPinned(!d.pinned)
	}
	d.syncPinIcon()

	header := container.NewBorder(nil, nil, nil, d.pinBtn,
		widget.NewLabelWithStyle("Sessions", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	d.panel = container.NewBorder(header, nil, nil, nil, inner)

	d.root = container.NewMax()
	d.rebuild()
	return d
}

// Content returns the drawer widget for Shell.SetSide.
func (d *SideDrawer) Content() fyne.CanvasObject {
	if d == nil {
		return nil
	}
	return d
}

// Pinned reports whether the drawer stays open.
func (d *SideDrawer) Pinned() bool {
	return d != nil && d.pinned
}

// SetPinned expands and keeps the drawer open, or returns to hover mode.
func (d *SideDrawer) SetPinned(on bool) {
	if d == nil {
		return
	}
	d.pinned = on
	if on {
		d.hoverOpen = false
	}
	d.syncPinIcon()
	d.rebuild()
	if d.onPinned != nil {
		d.onPinned(on)
	}
	if d.onChange != nil {
		d.onChange()
	}
}

func (d *SideDrawer) syncPinIcon() {
	if d == nil || d.pinBtn == nil {
		return
	}
	if d.pinned {
		d.pinBtn.SetIcon(theme.NavigateBackIcon())
		d.pinBtn.SetText("Hide")
		d.pinBtn.Importance = widget.MediumImportance
	} else {
		d.pinBtn.SetIcon(theme.ViewFullScreenIcon())
		d.pinBtn.SetText("Pin")
		d.pinBtn.Importance = widget.LowImportance
	}
	d.pinBtn.Refresh()
}

func (d *SideDrawer) expanded() bool {
	return d.pinned || d.hoverOpen
}

func (d *SideDrawer) rebuild() {
	if d.root == nil {
		return
	}
	if d.expanded() {
		d.root.Objects = []fyne.CanvasObject{d.panel}
	} else {
		d.root.Objects = []fyne.CanvasObject{d.rail}
	}
	d.root.Refresh()
	d.Refresh()
}

func (d *SideDrawer) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(d.root)
}

func (d *SideDrawer) MinSize() fyne.Size {
	if d.expanded() {
		return fyne.NewSize(sideDrawerOpenW, 120)
	}
	return fyne.NewSize(sideDrawerRailW, 120)
}

func (d *SideDrawer) MouseIn(*desktop.MouseEvent) {
	if d.pinned {
		return
	}
	d.hoverOpen = true
	d.rebuild()
	if d.onChange != nil {
		d.onChange()
	}
}

func (d *SideDrawer) MouseMoved(*desktop.MouseEvent) {}

func (d *SideDrawer) MouseOut() {
	if d.pinned {
		return
	}
	d.hoverOpen = false
	d.rebuild()
	if d.onChange != nil {
		d.onChange()
	}
}

func (d *SideDrawer) Cursor() desktop.Cursor {
	if d.expanded() {
		return desktop.DefaultCursor
	}
	return desktop.PointerCursor
}

var _ desktop.Hoverable = (*SideDrawer)(nil)
