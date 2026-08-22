// internal/ui/sftp_nav.go
// Small SFTP helpers shared with settings / theme icon embeds.
package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// LoadSftpNavSettings returns on-disk settings for the SFTP dialog, falling
// back to CurrentSettings when the file cannot be read.
func LoadSftpNavSettings() Settings {
	if s, err := LoadSettings(SettingsPath()); err == nil {
		return s
	}
	return CurrentSettings()
}

func themedEmbedIcon(name string) fyne.Resource {
	res := resourceFromEmbed(name)
	if res == nil {
		return theme.MoveUpIcon()
	}
	return theme.NewThemedResource(res)
}
