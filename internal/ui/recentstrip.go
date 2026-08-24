package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/scottpeterman/pathfinderssh/internal/recent"
	"github.com/scottpeterman/pathfinderssh/internal/sessions"
)

// recentStrip shows recently opened sessions as compact chips above the tree.
type recentStrip struct {
	box    *fyne.Container
	tree   sessions.Tree
	onOpen func(folder string, n sessions.Node)
}

func newRecentStrip(onOpen func(folder string, n sessions.Node)) *recentStrip {
	return &recentStrip{
		box:    container.NewHBox(),
		onOpen: onOpen,
	}
}

func (r *recentStrip) rebuild(path string, tree sessions.Tree) {
	if r == nil || r.box == nil {
		return
	}
	r.tree = tree
	r.box.Objects = nil
	if path == "" || r.onOpen == nil {
		r.box.Refresh()
		return
	}
	entries, err := recent.Load(path)
	if err != nil || len(entries) == 0 {
		r.box.Refresh()
		return
	}
	max := 8
	if len(entries) > max {
		entries = entries[:max]
	}
	r.box.Add(widget.NewLabelWithStyle("Recent:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
	for _, e := range entries {
		e := e
		label := e.Name
		if label == "" {
			label = e.Host
		}
		if label == "" {
			continue
		}
		btn := widget.NewButton(label, func() {
			n, ok := r.tree.SessionInFolder(e.Folder, e.Name)
			if !ok {
				return
			}
			r.onOpen(e.Folder, n)
		})
		btn.Importance = widget.MediumImportance
		r.box.Add(btn)
	}
	clearBtn := widget.NewButtonWithIcon("", theme.DeleteIcon(), func() {
		_ = recent.Clear(path)
		r.rebuild(path, tree)
	})
	clearBtn.Importance = widget.MediumImportance
	r.box.Add(clearBtn)
	r.box.Refresh()
}

func (r *recentStrip) Content() fyne.CanvasObject {
	if r == nil || r.box == nil {
		return widget.NewLabel("")
	}
	return r.box
}
