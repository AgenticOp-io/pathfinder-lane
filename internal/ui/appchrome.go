// App chrome: slim top toolbar + bottom ops dock (not an Office ribbon).
package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/scottpeterman/pathfinderssh/internal/buttons"
)

// AppChromeConfig wires host actions into top bar + bottom dock.
type AppChromeConfig struct {
	OnQuickConnect func()
	OnCrawl        func()
	CrawlMenu      func(anchor fyne.CanvasObject)
	OnCapture      func()
	OnMap          func()
	OnSearch       func()
	ScriptsMenu    func(anchor fyne.CanvasObject)
	TabsMenu       func(anchor fyne.CanvasObject)
	OnSettings     func()
	OnCursor       func()

	// RecentBar sits under Connect/Crawl (full window width, above sessions+tabs).
	RecentBar fyne.CanvasObject

	Customers      []string
	OnAdhocConnect func(host, user string, port int)
	OnSendChat     func(text string, mode ChatSendMode, customer string)
	BarButtons     []buttons.Button
	OnBarAction    func(b buttons.Button, all bool)
	OnBarEdit      func()
	ShowSendDock   bool
	WorkContext    *widget.Label
}

// AppChrome is the main window chrome above and below the tab strip.
type AppChrome struct {
	top     fyne.CanvasObject
	opsDock *OpsDock
}

func (c *AppChrome) Top() fyne.CanvasObject    { return c.top }
func (c *AppChrome) Bottom() fyne.CanvasObject { return c.opsDock.Content() }

// SetConnected shows or hides the button bar and send row without rebuilding widgets.
func (c *AppChrome) SetConnected(connected bool) {
	if c != nil && c.opsDock != nil {
		c.opsDock.SetConnected(connected)
	}
}

// BuildAppChrome assembles toolbar + ops dock once per window lifetime.
func BuildAppChrome(cfg AppChromeConfig) *AppChrome {
	top := buildTopBar(cfg)
	dock := newOpsDock(cfg)
	return &AppChrome{top: top, opsDock: dock}
}

func buildTopBar(cfg AppChromeConfig) fyne.CanvasObject {
	launch := container.NewHBox(
		toolbarAction("Connect", "New terminal session", theme.ComputerIcon(), cfg.OnQuickConnect),
		crawlToolbarButton(cfg),
		toolbarAction("Capture", "Packet capture", theme.DownloadIcon(), cfg.OnCapture),
		toolbarAction("Map", "Topology map", theme.GridIcon(), cfg.OnMap),
		toolbarAction("Search", "Search inventory", theme.SearchReplaceIcon(), cfg.OnSearch),
	)

	if cfg.ScriptsMenu != nil {
		var scriptsBtn *TipButton
		scriptsBtn = TipButtonLabeled("Scripts", theme.DocumentIcon(), func() {
			cfg.ScriptsMenu(scriptsBtn)
		})
		scriptsBtn.Importance = widget.MediumImportance
		scriptsBtn.SetToolTip("Run, record, or edit YAML scripts")
		launch.Add(scriptsBtn)
	}

	if cfg.OnCursor != nil {
		launch.Add(toolbarAction("Cursor", "Open Cursor AI in a separate window", theme.HelpIcon(), cfg.OnCursor))
	}

	var tabsBtn *TipButton
	if cfg.TabsMenu != nil {
		tabsBtn = TipButtonLabeled("Tabs", theme.ListIcon(), func() {
			cfg.TabsMenu(tabsBtn)
		})
		tabsBtn.Importance = widget.MediumImportance
		tabsBtn.SetToolTip("Switch or close open tabs")
	}

	var settingsBtn *TipButton
	if cfg.OnSettings != nil {
		settingsBtn = TipButtonLabeled("Settings", theme.SettingsIcon(), cfg.OnSettings)
		settingsBtn.Importance = widget.MediumImportance
		settingsBtn.SetToolTip("Preferences, vault, import/export, paths")
	}

	right := container.NewHBox()
	if tabsBtn != nil {
		right.Add(tabsBtn)
	}
	if settingsBtn != nil {
		right.Add(settingsBtn)
	}

	toolbar := container.NewPadded(container.NewBorder(nil, nil, launch, right, chromeSpacer()))
	toolbar = container.New(&chromeMinHeight{h: toolbarH}, toolbar)

	if cfg.RecentBar != nil {
		recent := container.NewPadded(cfg.RecentBar)
		return container.NewVBox(toolbar, recent)
	}
	return toolbar
}

const toolbarH float32 = 46

type chromeMinHeight struct{ h float32 }

func (chromeMinHeight) MinSize(objects []fyne.CanvasObject) fyne.Size {
	ms := fyne.NewSize(0, 0)
	for _, o := range objects {
		ms = ms.Max(o.MinSize())
	}
	if ms.Height < toolbarH {
		ms.Height = toolbarH
	}
	return ms
}

func (l chromeMinHeight) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	h := size.Height
	if h < l.h {
		h = l.h
	}
	for _, o := range objects {
		o.Resize(fyne.NewSize(size.Width, h))
		o.Move(fyne.NewPos(0, 0))
	}
}

func toolbarAction(label, tip string, icon fyne.Resource, fn func()) fyne.CanvasObject {
	if fn == nil {
		fn = func() {}
	}
	b := TipButtonLabeled(label, icon, fn)
	b.Importance = widget.MediumImportance
	if tip != "" {
		b.SetToolTip(tip)
	}
	return b
}

func crawlToolbarButton(cfg AppChromeConfig) fyne.CanvasObject {
	if cfg.CrawlMenu != nil {
		var btn *TipButton
		btn = TipButtonLabeled("Crawl", theme.SearchIcon(), func() {
			cfg.CrawlMenu(btn)
		})
		btn.Importance = widget.MediumImportance
		btn.SetToolTip("Discover network, crawl seeds, and related import actions")
		return btn
	}
	return toolbarAction("Crawl", "Network crawl", theme.SearchIcon(), cfg.OnCrawl)
}

func chromeSpacer() fyne.CanvasObject {
	r := canvas.NewRectangle(theme.Color(theme.ColorNameBackground))
	r.SetMinSize(fyne.NewSize(1, 1))
	return r
}

// OpsDock is the persistent bottom strip (button bar, optional status).
type OpsDock struct {
	root      fyne.CanvasObject
	buttonRow fyne.CanvasObject
	statusBar fyne.CanvasObject
	workCtx   *widget.Label
	note      *widget.Label
}

func (d *OpsDock) Content() fyne.CanvasObject {
	if d == nil {
		return nil
	}
	return d.root
}

// SetWorkContext updates the incident label in the bottom status strip.
func (c *AppChrome) SetWorkContext(text string) {
	if c == nil || c.opsDock == nil {
		return
	}
	d := c.opsDock
	if d.workCtx != nil {
		d.workCtx.SetText(text)
		if strings.TrimSpace(text) == "" {
			d.workCtx.Hide()
		} else {
			d.workCtx.Show()
		}
	}
	d.refreshStatusBar()
}

// SetStatusNote shows a short transient note (sync result, etc.) or clears it.
func (c *AppChrome) SetStatusNote(text string) {
	if c == nil || c.opsDock == nil || c.opsDock.note == nil {
		return
	}
	d := c.opsDock
	d.note.SetText(text)
	if strings.TrimSpace(text) == "" {
		d.note.Hide()
	} else {
		d.note.Show()
	}
	d.refreshStatusBar()
}

func (d *OpsDock) refreshStatusBar() {
	if d == nil || d.statusBar == nil {
		return
	}
	hasWork := d.workCtx != nil && !d.workCtx.Hidden && strings.TrimSpace(d.workCtx.Text) != ""
	hasNote := d.note != nil && !d.note.Hidden && strings.TrimSpace(d.note.Text) != ""
	if hasWork || hasNote {
		d.statusBar.Show()
	} else {
		d.statusBar.Hide()
	}
	if d.root != nil {
		d.root.Refresh()
	}
}

func (d *OpsDock) SetConnected(connected bool) {
	if d == nil {
		return
	}
	if d.buttonRow != nil {
		if connected {
			d.buttonRow.Show()
		} else {
			d.buttonRow.Hide()
		}
	}
	if d.root != nil {
		d.root.Refresh()
	}
}

func newOpsDock(cfg AppChromeConfig) *OpsDock {
	buttonRow := newButtonBarRow(cfg.BarButtons, cfg.OnBarAction, cfg.OnBarEdit)
	// Do not dock the shell tab-count summary here — tabs already show what is
	// open, and an empty "nothing open" line just wasted bottom space.

	workCtx := cfg.WorkContext
	if workCtx == nil {
		workCtx = widget.NewLabel("")
	}
	workCtx.Importance = widget.MediumImportance
	if strings.TrimSpace(workCtx.Text) == "" {
		workCtx.Hide()
	}

	note := widget.NewLabel("")
	note.Importance = widget.LowImportance
	note.Hide()

	left := container.NewHBox(workCtx, note)
	statusBar := container.NewBorder(nil, nil, left, nil, chromeSpacer())
	statusBar.Hide()

	layers := []fyne.CanvasObject{buttonRow}
	d := &OpsDock{buttonRow: buttonRow, workCtx: workCtx, note: note, statusBar: statusBar}
	// Quick-connect + freeform send row removed: Connect is on the top toolbar,
	// and macros / Scripts cover multi-session send without a permanent bar.
	// Cursor Ask lives in the right side pane (Shell.SetRight), not here.
	layers = append(layers, statusBar)
	d.root = container.NewVBox(layers...)
	d.SetConnected(cfg.ShowSendDock)
	d.refreshStatusBar()
	return d
}

func newButtonBarRow(btns []buttons.Button, onAction func(b buttons.Button, all bool), onEdit func()) fyne.CanvasObject {
	// Non-focusable toggle: widget.Check is Focusable and steals keys from SSH
	// as soon as the bar appears on connect.
	allOn := false
	var allChip *macroChip
	allChip = newMacroChip("All tabs: off", func() {
		allOn = !allOn
		if allOn {
			allChip.setLabel("All tabs: on")
		} else {
			allChip.setLabel("All tabs: off")
		}
	})
	row := container.NewHBox()
	for _, b := range btns {
		b := b
		label := strings.TrimSpace(b.Label)
		if label == "" {
			if b.Script != "" {
				label = b.Script
			} else {
				label = "Button"
			}
		}
		row.Add(newMacroChip(label, func() {
			if onAction == nil {
				return
			}
			forceAll := allOn || strings.EqualFold(b.Scope, "all")
			onAction(b, forceAll)
		}))
	}
	if onEdit != nil {
		row.Add(newMacroChip("Edit", onEdit))
	}
	row.Add(allChip)
	return container.NewPadded(row)
}

func newSendRow(customers []string, onSend func(string, ChatSendMode, string)) fyne.CanvasObject {
	sendLbl := widget.NewLabel("Send to")
	sendLbl.Importance = widget.MediumImportance
	modeOpts := []string{"Active tab", "All tabs"}
	for _, c := range customers {
		c = strings.TrimSpace(c)
		if c != "" {
			modeOpts = append(modeOpts, "Customer: "+c)
		}
	}
	mode := widget.NewSelect(modeOpts, nil)
	mode.SetSelected("Active tab")
	cmdEntry := widget.NewEntry()
	cmdEntry.SetPlaceHolder("Command to send to selected sessions…")
	sendBtn := widget.NewButton("Send", func() {
		if onSend == nil {
			return
		}
		text := cmdEntry.Text
		if text == "" {
			return
		}
		if !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
		m, customer := chatModeFromSelect(mode.Selected)
		onSend(text, m, customer)
		cmdEntry.SetText("")
	})
	cmdEntry.OnSubmitted = func(string) { sendBtn.OnTapped() }
	return container.NewPadded(container.NewBorder(nil, nil,
		container.NewHBox(sendLbl, mode),
		sendBtn,
		cmdEntry,
	))
}
