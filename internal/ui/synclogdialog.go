package ui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/scottpeterman/pathfinderssh/internal/synclog"
)

// ShowSyncLogDialog opens the MSP sync error/event log for troubleshooting.
func ShowSyncLogDialog(w fyne.Window) {
	if w == nil {
		return
	}
	home := GetAppHome()
	path := synclog.Path(home)

	body := widget.NewMultiLineEntry()
	body.Wrapping = fyne.TextWrapWord
	body.SetMinRowsVisible(18)
	refresh := func() {
		text, err := synclog.ReadTail(home, 384*1024)
		if err != nil {
			body.SetText("read log: " + err.Error())
			return
		}
		body.SetText(text)
		body.CursorRow = strings.Count(text, "\n")
	}
	refresh()

	tips := widget.NewLabel(
		"Tips: HTTP 403 = no access to that Auvik tenant (often MSP root).\n" +
			"Wrong base URL = use https://auvikapi.<region>.my.auvik.com.\n" +
			"AuvikTunnel errors = check Settings → Integrations → AuvikTunnel path; see also logs/auvik-tunnel.log.\n" +
			"High skipped = end-user devices filtered (uncheck Infra only to include all).\n" +
			"Log file: " + path,
	)
	tips.Wrapping = fyne.TextWrapWord

	reload := widget.NewButton("Refresh", refresh)
	clear := widget.NewButton("Clear log", func() {
		dialog.ShowConfirm("Clear sync log", "Delete all entries in the MSP sync log?", func(ok bool) {
			if !ok {
				return
			}
			if err := synclog.Clear(home); err != nil {
				dialog.ShowError(err, w)
				return
			}
			refresh()
		}, w)
	})
	copyPath := widget.NewButton("Copy log path", func() {
		w.Clipboard().SetContent(path)
		dialog.ShowInformation("Sync log", "Path copied:\n"+path, w)
	})
	openFolder := widget.NewButton("Open logs folder", func() {
		if err := revealInFileManager(synclog.Dir(home)); err != nil {
			dialog.ShowError(err, w)
		}
	})

	content := container.NewBorder(
		container.NewVBox(tips, container.NewHBox(reload, clear, copyPath, openFolder)),
		nil, nil, nil,
		container.NewScroll(body),
	)
	d := dialog.NewCustom("MSP sync log / troubleshoot", "Close", content, w)
	d.Resize(fyne.NewSize(720, 520))
	d.Show()
}

// ShowSyncResultDialog shows a sync summary and offers the log when there were errors.
func ShowSyncResultDialog(w fyne.Window, title, summary string, errCount int) {
	if w == nil {
		return
	}
	msg := strings.TrimSpace(summary)
	if errCount > 0 {
		msg += fmt.Sprintf("\n\n%d error(s) were written to the MSP sync log.", errCount)
	}
	label := widget.NewLabel(msg)
	label.Wrapping = fyne.TextWrapWord
	scroll := container.NewScroll(label)
	scroll.SetMinSize(fyne.NewSize(520, 220))

	openLog := widget.NewButton("Open sync log…", func() { ShowSyncLogDialog(w) })
	box := container.NewBorder(nil, container.NewHBox(openLog), nil, nil, scroll)
	d := dialog.NewCustom(title, "Close", box, w)
	d.Resize(fyne.NewSize(560, 360))
	d.Show()
}

func revealInFileManager(dir string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("explorer", dir).Start()
	case "darwin":
		return exec.Command("open", dir).Start()
	default:
		return exec.Command("xdg-open", dir).Start()
	}
}
