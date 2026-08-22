// New-customer crawl seed import wizard (CSV).
package ui

import (
	"fmt"
	"io"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/scottpeterman/pathfinderssh/internal/crawlcsv"
	"github.com/scottpeterman/pathfinderssh/internal/sessions"
)

// CustomerCrawlImport is the result of the CSV wizard.
type CustomerCrawlImport struct {
	Customer string
	// Folders are relative paths under Customers/<customer>/ → session nodes.
	Folders map[string][]sessions.Node
	// SeedHosts are crawl starting addresses (from the CSV host column).
	SeedHosts []string
	StartCrawl bool
}

// CustomerCrawlImportOptions wires inventory helpers into the wizard.
type CustomerCrawlImportOptions struct {
	ExistingCustomers []string
	CreateCustomer    func(name string) (string, error)
	// HomeDir for maps folder creation hint (optional).
	HomeDir string
}

// ShowCustomerCrawlImportWizard: create/pick customer → CSV → import seeds.
func ShowCustomerCrawlImportWizard(w fyne.Window, opts CustomerCrawlImportOptions, onDone func(CustomerCrawlImport)) {
	step := 0
	const nSteps = 3

	header := widget.NewLabel("")
	hint := widget.NewLabel("")
	hint.Wrapping = fyne.TextWrapWord
	status := statusLabel()

	// --- customer ---
	mode := widget.NewRadioGroup([]string{
		"Create a new customer",
		"Use an existing customer",
	}, nil)
	mode.Required = true
	mode.SetSelected("Create a new customer")

	newName := entryWith("")
	newName.SetPlaceHolder("Customer name")
	existing := widget.NewSelect(append([]string(nil), opts.ExistingCustomers...), nil)
	if len(opts.ExistingCustomers) > 0 {
		existing.SetSelected(opts.ExistingCustomers[0])
	}
	existing.Disable()

	mode.OnChanged = func(s string) {
		if strings.HasPrefix(s, "Create") {
			newName.Enable()
			existing.Disable()
		} else {
			newName.Disable()
			existing.Enable()
		}
	}

	page0 := container.NewVBox(
		mode,
		widget.NewSeparator(),
		widget.NewLabel("New customer"),
		newName,
		widget.NewSeparator(),
		widget.NewLabel("Existing"),
		existing,
	)

	resolveCustomer := func() (string, error) {
		if strings.HasPrefix(mode.Selected, "Create") {
			name := strings.TrimSpace(newName.Text)
			if name == "" {
				return "", fmt.Errorf("enter a customer name")
			}
			if opts.CreateCustomer == nil {
				return "", fmt.Errorf("cannot create customer here")
			}
			if _, err := opts.CreateCustomer(name); err != nil {
				return "", err
			}
			return name, nil
		}
		name := strings.TrimSpace(existing.Selected)
		if name == "" {
			return "", fmt.Errorf("pick a customer")
		}
		return name, nil
	}

	// --- CSV ---
	preview := widget.NewLabel("No CSV loaded yet.")
	preview.Wrapping = fyne.TextWrapWord
	var parsed []crawlcsv.Row
	csvPath := widget.NewLabel("")
	csvPath.TextStyle = fyne.TextStyle{Monospace: true}

	loadCSV := func(path string, data []byte) {
		rows, err := crawlcsv.ParseBytes(data)
		if err != nil {
			parsed = nil
			preview.SetText(err.Error())
			status.SetText(err.Error())
			return
		}
		parsed = rows
		csvPath.SetText(path)
		var b strings.Builder
		fmt.Fprintf(&b, "%d devices in CSV\n", len(rows))
		limit := len(rows)
		if limit > 12 {
			limit = 12
		}
		for i := 0; i < limit; i++ {
			r := rows[i]
			folder := r.Folder
			if folder == "" {
				folder = "seeds"
			}
			fmt.Fprintf(&b, "• %s (%s) → %s\n", r.Name, r.Host, folder)
		}
		if len(rows) > limit {
			fmt.Fprintf(&b, "… and %d more\n", len(rows)-limit)
		}
		preview.SetText(b.String())
		status.SetText(fmt.Sprintf("%d rows ready", len(rows)))
	}

	downloadTpl := widget.NewButtonWithIcon("Download CSV template…", theme.DocumentSaveIcon(), func() {
		save := dialog.NewFileSave(func(uc fyne.URIWriteCloser, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if uc == nil {
				return
			}
			defer uc.Close()
			if _, err := uc.Write([]byte(crawlcsv.Template)); err != nil {
				dialog.ShowError(err, w)
				return
			}
			status.SetText("Template saved — fill host column, then load the file here.")
		}, w)
		save.SetFileName(crawlcsv.TemplateFileName)
		save.Resize(fyne.NewSize(820, 600))
		save.Show()
	})

	pickCSV := widget.NewButtonWithIcon("Load filled CSV…", theme.FolderOpenIcon(), func() {
		open := dialog.NewFileOpen(func(uc fyne.URIReadCloser, err error) {
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			if uc == nil {
				return
			}
			defer uc.Close()
			data, err := io.ReadAll(uc)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}
			loadCSV(uc.URI().Path(), data)
		}, w)
		open.SetFilter(storage.NewExtensionFileFilter([]string{".csv"}))
		open.Resize(fyne.NewSize(820, 600))
		open.Show()
	})

	formatHelp := widget.NewLabel(
		"Columns: host (required), name, protocol (ssh|telnet), port, username, folder, notes.\n" +
			"folder is under the customer (default: seeds). Download the template for examples.",
	)
	formatHelp.Wrapping = fyne.TextWrapWord

	page1 := container.NewBorder(
		container.NewVBox(
			formatHelp,
			container.NewHBox(downloadTpl, pickCSV),
			csvPath,
		),
		nil, nil, nil,
		container.NewVScroll(preview),
	)

	// --- confirm ---
	startCrawl := widget.NewCheck("Open the crawl wizard next (with these hosts as starting devices)", nil)
	startCrawl.SetChecked(true)
	confirmLab := widget.NewLabel("Create customer folder, import sessions, and create maps/<customer>/.")
	confirmLab.Wrapping = fyne.TextWrapWord
	page2 := container.NewVBox(confirmLab, startCrawl)

	pages := []fyne.CanvasObject{page0, page1, page2}
	pageBox := container.NewStack(pages...)
	titles := []string{"Customer", "Seed CSV", "Import"}
	hints := []string{
		"New customer crawls start with a customer folder under Customers.",
		"Download the template, add real hosts, then load the CSV.",
		"Sessions land under Customers/<name>/<folder>/. Maps go under maps/<name>/.",
	}

	showPage := func() {
		for i, p := range pages {
			if i == step {
				p.Show()
			} else {
				p.Hide()
			}
		}
		header.SetText(fmt.Sprintf("Step %d of %d — %s", step+1, nSteps, titles[step]))
		hint.SetText(hints[step])
		status.SetText("")
	}

	var d dialog.Dialog
	back := widget.NewButton("Back", nil)
	next := widget.NewButton("Next", nil)
	back.OnTapped = func() {
		if step > 0 {
			step--
			showPage()
		}
	}
	next.OnTapped = func() {
		switch step {
		case 0:
			// Defer CreateCustomer until final import so Cancel is clean —
			// but validate name now.
			if strings.HasPrefix(mode.Selected, "Create") {
				if strings.TrimSpace(newName.Text) == "" {
					status.SetText("enter a customer name")
					return
				}
			} else if strings.TrimSpace(existing.Selected) == "" {
				status.SetText("pick a customer")
				return
			}
			step++
			showPage()
		case 1:
			if len(parsed) == 0 {
				status.SetText("load a CSV first (or download the template and fill it)")
				return
			}
			step++
			showPage()
		case 2:
			customer, err := resolveCustomer()
			if err != nil {
				// Existing customer path; create may have failed if duplicate on create mode
				status.SetText(err.Error())
				dialog.ShowError(err, w)
				return
			}
			if opts.HomeDir != "" {
				_, _ = EnsureCustomerMapsDir(opts.HomeDir, customer)
			}
			grouped := crawlcsv.GroupByFolder(parsed)
			hosts := make([]string, 0, len(parsed))
			seen := map[string]bool{}
			for _, r := range parsed {
				if seen[r.Host] {
					continue
				}
				seen[r.Host] = true
				hosts = append(hosts, r.Host)
			}
			d.Hide()
			if onDone != nil {
				onDone(CustomerCrawlImport{
					Customer:   customer,
					Folders:    grouped,
					SeedHosts:  hosts,
					StartCrawl: startCrawl.Checked,
				})
			}
		}
	}

	nav := container.NewBorder(nil, nil, back, next, status)
	body := container.NewBorder(
		container.NewVBox(header, hint, widget.NewSeparator()),
		nav, nil, nil, pageBox,
	)
	showPage()
	d = dialog.NewCustom("Import customer crawl seeds", "Close", body, w)
	d.Resize(fyne.NewSize(640, 520))
	d.Show()
}
