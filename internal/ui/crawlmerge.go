package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/scottpeterman/pathfinderssh/internal/crawlrun"
)

// ShowCrawlMergeHintsDialog lists crawl duplicate / low-confidence merge suggestions.
func ShowCrawlMergeHintsDialog(w fyne.Window, suggestions []crawlrun.MergeSuggestion) {
	if w == nil || len(suggestions) == 0 {
		return
	}
	lines := widget.NewList(
		func() int { return len(suggestions) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			if i < 0 || i >= len(suggestions) {
				return
			}
			s := suggestions[i]
			o.(*widget.Label).SetText(fmt.Sprintf("%s ↔ %s (%s)", s.NameA, s.NameB, s.Reason))
		},
	)
	body := container.NewVBox(
		widget.NewLabel("Review duplicate IPs and low-confidence rows.\n"+
			"Merge sessions manually in the session tree or re-crawl after fixing inventory."),
		container.NewScroll(lines),
	)
	dialog.ShowCustom("Crawl merge hints", "Close", body, w)
}

// FormatMergeHints returns a clipboard-friendly summary.
func FormatMergeHints(suggestions []crawlrun.MergeSuggestion) string {
	var parts []string
	for _, s := range suggestions {
		parts = append(parts, fmt.Sprintf("%s ↔ %s — %s", s.NameA, s.NameB, s.Reason))
	}
	return strings.Join(parts, "\n")
}
