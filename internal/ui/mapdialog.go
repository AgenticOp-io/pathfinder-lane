// internal/ui/mapdialog.go
//
// The map picker: choose a customer (MSP), see that customer's maps, open one.
//
// Maps live under ~/.pathfinderssh/maps/<Customer>/ — same association the
// crawl wizard uses. The dialog does not rename or delete files.
package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/scottpeterman/pathfinderssh/internal/mapweb"
)

// MapLaunch is what the picker produces.
type MapLaunch struct {
	Dir      string
	Customer string
	File     mapweb.MapFile
}

// MapDialogOptions configures the MSP map picker.
type MapDialogOptions struct {
	// MapsRoot is usually ~/.pathfinderssh/maps. Required.
	MapsRoot string
	// Customers are inventory customer names (preferred order).
	Customers []string
	// InitialCustomer pre-selects a customer when present in the list.
	InitialCustomer string
	// InitialDir, when under maps/<Customer>/, also sets the starting customer.
	InitialDir string
}

// ShowMapDialog opens a folder-based picker (legacy). Prefer ShowCustomerMapDialog.
func ShowMapDialog(w fyne.Window, dir string, onOpen func(MapLaunch)) {
	root := MapsRootDir(GetAppHome())
	cust := InferCustomerFromMapsPath(dir)
	ShowCustomerMapDialog(w, MapDialogOptions{
		MapsRoot:        root,
		InitialCustomer: cust,
		InitialDir:      dir,
	}, onOpen)
}

// ShowCustomerMapDialog lists maps for one customer under MapsRoot/<Customer>/.
func ShowCustomerMapDialog(w fyne.Window, opts MapDialogOptions, onOpen func(MapLaunch)) {
	root := strings.TrimSpace(opts.MapsRoot)
	if root == "" {
		root = MapsRootDir(GetAppHome())
	}
	_ = os.MkdirAll(root, 0o755)

	customers := mergeCustomerNames(opts.Customers, listSubdirs(root))

	initial := strings.TrimSpace(opts.InitialCustomer)
	if initial == "" {
		initial = InferCustomerFromMapsPath(opts.InitialDir)
	}
	if initial == "" && len(customers) > 0 {
		initial = customers[0]
	}
	if initial != "" {
		customers = mergeCustomerNames([]string{initial}, customers)
	}

	var files []mapweb.MapFile
	selected := -1

	status := widget.NewLabel("")
	status.Wrapping = fyne.TextWrapWord

	custSel := widget.NewSelect(customers, nil)
	if initial != "" {
		custSel.SetSelected(initial)
	} else if len(customers) > 0 {
		custSel.SetSelected(customers[0])
	}

	folderHint := widget.NewLabel("")
	folderHint.TextStyle = fyne.TextStyle{Monospace: true}
	folderHint.Wrapping = fyne.TextWrapWord

	list := widget.NewList(
		func() int { return len(files) },
		func() fyne.CanvasObject {
			return container.NewBorder(nil, nil, widget.NewLabel(""), nil, widget.NewLabel(""))
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			if i < 0 || i >= len(files) {
				return
			}
			row, ok := o.(*fyne.Container)
			if !ok || len(row.Objects) < 2 {
				return
			}
			summary, _ := row.Objects[0].(*widget.Label)
			name, _ := row.Objects[1].(*widget.Label)
			if name == nil || summary == nil {
				return
			}
			f := files[i]
			name.TextStyle = fyne.TextStyle{Monospace: true}
			name.SetText(f.Name)
			summary.SetText(f.Summary())
		},
	)
	list.OnSelected = func(i widget.ListItemID) { selected = int(i) }
	list.OnUnselected = func(widget.ListItemID) { selected = -1 }

	customerDir := func() string {
		c := strings.TrimSpace(custSel.Selected)
		if c == "" {
			return ""
		}
		return filepath.Join(root, SanitizePathSegment(c))
	}

	refresh := func() {
		selected = -1
		list.UnselectAll()
		dir := customerDir()
		folderHint.SetText(dir)
		if dir == "" {
			files = nil
			status.SetText("Pick a customer. Crawl maps are stored per customer under maps/<name>/.")
			list.Refresh()
			return
		}
		_ = os.MkdirAll(dir, 0o755)
		found, err := mapweb.ListMaps(dir)
		if err != nil {
			files = nil
			status.SetText(err.Error())
			list.Refresh()
			return
		}
		files = found
		switch {
		case len(files) == 0:
			status.SetText("No maps for this customer yet. Run a crawl with this customer selected.")
		case len(files) == 1:
			status.SetText("1 map")
		default:
			status.SetText(fmt.Sprintf("%d maps, newest first", len(files)))
		}
		list.Refresh()
	}

	custSel.OnChanged = func(string) { refresh() }
	reload := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), refresh)
	reload.Importance = widget.LowImportance

	top := container.NewBorder(nil, nil, widget.NewLabel("Customer"), reload, custSel)
	pathRow := container.NewBorder(nil, nil, widget.NewLabel("Folder"), nil, folderHint)
	content := container.NewBorder(
		container.NewVBox(top, pathRow),
		status, nil, nil, list,
	)

	refresh()

	var show func()
	show = func() {
		d := dialog.NewCustomConfirm("Open customer map", "Open", "Cancel", content, func(ok bool) {
			if !ok {
				return
			}
			if selected < 0 || selected >= len(files) {
				status.SetText("Select a map to open.")
				show()
				return
			}
			f := files[selected]
			if !f.OK() {
				status.SetText(f.Name + ": " + f.Problem)
				show()
				return
			}
			onOpen(MapLaunch{
				Dir:      customerDir(),
				Customer: strings.TrimSpace(custSel.Selected),
				File:     f,
			})
		}, w)
		d.Resize(fyne.NewSize(640, 480))
		d.Show()
		EnterConfirms(w, content, d.Confirm)
	}
	show()
}

func listSubdirs(root string) []string {
	ents, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range ents {
		if e.IsDir() {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out
}

func mergeCustomerNames(lists ...[]string) []string {
	seen := map[string]string{} // lower -> display
	var order []string
	for _, list := range lists {
		for _, n := range list {
			n = strings.TrimSpace(n)
			if n == "" {
				continue
			}
			k := strings.ToLower(n)
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = n
			order = append(order, n)
		}
	}
	return order
}
