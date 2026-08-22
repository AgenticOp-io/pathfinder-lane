package ui

import (
	"strings"

	"fyne.io/fyne/v2/widget"

	"github.com/scottpeterman/pathfinderssh/internal/mspsync"
)

// CustomerFolderPicker selects an existing Customers/<name>/ folder or a new name.
type CustomerFolderPicker struct {
	Select *widget.Select
	New    *widget.Entry
}

// NewCustomerFolderPicker builds widgets. suggested is a PSA/Auvik/doc org label for pre-selection.
func NewCustomerFolderPicker(existing []string, suggested string) *CustomerFolderPicker {
	sel := widget.NewSelect(existing, nil)
	newName := widget.NewEntry()
	newName.SetPlaceHolder("Or type a new customer folder name")

	if len(existing) == 0 {
		sel.Hide()
		newName.SetText(strings.TrimSpace(suggested))
		return &CustomerFolderPicker{Select: sel, New: newName}
	}

	resolved := mspsync.ResolveCustomerName(existing, suggested)
	if resolved != "" && customerInList(existing, resolved) {
		sel.SetSelected(resolved)
	} else {
		sel.SetSelected(existing[0])
	}

	sel.OnChanged = func(string) {
		if sel.Selected != "" {
			newName.SetText("")
		}
	}
	return &CustomerFolderPicker{Select: sel, New: newName}
}

// Chosen returns the customer folder name to use.
func (p *CustomerFolderPicker) Chosen() string {
	if p == nil {
		return ""
	}
	if t := strings.TrimSpace(p.New.Text); t != "" {
		return t
	}
	return strings.TrimSpace(p.Select.Selected)
}

// SetSuggested re-selects the folder best matching an external org/tenant label.
func (p *CustomerFolderPicker) SetSuggested(existing []string, suggested string) {
	if p == nil || p.Select == nil {
		return
	}
	resolved := mspsync.ResolveCustomerName(existing, suggested)
	if resolved != "" && customerInList(existing, resolved) {
		p.Select.SetSelected(resolved)
		if p.New != nil {
			p.New.SetText("")
		}
	}
}

func customerInList(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}
