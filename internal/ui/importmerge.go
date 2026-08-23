package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/scottpeterman/pathfinderssh/internal/sessions"
)

// ShowImportMergeReport shows a scrollable merge/diff summary after import.
func ShowImportMergeReport(w fyne.Window, title string, sum sessions.ImportSummary) {
	if w == nil {
		return
	}
	body := widget.NewLabel(sum.MergeReport())
	body.Wrapping = fyne.TextWrapWord
	scroll := container.NewScroll(body)
	scroll.SetMinSize(fyne.NewSize(560, 320))
	d := dialog.NewCustom(title, "OK", scroll, w)
	d.Resize(fyne.NewSize(640, 420))
	d.Show()
}
