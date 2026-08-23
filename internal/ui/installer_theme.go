package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// Windows 11–style installer chrome: light background, readable Segoe UI, accent blue.
type installerTheme struct {
	base *NativeTheme
}

// NewInstallerTheme returns a light, high-contrast theme for setup wizards.
func NewInstallerTheme() fyne.Theme {
	return &installerTheme{base: NewNativeTheme(AppLight)}
}

// ApplyInstallerTheme installs the Windows-style light installer theme on a Fyne app.
func ApplyInstallerTheme(a fyne.App) {
	a.Settings().SetTheme(NewInstallerTheme())
}

func (t *installerTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNamePrimary:
		return color.NRGBA{R: 0, G: 103, B: 192, A: 255} // Windows accent #0067C0
	case theme.ColorNameBackground:
		return color.NRGBA{R: 243, G: 243, B: 243, A: 255} // Mica-like light gray
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	case theme.ColorNameForeground:
		return color.NRGBA{R: 32, G: 32, B: 32, A: 255}
	case theme.ColorNameDisabled:
		return color.NRGBA{R: 120, G: 120, B: 120, A: 255}
	case theme.ColorNamePlaceHolder:
		return color.NRGBA{R: 96, G: 96, B: 96, A: 255}
	}
	return t.base.Color(name, variant)
}

func (t *installerTheme) Font(style fyne.TextStyle) fyne.Resource {
	return t.base.Font(style)
}

func (t *installerTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return t.base.Icon(name)
}

func (t *installerTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNameText:
		return 15
	case theme.SizeNameInnerPadding:
		return 10
	case theme.SizeNamePadding:
		return 6
	case theme.SizeNameHeadingText:
		return 22
	case theme.SizeNameSubHeadingText:
		return 17
	}
	return t.base.Size(name)
}
