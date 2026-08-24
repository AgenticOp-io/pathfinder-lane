package ui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type localDirEntry struct {
	name  string
	path  string
	isDir bool
	size  int64
}

type localSFTPPane struct {
	dir       string
	entries   []localDirEntry
	selected  int
	pathEntry *widget.Entry
	list      *widget.List
	content   fyne.CanvasObject
	refresh   func()
}

func newLocalSFTPPane(startDir string) *localSFTPPane {
	p := &localSFTPPane{dir: startDir, selected: -1}
	if p.dir == "" {
		if home, err := osUserHome(); err == nil {
			p.dir = home
		} else {
			p.dir = "."
		}
	}

	p.pathEntry = widget.NewEntry()
	p.pathEntry.SetText(p.dir)

	p.list = widget.NewList(
		func() int { return len(p.entries) },
		func() fyne.CanvasObject {
			return container.NewBorder(nil, nil, widget.NewLabel(""), nil, widget.NewLabel(""))
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			if i < 0 || i >= len(p.entries) {
				return
			}
			row, ok := o.(*fyne.Container)
			if !ok || len(row.Objects) < 2 {
				return
			}
			meta, _ := row.Objects[0].(*widget.Label)
			name, _ := row.Objects[1].(*widget.Label)
			if meta == nil || name == nil {
				return
			}
			e := p.entries[i]
			name.TextStyle = fyne.TextStyle{Monospace: true}
			if e.isDir {
				name.SetText(e.name + "/")
				meta.SetText("dir")
			} else {
				name.SetText(e.name)
				meta.SetText(formatSize(e.size))
			}
		},
	)
	p.list.OnSelected = func(i widget.ListItemID) { p.selected = int(i) }
	p.list.OnUnselected = func(widget.ListItemID) { p.selected = -1 }

	p.refresh = func() {
		dir := strings.TrimSpace(p.pathEntry.Text)
		if dir == "" {
			dir = p.dir
		}
		p.pathEntry.SetText(dir)
		p.dir = dir
		p.selected = -1
		p.list.UnselectAll()
		ents, err := os.ReadDir(dir)
		if err != nil {
			p.entries = nil
			p.list.Refresh()
			return
		}
		p.entries = p.entries[:0]
		for _, e := range ents {
			name := e.Name()
			if strings.HasPrefix(name, ".") {
				continue
			}
			full := filepath.Join(dir, name)
			info, err := e.Info()
			if err != nil {
				continue
			}
			p.entries = append(p.entries, localDirEntry{
				name:  name,
				path:  full,
				isDir: info.IsDir(),
				size:  info.Size(),
			})
		}
		sort.Slice(p.entries, func(i, j int) bool {
			if p.entries[i].isDir != p.entries[j].isDir {
				return p.entries[i].isDir
			}
			return strings.ToLower(p.entries[i].name) < strings.ToLower(p.entries[j].name)
		})
		p.list.Refresh()
	}

	goUp := func() {
		parent := filepath.Dir(p.dir)
		if parent == p.dir {
			return
		}
		p.pathEntry.SetText(parent)
		p.refresh()
	}
	goHome := func() {
		if home, err := osUserHome(); err == nil {
			p.pathEntry.SetText(home)
			p.refresh()
		}
	}
	openSel := func() {
		if p.selected < 0 || p.selected >= len(p.entries) {
			return
		}
		e := p.entries[p.selected]
		if !e.isDir {
			return
		}
		p.pathEntry.SetText(e.path)
		p.refresh()
	}

	p.pathEntry.OnSubmitted = func(string) { p.refresh() }

	toolbar := container.NewHBox(
		widget.NewButtonWithIcon("Up", theme.MoveUpIcon(), goUp),
		widget.NewButtonWithIcon("Home", theme.HomeIcon(), goHome),
		widget.NewButtonWithIcon("Open", theme.FolderOpenIcon(), openSel),
		widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), p.refresh),
	)

	p.content = container.NewBorder(
		container.NewBorder(nil, nil, widget.NewLabel("Local"), nil, p.pathEntry),
		nil, nil, nil,
		container.NewBorder(toolbar, nil, nil, nil, p.list),
	)
	p.refresh()
	return p
}

func (p *localSFTPPane) Content() fyne.CanvasObject { return p.content }

func (p *localSFTPPane) SelectedFile() (path string, ok bool) {
	if p == nil || p.selected < 0 || p.selected >= len(p.entries) {
		return "", false
	}
	e := p.entries[p.selected]
	if e.isDir {
		return "", false
	}
	return e.path, true
}

func (p *localSFTPPane) CurrentDir() string {
	if p == nil {
		return ""
	}
	return p.dir
}
