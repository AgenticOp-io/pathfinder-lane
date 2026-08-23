package mspbranding

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/scottpeterman/pathfinderssh/internal/product"
)

const brandingFileName = "msp-branding.json"
const logoFileName = "logo.png"

// Branding is MSP chrome for installers and engineer workstations.
type Branding struct {
	OrgDisplayName string    `json:"org_display_name,omitempty"`
	ProductTitle   string    `json:"product_title,omitempty"`
	AccentHex      string    `json:"accent_hex,omitempty"`
	UpdatedAt      time.Time `json:"updated_at,omitempty"`
}

func installRoot() string {
	if runtime.GOOS == "windows" {
		if base := os.Getenv("LOCALAPPDATA"); base != "" {
			return filepath.Join(base, product.InstallDir)
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return product.InstallDir
	}
	return filepath.Join(home, ".pathfinderssh-msp-app")
}

// Path returns the branding JSON path beside enrollment.
func Path() string {
	return filepath.Join(installRoot(), brandingFileName)
}

// LogoPath returns the install-root logo.png path.
func LogoPath() string {
	return filepath.Join(installRoot(), logoFileName)
}

// Load reads branding from the install root.
func Load() (Branding, bool, error) {
	return LoadFile(Path())
}

// LoadFile reads branding from an explicit path.
func LoadFile(path string) (Branding, bool, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Branding{}, false, nil
		}
		return Branding{}, false, err
	}
	var b Branding
	if err := json.Unmarshal(raw, &b); err != nil {
		return Branding{}, false, fmt.Errorf("parse branding: %w", err)
	}
	return b, true, nil
}

// Save writes branding to the install root and refreshes UpdatedAt.
func Save(b Branding) error {
	b.OrgDisplayName = strings.TrimSpace(b.OrgDisplayName)
	b.ProductTitle = strings.TrimSpace(b.ProductTitle)
	b.AccentHex = strings.TrimSpace(b.AccentHex)
	if b.UpdatedAt.IsZero() {
		b.UpdatedAt = time.Now()
	}
	path := Path()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// InstallTitle returns the branded product title for installer chrome.
func (b Branding) InstallTitle() string {
	if t := strings.TrimSpace(b.ProductTitle); t != "" {
		return t
	}
	if o := strings.TrimSpace(b.OrgDisplayName); o != "" {
		return o + " Pathfinder"
	}
	return "PathfinderSSH MSP"
}

// InstallSubtitle returns installer subtitle text.
func (b Branding) InstallSubtitle() string {
	if o := strings.TrimSpace(b.OrgDisplayName); o != "" {
		return o + " — engineer workstation setup"
	}
	return "Install tools for this Windows profile"
}

// CopyLogoFrom copies a user-selected image into the install root as logo.png.
func CopyLogoFrom(src string) error {
	src = strings.TrimSpace(src)
	if src == "" {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open logo: %w", err)
	}
	defer in.Close()
	dst := LogoPath()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp := dst + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	_ = os.Remove(dst)
	return os.Rename(tmp, dst)
}
