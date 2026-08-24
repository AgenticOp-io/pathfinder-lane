package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/scottpeterman/pathfinderssh/internal/sessions"
)

// ImportMergeOptions configures the post-import merge report dialog.
type ImportMergeOptions struct {
	OnFilterTree func(text string)
}

// ShowImportMergeReport shows a scrollable merge/diff summary after import.
func ShowImportMergeReport(w fyne.Window, title string, sum sessions.ImportSummary, opts ImportMergeOptions) {
	if w == nil {
		return
	}
	body := widget.NewLabel(sum.MergeReport())
	body.Wrapping = fyne.TextWrapWord
	scroll := container.NewScroll(body)
	scroll.SetMinSize(fyne.NewSize(560, 320))
	copyBtn := widget.NewButtonWithIcon("Copy report", theme.ContentCopyIcon(), func() {
		w.Clipboard().SetContent(sum.MergeReport())
	})
	filterBtn := widget.NewButtonWithIcon("Filter tree (first skipped)", theme.SearchIcon(), func() {
		if opts.OnFilterTree == nil {
			return
		}
		for _, fr := range sum.Results {
			if len(fr.Skipped) > 0 {
				opts.OnFilterTree(fr.Skipped[0])
				return
			}
		}
	})
	if opts.OnFilterTree == nil {
		filterBtn.Disable()
	}
	buttons := container.NewHBox(copyBtn, filterBtn)
	content := container.NewBorder(nil, buttons, nil, nil, scroll)
	d := dialog.NewCustom(title, "OK", content, w)
	d.Resize(fyne.NewSize(640, 420))
	d.Show()
}
