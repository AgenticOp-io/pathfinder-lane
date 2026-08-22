// File transfer (SFTP) dialog over an existing SSH connection.
//
// Uses the same crypto/ssh client as the open terminal — no second login.
// SCP is not implemented; SFTP is the modern equivalent Pathfinder ships.
package ui

import (
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/crypto/ssh"

	"github.com/scottpeterman/pathfinderssh/internal/sftpclient"
)

// ShowSFTPDialog opens a remote browser for the given SSH client.
// Caller keeps the SSH session alive; this only closes the SFTP subsystem.
func ShowSFTPDialog(w fyne.Window, title string, sshClient *ssh.Client) {
	if w == nil || sshClient == nil {
		return
	}
	cli, err := sftpclient.Open(sshClient)
	if err != nil {
		dialog.ShowError(fmt.Errorf("SFTP: %w", err), w)
		return
	}

	remoteDir := "."
	var entries []sftpclient.Entry
	selected := -1

	status := widget.NewLabel("Ready")
	status.Wrapping = fyne.TextWrapWord

	pathEntry := widget.NewEntry()
	pathEntry.SetText(remoteDir)

	list := widget.NewList(
		func() int { return len(entries) },
		func() fyne.CanvasObject {
			return container.NewBorder(nil, nil, widget.NewLabel(""), nil, widget.NewLabel(""))
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			if i < 0 || i >= len(entries) {
				return
			}
			row, ok := o.(*fyne.Container)
			if !ok || len(row.Objects) < 2 {
				return
			}
			meta, _ := row.Objects[0].(*widget.Label)
			name, _ := row.Objects[1].(*widget.Label)
			if name == nil || meta == nil {
				return
			}
			e := entries[i]
			name.TextStyle = fyne.TextStyle{Monospace: true}
			if e.IsDir {
				name.SetText(e.Name + "/")
				meta.SetText("dir")
			} else {
				name.SetText(e.Name)
				meta.SetText(formatSize(e.Size))
			}
		},
	)

	var refresh func()
	refresh = func() {
		dir := strings.TrimSpace(pathEntry.Text)
		if dir == "" {
			dir = "."
		}
		ents, err := cli.List(dir)
		if err != nil {
			status.SetText(err.Error())
			return
		}
		sort.Slice(ents, func(i, j int) bool {
			if ents[i].IsDir != ents[j].IsDir {
				return ents[i].IsDir
			}
			return strings.ToLower(ents[i].Name) < strings.ToLower(ents[j].Name)
		})
		remoteDir = dir
		pathEntry.SetText(dir)
		entries = ents
		selected = -1
		list.UnselectAll()
		list.Refresh()
		status.SetText(fmt.Sprintf("%d items in %s", len(ents), dir))
	}

	list.OnSelected = func(i widget.ListItemID) { selected = int(i) }
	list.OnUnselected = func(widget.ListItemID) { selected = -1 }

	goUp := func() {
		dir := strings.TrimSpace(pathEntry.Text)
		if dir == "" || dir == "." || dir == "/" {
			return
		}
		parent := path.Dir(dir)
		if parent == dir {
			parent = "/"
		}
		pathEntry.SetText(parent)
		refresh()
	}

	homeDir := "/"
	if wd, err := cli.Getwd(); err == nil && strings.TrimSpace(wd) != "" {
		homeDir = wd
	}
	if rp, err := cli.RealPath("~"); err == nil && strings.TrimSpace(rp) != "" {
		homeDir = rp
	}
	goHome := func() {
		pathEntry.SetText(homeDir)
		refresh()
	}

	pathEntry.OnSubmitted = func(string) { refresh() }

	var downloadSel func()
	downloadSel = func() {
		if selected < 0 || selected >= len(entries) {
			status.SetText("Select a file to download")
			return
		}
		e := entries[selected]
		if e.IsDir {
			status.SetText("Download applies to files; open a directory first")
			return
		}
		save := dialog.NewFileSave(func(uc fyne.URIWriteCloser, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if uc == nil {
				return
			}
			local := uc.URI().Path()
			_ = uc.Close()
			status.SetText("Downloading…")
			go func() {
				err := cli.Download(e.Path, local)
				fyne.Do(func() {
					if err != nil {
						status.SetText(err.Error())
						dialog.ShowError(err, w)
						return
					}
					status.SetText("Downloaded " + filepath.Base(local))
				})
			}()
		}, w)
		save.SetFileName(e.Name)
		save.Resize(fyne.NewSize(820, 600))
		save.Show()
	}

	openSel := func() {
		if selected < 0 || selected >= len(entries) {
			status.SetText("Select a folder to open, or a file to download")
			return
		}
		e := entries[selected]
		if e.IsDir {
			pathEntry.SetText(e.Path)
			refresh()
			return
		}
		downloadSel()
	}

	uploadFile := func() {
		open := dialog.NewFileOpen(func(uc fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if uc == nil {
				return
			}
			local := uc.URI().Path()
			_ = uc.Close()
			base := filepath.Base(local)
			remote := sftpclient.Join(remoteDir, base)
			status.SetText("Uploading…")
			go func() {
				err := cli.Upload(local, remote)
				fyne.Do(func() {
					if err != nil {
						status.SetText(err.Error())
						dialog.ShowError(err, w)
						return
					}
					status.SetText("Uploaded " + base)
					refresh()
				})
			}()
		}, w)
		open.Resize(fyne.NewSize(820, 600))
		if home, err := osUserHome(); err == nil {
			if l, err := storage.ListerForURI(storage.NewFileURI(home)); err == nil {
				open.SetLocation(l)
			}
		}
		open.Show()
	}

	mkdir := func() {
		name := widget.NewEntry()
		name.SetPlaceHolder("folder name")
		dialog.ShowForm("New remote folder", "Create", "Cancel", []*widget.FormItem{
			{Text: "Name", Widget: name},
		}, func(ok bool) {
			if !ok {
				return
			}
			n := strings.TrimSpace(name.Text)
			if n == "" {
				return
			}
			remote := sftpclient.Join(remoteDir, n)
			if err := cli.Mkdir(remote); err != nil {
				dialog.ShowError(err, w)
				return
			}
			refresh()
		}, w)
	}

	removeSel := func() {
		if selected < 0 || selected >= len(entries) {
			status.SetText("Select an item to delete")
			return
		}
		e := entries[selected]
		dialog.ShowConfirm("Delete", "Delete remote "+e.Path+"?", func(ok bool) {
			if !ok {
				return
			}
			if err := cli.Remove(e.Path); err != nil {
				dialog.ShowError(err, w)
				return
			}
			refresh()
		}, w)
	}

	cfg := LoadSftpNavSettings()
	upBtn := widget.NewButtonWithIcon("Up", theme.MoveUpIcon(), goUp)
	tools := []fyne.CanvasObject{upBtn}
	if cfg.SftpShowHome {
		tools = append([]fyne.CanvasObject{
			widget.NewButtonWithIcon("Home", theme.HomeIcon(), goHome),
		}, tools...)
	}
	tools = append(tools,
		widget.NewButtonWithIcon("Open", theme.FolderOpenIcon(), openSel),
		widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), refresh),
		widget.NewButtonWithIcon("Upload…", theme.UploadIcon(), uploadFile),
		widget.NewButtonWithIcon("Download…", theme.DownloadIcon(), downloadSel),
		widget.NewButtonWithIcon("New folder…", theme.FolderNewIcon(), mkdir),
		widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), removeSel),
	)
	toolbar := container.NewHBox(tools...)

	body := container.NewBorder(
		container.NewBorder(nil, nil, widget.NewLabel("Remote"), nil, pathEntry),
		status,
		nil, nil,
		container.NewBorder(toolbar, nil, nil, nil, list),
	)

	if title == "" {
		title = "File Transfer (SFTP)"
	}
	d := dialog.NewCustom(title, "Close", body, w)
	d.SetOnClosed(func() {
		_ = cli.Close()
	})
	d.Resize(fyne.NewSize(720, 520))
	d.Show()
	refresh()
}

func formatSize(n int64) string {
	const (
		kb = 1024
		mb = 1024 * kb
		gb = 1024 * mb
	)
	switch {
	case n >= gb:
		return fmt.Sprintf("%.1f GB", float64(n)/float64(gb))
	case n >= mb:
		return fmt.Sprintf("%.1f MB", float64(n)/float64(mb))
	case n >= kb:
		return fmt.Sprintf("%.1f KB", float64(n)/float64(kb))
	default:
		return fmt.Sprintf("%d B", n)
	}
}

func osUserHome() (string, error) {
	return filepath.Abs(ExpandHome("~"))
}
