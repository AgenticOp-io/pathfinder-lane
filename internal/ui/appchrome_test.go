package ui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

func TestBuildAppChromeDoesNotPanic(t *testing.T) {
	test.NewApp()
	w := test.NewWindow(nil)
	defer w.Close()

	ch := BuildAppChrome(AppChromeConfig{
		OnQuickConnect: func() {},
		OnCrawl:        func() {},
		OnCapture:      func() {},
		OnMap:          func() {},
		OnSearch:       func() {},
		ScriptsMenu:    func(fyne.CanvasObject) {},
		TabsMenu:       func(fyne.CanvasObject) {},
		OnSettings:     func() {},
		ShowSendDock:   true,
	})
	if ch == nil || ch.Top() == nil || ch.Bottom() == nil {
		t.Fatal("nil chrome")
	}
	s := NewShell(test.NewApp(), w)
	s.SetTopChrome(ch.Top())
	s.SetBottom(ch.Bottom())
	w.SetContent(s.Content())
	w.Resize(fyne.NewSize(1200, 800))
}
