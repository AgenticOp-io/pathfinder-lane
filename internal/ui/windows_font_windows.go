//go:build windows

package ui

import (
	"os"
	"path/filepath"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

var (
	segoeOnce      sync.Once
	segoeUIRes     fyne.Resource
	segoeUIBoldRes fyne.Resource
)

func loadSegoeUIFonts() {
	windir := os.Getenv("WINDIR")
	if windir == "" {
		windir = "C:\\Windows"
	}
	fonts := filepath.Join(windir, "Fonts")
	if data, err := os.ReadFile(filepath.Join(fonts, "segoeui.ttf")); err == nil {
		segoeUIRes = fyne.NewStaticResource("SegoeUI.ttf", data)
	}
	if data, err := os.ReadFile(filepath.Join(fonts, "segoeuib.ttf")); err == nil {
		segoeUIBoldRes = fyne.NewStaticResource("SegoeUIB.ttf", data)
	}
}

func chromeFont(style fyne.TextStyle) fyne.Resource {
	segoeOnce.Do(loadSegoeUIFonts)
	if style.Monospace {
		return theme.DefaultTheme().Font(style)
	}
	if style.Bold {
		if segoeUIBoldRes != nil {
			return segoeUIBoldRes
		}
	}
	if segoeUIRes != nil {
		return segoeUIRes
	}
	return theme.DefaultTheme().Font(style)
}
