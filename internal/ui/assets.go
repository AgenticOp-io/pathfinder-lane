// Package ui — product images compiled into the binary, with runtime overrides.
//
// # Customizing the logo
//
// Resolution order for Logo() / AppIcon():
//
//  1. Env PATHFINDERSSH_LOGO (or PATHFINDER_LOGO) — path to a .png/.jpg/.svg
//  2. {AppHome}/logo.png          (default ~/.pathfinderssh/logo.png)
//  3. {InstallRoot}/logo.png      (%LOCALAPPDATA%\PathfinderSSH-MSP\logo.png)
//  4. Embedded assets/app-logo.png (AgenticOps mark by default)
//  5. Embedded assets/pathfinderlogo.png (legacy About filename)
//
// The Windows .exe file icon is baked in at build time from
// assets/app-icon.ico (see cmd/pathfinder/winres). Replace that file and
// rebuild to change Explorer / taskbar identity for the PE itself.
package ui

import (
	"embed"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"fyne.io/fyne/v2"

	"github.com/scottpeterman/pathfinderssh/internal/appinstall"
)

//go:embed assets
var assetFS embed.FS

const (
	embeddedLogoFile = "assets/app-logo.png"
	embeddedLegacy   = "assets/pathfinderlogo.png"
	embeddedIconFile = "assets/app-icon.png"
)

var (
	logoOnce   sync.Once
	logoRes    fyne.Resource
	iconOnce   sync.Once
	iconRes    fyne.Resource
	overrideMu sync.Mutex
	// test / product hooks
	logoOverridePath string
)

// SetLogoPath forces Logo()/AppIcon() to load from path (empty clears).
// Intended for tests and for -logo CLI flags.
func SetLogoPath(path string) {
	overrideMu.Lock()
	logoOverridePath = strings.TrimSpace(path)
	overrideMu.Unlock()
	logoOnce = sync.Once{}
	iconOnce = sync.Once{}
	logoRes, iconRes = nil, nil
}

// Logo returns the product splash / About logo, or nil when none is available.
func Logo() fyne.Resource {
	logoOnce.Do(func() {
		logoRes = loadLogoResource(false)
	})
	return logoRes
}

// AppIcon returns a square-ish icon for the window / taskbar, falling back to Logo.
func AppIcon() fyne.Resource {
	iconOnce.Do(func() {
		iconRes = loadLogoResource(true)
		if iconRes == nil {
			iconRes = Logo()
		}
	})
	return iconRes
}

func loadLogoResource(preferIcon bool) fyne.Resource {
	for _, path := range logoCandidatePaths(preferIcon) {
		if res := resourceFromFile(path); res != nil {
			return res
		}
	}
	if preferIcon {
		if res := resourceFromEmbed(embeddedIconFile); res != nil {
			return res
		}
	}
	if res := resourceFromEmbed(embeddedLogoFile); res != nil {
		return res
	}
	return resourceFromEmbed(embeddedLegacy)
}

func logoCandidatePaths(preferIcon bool) []string {
	var out []string
	overrideMu.Lock()
	forced := logoOverridePath
	overrideMu.Unlock()
	if forced != "" {
		out = append(out, forced)
	}
	for _, env := range []string{"PATHFINDERSSH_LOGO", "PATHFINDER_LOGO"} {
		if v := strings.TrimSpace(os.Getenv(env)); v != "" {
			out = append(out, v)
		}
	}
	home := GetAppHome()
	if preferIcon {
		out = append(out,
			filepath.Join(home, "icon.png"),
			filepath.Join(home, "logo.png"),
		)
	} else {
		out = append(out,
			filepath.Join(home, "logo.png"),
			filepath.Join(home, "icon.png"),
		)
	}
	root := appinstall.Root()
	if root != "" {
		out = append(out,
			filepath.Join(root, "logo.png"),
			filepath.Join(root, "icon.png"),
		)
	}
	return out
}

func resourceFromFile(path string) fyne.Resource {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return nil
	}
	name := filepath.Base(path)
	return fyne.NewStaticResource(name, data)
}

func resourceFromEmbed(name string) fyne.Resource {
	data, err := assetFS.ReadFile(name)
	if err != nil || len(data) == 0 {
		return nil
	}
	return fyne.NewStaticResource(filepath.Base(name), data)
}
