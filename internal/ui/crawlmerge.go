package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/scottpeterman/pathfinderssh/internal/crawlrun"
)

// CrawlMergeHintsOptions configures the merge-hints dialog.
type CrawlMergeHintsOptions struct {
	OnFilterTree func(text string)
}

// ShowCrawlMergeHintsDialog lists crawl duplicate / low-confidence merge suggestions.
func ShowCrawlMergeHintsDialog(w fyne.Window, suggestions []crawlrun.MergeSuggestion, opts CrawlMergeHintsOptions) {
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
	selected := -1
	lines.OnSelected = func(i widget.ListItemID) { selected = int(i) }

	body := container.NewVBox(
		widget.NewLabel("Review duplicate IPs and low-confidence rows.\n"+
			"Filter the session tree to locate a row, then merge or delete duplicates manually."),
		container.NewScroll(lines),
	)
	copyBtn := widget.NewButtonWithIcon("Copy all", theme.ContentCopyIcon(), func() {
		w.Clipboard().SetContent(FormatMergeHints(suggestions))
	})
	filterBtn := widget.NewButtonWithIcon("Filter tree to selected", theme.SearchIcon(), func() {
		if selected < 0 || selected >= len(suggestions) || opts.OnFilterTree == nil {
			return
		}
		opts.OnFilterTree(mergeHintFilterText(suggestions[selected]))
	})
	if opts.OnFilterTree == nil {
		filterBtn.Disable()
	}
	buttons := container.NewHBox(copyBtn, filterBtn)
	content := container.NewBorder(nil, buttons, nil, nil, body)
	d := dialog.NewCustom("Crawl merge hints", "Close", content, w)
	d.Resize(fyne.NewSize(640, 420))
	d.Show()
}

func mergeHintFilterText(s crawlrun.MergeSuggestion) string {
	for _, name := range []string{s.NameA, s.NameB} {
		if i := strings.Index(name, " ("); i > 0 {
			return strings.TrimSpace(name[:i])
		}
	}
	if i := strings.Index(s.Reason, "IP "); i >= 0 {
		rest := strings.TrimSpace(s.Reason[i+3:])
		if j := strings.IndexAny(rest, " ,);"); j > 0 {
			rest = rest[:j]
		}
		return rest
	}
	return strings.TrimSpace(s.NameA)
}

// FormatMergeHints returns a clipboard-friendly summary.
func FormatMergeHints(suggestions []crawlrun.MergeSuggestion) string {
	var parts []string
	for _, s := range suggestions {
		parts = append(parts, fmt.Sprintf("%s ↔ %s — %s", s.NameA, s.NameB, s.Reason))
	}
	return strings.Join(parts, "\n")
}
