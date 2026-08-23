package ui

import (
	"os"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/scottpeterman/pathfinderssh/internal/mspbranding"
)

// BrandingForm collects MSP display branding for installers.
type BrandingForm struct {
	orgName   *widget.Entry
	product   *widget.Entry
	accent    *widget.Entry
	logoPath  *widget.Entry
	preview   *widget.Icon
	content   fyne.CanvasObject
}

// NewBrandingForm builds branding fields with optional preset values.
func NewBrandingForm(w fyne.Window, preset mspbranding.Branding) *BrandingForm {
	f := &BrandingForm{}
	f.orgName = widget.NewEntry()
	f.orgName.SetPlaceHolder("e.g. Contoso MSP")
	f.product = widget.NewEntry()
	f.product.SetPlaceHolder("e.g. Contoso NOC Console")
	f.accent = widget.NewEntry()
	f.accent.SetPlaceHolder("#0067C0 (optional Windows accent)")
	f.logoPath = widget.NewEntry()
	f.logoPath.SetPlaceHolder("PNG logo for installer and About box")
	f.preview = widget.NewIcon(theme.AccountIcon())

	f.orgName.SetText(preset.OrgDisplayName)
	f.product.SetText(preset.ProductTitle)
	f.accent.SetText(preset.AccentHex)

	browse := widget.NewButtonWithIcon("Choose logo…", theme.FolderOpenIcon(), func() {
		dialog.NewFileOpen(func(uc fyne.URIReadCloser, err error) {
			if err != nil || uc == nil {
				return
			}
			defer uc.Close()
			path := uc.URI().Path()
			f.logoPath.SetText(path)
			f.refreshPreview(path)
		}, w).Show()
	})

	f.content = container.NewVBox(
		widget.NewLabel("Branding appears on engineer installers and the Pathfinder window."),
		widget.NewForm(
			widget.NewFormItem("Organization name", f.orgName),
			widget.NewFormItem("Product title", f.product),
			widget.NewFormItem("Accent color", f.accent),
			widget.NewFormItem("Logo file", container.NewBorder(nil, nil, nil, browse, f.logoPath)),
		),
		widget.NewLabel("Preview"),
		container.NewCenter(f.preview),
	)
	return f
}

func (f *BrandingForm) Content() fyne.CanvasObject {
	if f == nil {
		return widget.NewLabel("")
	}
	return f.content
}

func (f *BrandingForm) Branding() mspbranding.Branding {
	if f == nil {
		return mspbranding.Branding{}
	}
	return mspbranding.Branding{
		OrgDisplayName: strings.TrimSpace(f.orgName.Text),
		ProductTitle:   strings.TrimSpace(f.product.Text),
		AccentHex:      strings.TrimSpace(f.accent.Text),
	}
}

func (f *BrandingForm) LogoSourcePath() string {
	if f == nil {
		return ""
	}
	return strings.TrimSpace(f.logoPath.Text)
}

// Save applies logo copy and writes msp-branding.json to the install root.
func (f *BrandingForm) Save() error {
	b := f.Branding()
	if err := mspbranding.Save(b); err != nil {
		return err
	}
	if src := f.LogoSourcePath(); src != "" {
		if err := mspbranding.CopyLogoFrom(src); err != nil {
			return err
		}
	}
	SetLogoPath(mspbranding.LogoPath())
	return nil
}

func (f *BrandingForm) refreshPreview(path string) {
	if path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		return
	}
	f.preview.SetResource(fyne.NewStaticResource(path, data))
}
