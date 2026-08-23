// File transfer (SFTP) over an existing SSH connection — dialog or shell tab.
//
// Uses the same crypto/ssh client as the open terminal — no second login.
package ui

import (
	"errors"
	"sync/atomic"
	"fmt"
	"os"
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

// SFTPView is an Applet for docking SFTP in a DocTabs tab.
type SFTPView struct {
	content fyne.CanvasObject
	cli     *sftpclient.Client
	win     fyne.Window
}

// DialSFTP opens the SFTP subsystem on an existing SSH client.
// Safe to call off the UI thread (network only; no widgets).
func DialSFTP(sshClient *ssh.Client) (*sftpclient.Client, error) {
	if sshClient == nil {
		return nil, fmt.Errorf("SFTP requires an SSH client")
	}
	return sftpclient.Open(sshClient)
}

// NewSFTPViewFromClient builds the browser UI for an already-open SFTP client.
// Call on the UI thread.
func NewSFTPViewFromClient(w fyne.Window, cli *sftpclient.Client) *SFTPView {
	return NewSFTPViewFromSeed(w, cli, SFTPSeed{})
}

// SFTPSeed is a prefetched directory listing for opening the browser without
// touching the network on the UI thread.
type SFTPSeed struct {
	Dir  string
	Ents []sftpclient.Entry
	Err  error
}

// PrefetchSFTP lists "." off the UI thread. It never calls Getwd/RealPath.
func PrefetchSFTP(cli *sftpclient.Client) SFTPSeed {
	if cli == nil {
		return SFTPSeed{Dir: ".", Err: fmt.Errorf("SFTP client is nil")}
	}
	ents, err := cli.List(".")
	return SFTPSeed{Dir: ".", Ents: ents, Err: err}
}

// NewSFTPViewFromSeed builds the browser from a PrefetchSFTP result.
func NewSFTPViewFromSeed(w fyne.Window, cli *sftpclient.Client, seed SFTPSeed) *SFTPView {
	dir := seed.Dir
	if dir == "" {
		dir = "."
	}
	return NewSFTPViewSeeded(w, cli, dir, seed.Ents, seed.Err)
}

// NewSFTPViewSeeded builds the browser with a prefetched listing so the UI
// thread never waits on SFTP after the tab opens.
func NewSFTPViewSeeded(w fyne.Window, cli *sftpclient.Client, dir string, ents []sftpclient.Entry, err error) *SFTPView {
	v := &SFTPView{cli: cli, win: w}
	v.content = buildSFTPBody(w, cli, dir, ents, err)
	return v
}

// NewSFTPView opens the SFTP subsystem and builds the browser UI.
// Prefer DialSFTP + NewSFTPViewFromClient from the host so the handshake
// does not freeze the Fyne thread.
func NewSFTPView(w fyne.Window, sshClient *ssh.Client) (*SFTPView, error) {
	cli, err := DialSFTP(sshClient)
	if err != nil {
		return nil, err
	}
	if w == nil {
		_ = cli.Close()
		return nil, fmt.Errorf("SFTP requires a window")
	}
	return NewSFTPViewFromClient(w, cli), nil
}

func (v *SFTPView) Content() fyne.CanvasObject { return v.content }
func (v *SFTPView) Start()                     {}
func (v *SFTPView) Stop() {
	if v != nil && v.cli != nil {
		_ = v.cli.Close()
		v.cli = nil
	}
}

// ShowSFTPDialog opens a remote browser as a modal. Prefer OpenSFTPTab for MSP.
func ShowSFTPDialog(w fyne.Window, title string, sshClient *ssh.Client) {
	if w == nil || sshClient == nil {
		return
	}
	cli, err := sftpclient.Open(sshClient)
	if err != nil {
		dialog.ShowError(fmt.Errorf("SFTP: %w", err), w)
		return
	}
	body := buildSFTPBody(w, cli, "", nil, nil)
	if title == "" {
		title = "File Transfer (SFTP)"
	}
	d := dialog.NewCustom(title, "Close", body, w)
	d.SetOnClosed(func() { _ = cli.Close() })
	d.Resize(fyne.NewSize(720, 520))
	d.Show()
}

func buildSFTPBody(w fyne.Window, cli *sftpclient.Client, initDir string, initEnts []sftpclient.Entry, initErr error) fyne.CanvasObject {
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

	var refreshGen uint64
	applyListing := func(dir string, ents []sftpclient.Entry, err error, gen uint64) {
		if gen != refreshGen {
			return // superseded by a newer refresh
		}
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

	var refresh func()
	refresh = func() {
		dir := strings.TrimSpace(pathEntry.Text)
		if dir == "" {
			dir = "."
		}
		refreshGen++
		gen := refreshGen
		status.SetText("Loading...")
		go func(dir string, gen uint64) {
			ents, err := cli.List(dir)
			fyne.Do(func() { applyListing(dir, ents, err, gen) })
		}(dir, gen)
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

	homeDir := "."
	goHome := func() {
		pathEntry.SetText(homeDir)
		refresh()
	}
	// Never call Getwd/RealPath on open. pkg/sftp Getwd is RealPath("."), and
	// RealPath hangs on many network-OS SSH daemons — that wedges the shared
	// SSH mux (terminal + SFTP) and freezes the app after the tab appears.
	// List(".") is the SFTP session cwd and is what those boxes expect.
	if initDir != "" {
		homeDir = initDir
		pathEntry.SetText(initDir)
	}
	if initErr != nil {
		status.SetText(initErr.Error())
	} else if initEnts != nil {
		refreshGen++
		applyListing(homeDir, initEnts, nil, refreshGen)
	} else {
		status.SetText("Loading...")
		go func() {
			ents, err := cli.List(".")
			fyne.Do(func() {
				refreshGen++
				applyListing(".", ents, err, refreshGen)
			})
		}()
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
			runSFTPTransferWindow(w, "Download", e.Name, e.Size, status, func(report sftpclient.ProgressFunc, ctrl *sftpclient.TransferControl) error {
				err := cli.DownloadProgress(e.Path, local, report, ctrl)
				if errors.Is(err, sftpclient.ErrStopped) {
					_ = os.Remove(local) // drop partial download
				}
				return err
			}, nil)
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
			var total int64
			if st, err := os.Stat(local); err == nil {
				total = st.Size()
			}
			runSFTPTransferWindow(w, "Upload", base, total, status, func(report sftpclient.ProgressFunc, ctrl *sftpclient.TransferControl) error {
				return cli.UploadProgress(local, remote, report, ctrl)
			}, refresh)
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
			status.SetText("Creating folder...")
			go func() {
				err := cli.Mkdir(remote)
				fyne.Do(func() {
					if err != nil {
						status.SetText(err.Error())
						dialog.ShowError(err, w)
						return
					}
					refresh()
				})
			}()
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
			status.SetText("Deleting...")
			go func(path string) {
				err := cli.Remove(path)
				fyne.Do(func() {
					if err != nil {
						status.SetText(err.Error())
						dialog.ShowError(err, w)
						return
					}
					refresh()
				})
			}(e.Path)
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
	status.SetText("Loading...")
	return body
}


// runSFTPTransferWindow opens a detached transfer window with pause / resume / stop.
func runSFTPTransferWindow(parent fyne.Window, title, fileName string, total int64, status *widget.Label, work func(report sftpclient.ProgressFunc, ctrl *sftpclient.TransferControl) error, onOK func()) {
	ctrl := sftpclient.NewTransferControl()
	app := fyne.CurrentApp()
	if app == nil {
		dialog.ShowError(fmt.Errorf("no application for transfer window"), parent)
		return
	}
	win := app.NewWindow(title + " — " + fileName)
	win.Resize(fyne.NewSize(480, 220))

	bar := widget.NewProgressBar()
	if total > 0 {
		bar.Max = float64(total)
	} else {
		bar.Max = 1
	}
	detail := widget.NewLabel("Starting…")
	detail.Wrapping = fyne.TextWrapWord
	state := widget.NewLabel("Transferring")
	name := widget.NewLabel(fileName)
	name.TextStyle = fyne.TextStyle{Bold: true}

	pauseBtn := widget.NewButton("Pause", nil)
	resumeBtn := widget.NewButton("Resume", nil)
	stopBtn := widget.NewButton("Stop", nil)
	resumeBtn.Disable()

	setRunning := func() {
		state.SetText("Transferring")
		pauseBtn.Enable()
		resumeBtn.Disable()
		stopBtn.Enable()
	}
	setPaused := func() {
		state.SetText("Paused")
		pauseBtn.Disable()
		resumeBtn.Enable()
		stopBtn.Enable()
	}
	setStopping := func() {
		state.SetText("Stopping…")
		pauseBtn.Disable()
		resumeBtn.Disable()
		stopBtn.Disable()
	}

	pauseBtn.OnTapped = func() {
		ctrl.Pause()
		setPaused()
	}
	resumeBtn.OnTapped = func() {
		ctrl.Resume()
		setRunning()
	}
	stopBtn.OnTapped = func() {
		ctrl.Stop()
		setStopping()
	}

	finished := false
	closeWin := func() {
		if finished {
			return
		}
		finished = true
		win.SetCloseIntercept(nil)
		win.Close()
	}

	win.SetCloseIntercept(func() {
		// Closing the window stops the transfer.
		ctrl.Stop()
		setStopping()
	})

	buttons := container.NewHBox(pauseBtn, resumeBtn, stopBtn)
	win.SetContent(container.NewPadded(container.NewVBox(
		name,
		state,
		bar,
		detail,
		buttons,
	)))
	win.Show()
	status.SetText(title + "…")

	// Coalesce progress onto the UI thread: at most one fyne.Do in flight.
	// Unbounded fyne.Do from the copy goroutine made the app look frozen.
	var progDone, progTotal atomic.Int64
	var progSched atomic.Bool
	scheduleProgress := func(done, tot int64) {
		progDone.Store(done)
		progTotal.Store(tot)
		if !progSched.CompareAndSwap(false, true) {
			return
		}
		fyne.Do(func() {
			for {
				d := progDone.Load()
				tot := progTotal.Load()
				progSched.Store(false)
				if tot > 0 {
					bar.Max = float64(tot)
					bar.SetValue(float64(d))
					detail.SetText(fmt.Sprintf("%s / %s", formatSize(d), formatSize(tot)))
				} else {
					detail.SetText(formatSize(d) + " transferred")
				}
				// If a newer sample arrived while we painted, schedule once more.
				if progDone.Load() == d {
					return
				}
				if !progSched.CompareAndSwap(false, true) {
					return
				}
			}
		})
	}

	go func() {
		err := work(scheduleProgress, ctrl)
		fyne.Do(func() {
			pauseBtn.Disable()
			resumeBtn.Disable()
			stopBtn.Disable()
			if errors.Is(err, sftpclient.ErrStopped) {
				state.SetText("Stopped")
				status.SetText(title + " stopped: " + fileName)
				detail.SetText("Transfer stopped")
				closeWin()
				return
			}
			if err != nil {
				state.SetText("Failed")
				status.SetText(err.Error())
				dialog.ShowError(err, parent)
				closeWin()
				return
			}
			if total > 0 {
				bar.SetValue(float64(total))
			} else {
				bar.SetValue(1)
			}
			state.SetText("Complete")
			detail.SetText("Finished")
			status.SetText(title + " complete: " + fileName)
			if onOK != nil {
				onOK()
			}
			closeWin()
		})
	}()
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
