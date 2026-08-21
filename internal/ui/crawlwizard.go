// internal/ui/crawlwizard.go
//
// MSP crawl wizard. Step 1 is always the customer (existing or new). Starting
// devices, depth, login, and map path follow. "Seeds" is never shown as a word.
package ui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/scottpeterman/pathfinderssh/internal/crawlrun"
	"github.com/scottpeterman/pathfinderssh/internal/product"
	"github.com/scottpeterman/pathfinderssh/internal/sessions"
)

// CrawlSeedOption is one inventory device offered as a crawl starting point.
type CrawlSeedOption struct {
	Label    string // checklist label
	Host     string // Params.Seeds value
	Customer string // customer leaf name under 3_Customers
}

// CrawlWizardOptions carries inventory + customer state into the wizard.
type CrawlWizardOptions struct {
	Prev           CrawlLaunch
	Sessions       []CrawlSeedOption
	Customers      []string // leaf names under CustomersRoot
	CustomersRoot  string
	HomeDir        string
	CreateCustomer func(name string) (string, error) // returns folder path
}

// ShowCrawlWizard walks the operator through a customer-scoped crawl.
func ShowCrawlWizard(w fyne.Window, opts CrawlWizardOptions, onRun func(CrawlLaunch)) {
	prev := opts.Prev
	p := prev.Params
	if p.Depth == 0 {
		p = crawlrun.Defaults()
	}
	root := opts.CustomersRoot
	if root == "" {
		root = product.CustomersRoot
	}
	if root == "" {
		root = sessions.DefaultCustomersRoot
	}

	customers := append([]string(nil), opts.Customers...)
	selectedCustomer := ""
	if prev.MapPath != "" {
		// Best-effort: last map under …/maps/<customer>/…
		parts := strings.Split(filepath.ToSlash(prev.MapPath), "/")
		for i, part := range parts {
			if part == "maps" && i+1 < len(parts) {
				selectedCustomer = parts[i+1]
				break
			}
		}
	}

	step := 0
	const nSteps = 5
	stepTitles := []string{
		"Customer",
		"Starting devices",
		"How far to crawl",
		"How to log in",
		"Where to save the map",
	}
	stepHints := []string{
		"Every crawl belongs to one customer. Pick an existing customer, or create a new one (adds a folder under Customers).",
		"Devices under this customer that you can already reach. The crawl starts here and walks to neighbors.",
		"Depth is how many hops from a starting device. Start shallow on an unfamiliar network.",
		"Use the unlocked vault when you can. Manual login is for a one-off password.",
		"A map path is filled in under this customer so the crawl always leaves a topology you can open later.",
	}

	header := widget.NewLabel("")
	hint := widget.NewLabel("")
	hint.Wrapping = fyne.TextWrapWord
	status := statusLabel()

	// --- step 0: customer -------------------------------------------------
	modeExisting := widget.NewRadioGroup([]string{
		"Use an existing customer",
		"Create a new customer",
	}, nil)
	modeExisting.Required = true
	modeExisting.SetSelected("Use an existing customer")

	customerSel := widget.NewSelect(customerChoices(customers), nil)
	if selectedCustomer != "" {
		customerSel.SetSelected(selectedCustomer)
	} else if len(customers) > 0 {
		customerSel.SetSelected(customers[0])
	}
	newName := entryWith("")
	newName.SetPlaceHolder("Customer name (folder under Customers)")
	newName.Disable()

	modeExisting.OnChanged = func(s string) {
		if strings.HasPrefix(s, "Create") {
			customerSel.Disable()
			newName.Enable()
		} else {
			customerSel.Enable()
			newName.Disable()
		}
	}

	page0 := container.NewVBox(
		modeExisting,
		widget.NewSeparator(),
		widget.NewLabel("Existing customers (from "+root+")"),
		customerSel,
		widget.NewSeparator(),
		widget.NewLabel("New customer"),
		newName,
	)

	resolveCustomer := func() (string, error) {
		if strings.HasPrefix(modeExisting.Selected, "Create") {
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
			customers = appendUnique(customers, name)
			customerSel.Options = customerChoices(customers)
			customerSel.SetSelected(name)
			modeExisting.SetSelected("Use an existing customer")
			modeExisting.OnChanged(modeExisting.Selected)
			return name, nil
		}
		name := strings.TrimSpace(customerSel.Selected)
		if name == "" {
			return "", fmt.Errorf("pick a customer")
		}
		return name, nil
	}

	// --- step 1: starting devices -----------------------------------------
	sessionChecks := widget.NewCheckGroup(nil, nil)
	extraSeeds := multiline("optional: more hosts, one per line")
	selectAll := widget.NewButton("Select all for this customer", nil)
	selectNone := widget.NewButton("Clear", func() { sessionChecks.SetSelected(nil) })
	hostByLabel := map[string]string{}

	refreshSessionList := func(customer string) {
		hostByLabel = map[string]string{}
		var labels []string
		for _, s := range opts.Sessions {
			if customer != "" && !strings.EqualFold(s.Customer, customer) {
				continue
			}
			if s.Host == "" || s.Label == "" {
				continue
			}
			if _, dup := hostByLabel[s.Label]; dup {
				continue
			}
			labels = append(labels, s.Label)
			hostByLabel[s.Label] = s.Host
		}
		sessionChecks.Options = labels
		sessionChecks.SetSelected(nil)
		selectAll.OnTapped = func() {
			sessionChecks.SetSelected(append([]string(nil), labels...))
		}
		if len(labels) == 0 {
			status.SetText("No sessions under this customer yet — type hosts below, or add sessions to their folder.")
		}
	}

	page1 := container.NewBorder(
		container.NewHBox(selectAll, selectNone),
		container.NewVBox(
			widget.NewSeparator(),
			widget.NewLabel("Or type hosts / IPs"),
			tall(extraSeeds, 90),
		),
		nil, nil,
		tall(container.NewVScroll(sessionChecks), 200),
	)

	// --- step 2: depth ----------------------------------------------------
	depthChoices := []string{
		"1 — Neighbors only (quick look)",
		"2 — Nearby (small site)",
		"3 — Wider (typical default)",
		"5 — Deep (large estate)",
	}
	depthSel := widget.NewRadioGroup(depthChoices, nil)
	depthSel.Required = true
	depthSel.SetSelected(depthChoiceFor(p.Depth))
	legacy := widget.NewCheck("Allow older SSH algorithms (legacy gear)", nil)
	legacy.SetChecked(p.Legacy)
	page2 := container.NewVBox(widget.NewLabel("Coverage"), depthSel, widget.NewSeparator(), legacy)

	// --- step 3: credentials ----------------------------------------------
	user := entryWith(prev.Auth.Username)
	pass := widget.NewPasswordEntry()
	pass.SetText(prev.Auth.Password)
	keyPath := entryWith(prev.Auth.KeyPath)
	credTags := entryWith(strings.Join(p.CredTags, ", "))
	vaultOpen := p.VaultPath != ""
	manual := !vaultOpen || prev.ManualCreds
	credSource := credSourceRow(vaultOpen, prev.ManualCreds, func(m bool) {
		manual = m
		applyCredSource(m, credTags, user, pass, keyPath)
	})
	applyCredSource(manual, credTags, user, pass, keyPath)
	vaultNote := widget.NewLabel("Vault is unlocked — credentials there will be tried automatically.")
	if !vaultOpen {
		vaultNote.SetText("Vault is locked. Unlock it from the toolbar, or type a username/password below.")
	}
	page3 := container.NewVBox(
		vaultNote,
		formOf(
			"Credentials from", credSource,
			"Credential tags", credTags,
			"Username", user,
			"Password", pass,
			"Key file", pathRow(w, keyPath, pathOpenFile, ""),
		),
	)

	// --- step 4: output ---------------------------------------------------
	mapPath := entryWith("")
	mapPath.SetPlaceHolder("filled from customer name")
	saveRun := entryWith(prev.SaveRun)
	saveRun.SetPlaceHolder("optional — blank is fine")
	page4 := formOf(
		"Map file (required)", pathRow(w, mapPath, pathOutput, "crawl-map.json"),
		"Save run record (optional)", pathRow(w, saveRun, pathOutput, "last-run.json"),
	)

	setMapDefault := func(customer string) {
		if opts.HomeDir == "" || customer == "" {
			return
		}
		safe := sanitizePathSegment(customer)
		def := filepath.Join(opts.HomeDir, "maps", safe, time.Now().Format("crawl-2006-01-02.json"))
		if strings.TrimSpace(mapPath.Text) == "" || strings.Contains(mapPath.Text, string(filepath.Separator)+"maps"+string(filepath.Separator)) {
			mapPath.SetText(def)
		}
	}

	pages := []fyne.CanvasObject{page0, page1, page2, page3, page4}
	pageBox := container.NewStack(pages...)

	showPage := func() {
		for i, pg := range pages {
			if i == step {
				pg.Show()
			} else {
				pg.Hide()
			}
		}
		header.SetText(fmt.Sprintf("Step %d of %d — %s", step+1, nSteps, stepTitles[step]))
		hint.SetText(stepHints[step])
		status.SetText("")
	}

	collectSeeds := func() []string {
		var hosts []string
		for _, lab := range sessionChecks.Selected {
			if h := hostByLabel[lab]; h != "" {
				hosts = append(hosts, h)
			}
		}
		hosts = append(hosts, crawlrun.ParseSeeds(extraSeeds.Text)...)
		return crawlrun.ParseSeeds(strings.Join(hosts, "\n"))
	}

	buildLaunch := func(customer string) (CrawlLaunch, []string) {
		out := CrawlLaunch{
			Params:      crawlrun.Defaults(),
			Auth:        LaunchAuth{Username: user.Text, Password: pass.Text, KeyPath: ExpandHome(keyPath.Text)},
			ManualCreds: manual,
			MapPath:     ExpandHome(mapPath.Text),
			SaveRun:     ExpandHome(saveRun.Text),
			LastRun:     ExpandHome(prev.LastRun),
			Verbose:     prev.Verbose,
		}
		out.Params.Seeds = collectSeeds()
		out.Params.Depth = depthFromChoice(depthSel.Selected)
		out.Params.Legacy = legacy.Checked
		out.Params.HostKeys = crawlrun.HostKeyTOFU
		if !manual {
			out.Params.VaultPath = p.VaultPath
			out.Params.CredTags = crawlrun.ParseSeeds(credTags.Text)
		}
		var msgs []string
		if customer == "" {
			msgs = append(msgs, "pick or create a customer")
		}
		for _, e := range out.Params.Validate() {
			msg := e.Error()
			if strings.Contains(strings.ToLower(msg), "seed") {
				msg = "pick at least one starting device (from the list or by typing a host)"
			}
			msgs = append(msgs, msg)
		}
		if strings.TrimSpace(out.MapPath) == "" {
			msgs = append(msgs, "map file is required")
		} else if err := checkOutputPath("map file", out.MapPath); err != nil {
			msgs = append(msgs, err.Error())
		}
		if err := checkOutputPath("save run", out.SaveRun); err != nil {
			msgs = append(msgs, err.Error())
		}
		if err := checkInputPath("key file", out.Auth.KeyPath); err != nil {
			msgs = append(msgs, err.Error())
		}
		return out, msgs
	}

	var d dialog.Dialog
	backBtn := widget.NewButton("Back", nil)
	nextBtn := widget.NewButton("Next", nil)
	startBtn := widget.NewButtonWithIcon("Start crawl", theme.MediaPlayIcon(), nil)
	startBtn.Importance = widget.HighImportance
	advancedBtn := widget.NewButton("Advanced form…", nil)

	updateNav := func() {
		if step <= 0 {
			backBtn.Disable()
		} else {
			backBtn.Enable()
		}
		if step >= nSteps-1 {
			nextBtn.Hide()
			startBtn.Show()
		} else {
			nextBtn.Show()
			startBtn.Hide()
		}
	}

	backBtn.OnTapped = func() {
		if step > 0 {
			step--
			showPage()
			updateNav()
		}
	}
	nextBtn.OnTapped = func() {
		if step == 0 {
			name, err := resolveCustomer()
			if err != nil {
				status.SetText("⚠  " + err.Error())
				return
			}
			selectedCustomer = name
			refreshSessionList(selectedCustomer)
			setMapDefault(selectedCustomer)
		}
		if step == 1 && len(collectSeeds()) == 0 {
			status.SetText("⚠  pick or type at least one starting device")
			return
		}
		if step < nSteps-1 {
			step++
			showPage()
			updateNav()
		}
	}
	startBtn.OnTapped = func() {
		if selectedCustomer == "" {
			name, err := resolveCustomer()
			if err != nil {
				status.SetText("⚠  " + err.Error())
				return
			}
			selectedCustomer = name
			setMapDefault(selectedCustomer)
		}
		out, msgs := buildLaunch(selectedCustomer)
		if len(msgs) > 0 {
			status.SetText("⚠  " + strings.Join(msgs, " · "))
			return
		}
		d.Hide()
		onRun(out)
	}
	advancedBtn.OnTapped = func() {
		if selectedCustomer == "" {
			if name, err := resolveCustomer(); err == nil {
				selectedCustomer = name
				setMapDefault(selectedCustomer)
			}
		}
		draft, _ := buildLaunch(selectedCustomer)
		d.Hide()
		ShowCrawlDialog(w, draft, onRun)
	}

	nav := container.NewBorder(nil, nil,
		container.NewHBox(backBtn, advancedBtn),
		container.NewHBox(nextBtn, startBtn),
		nil,
	)
	content := container.NewBorder(
		container.NewVBox(header, hint, widget.NewSeparator()),
		container.NewVBox(status, nav),
		nil, nil,
		pageBox,
	)

	d = dialog.NewCustom("Discover network — "+product.Name, "Cancel", content, w)
	d.Resize(fyne.NewSize(740, 640))
	showPage()
	updateNav()
	d.Show()
}

func customerChoices(names []string) []string {
	out := make([]string, 0, len(names))
	for _, n := range names {
		n = strings.TrimSpace(n)
		if n != "" {
			out = append(out, n)
		}
	}
	return out
}

func appendUnique(list []string, name string) []string {
	for _, x := range list {
		if strings.EqualFold(x, name) {
			return list
		}
	}
	return append(list, name)
}

func sanitizePathSegment(s string) string {
	s = strings.TrimSpace(s)
	repl := strings.NewReplacer(`/`, `_`, `\`, `_`, `:`, `_`, `*`, `_`, `?`, `_`, `"`, `_`, `<`, `_`, `>`, `_`, `|`, `_`)
	return repl.Replace(s)
}

func depthChoiceFor(d int) string {
	switch {
	case d <= 1:
		return "1 — Neighbors only (quick look)"
	case d == 2:
		return "2 — Nearby (small site)"
	case d >= 5:
		return "5 — Deep (large estate)"
	default:
		return "3 — Wider (typical default)"
	}
}

func depthFromChoice(s string) int {
	switch {
	case strings.HasPrefix(s, "1"):
		return 1
	case strings.HasPrefix(s, "2"):
		return 2
	case strings.HasPrefix(s, "5"):
		return 5
	default:
		return 3
	}
}
