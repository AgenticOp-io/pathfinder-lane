//go:build !windows

package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

func chromeFont(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}
