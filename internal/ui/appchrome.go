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
	OnCapture      func()
	OnMap          func()
	OnSearch       func()
	ScriptsMenu    func(anchor fyne.CanvasObject)
	TabsMenu       func(anchor fyne.CanvasObject)
	OnSettings     func()

	Customers      []string
	OnAdhocConnect func(host, user string, port int)
	OnSendChat     func(text string, mode ChatSendMode, customer string)
	BarButtons     []buttons.Button
	OnBarAction    func(b buttons.Button, all bool)
	OnBarEdit      func()
	ShowSendDock   bool
	Status         *widget.Label
	WorkContext    *widget.Label

	// Cursor AI side pane (Troubleshoot addon).
	ShowCursorAI bool
	CursorAIOpen bool
	OnCursorAI   func()
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
		toolbarAction("Crawl", "Network crawl", theme.SearchIcon(), cfg.OnCrawl),
		toolbarAction("Capture", "Packet capture", theme.DownloadIcon(), cfg.OnCapture),
		toolbarAction("Map", "Topology map", theme.GridIcon(), cfg.OnMap),
		toolbarAction("Search", "Search inventory", theme.SearchReplaceIcon(), cfg.OnSearch),
	)

	if cfg.ScriptsMenu != nil {
		var scriptsBtn *TipButton
		scriptsBtn = TipButtonLabeled("Scripts", theme.DocumentIcon(), func() {
			cfg.ScriptsMenu(scriptsBtn)
		})
		scriptsBtn.Importance = widget.LowImportance
		scriptsBtn.SetToolTip("Run, record, or edit YAML scripts")
		launch.Add(scriptsBtn)
	}

	var tabsBtn *TipButton
	if cfg.TabsMenu != nil {
		tabsBtn = TipButtonLabeled("Tabs", theme.ListIcon(), func() {
			cfg.TabsMenu(tabsBtn)
		})
		tabsBtn.Importance = widget.LowImportance
		tabsBtn.SetToolTip("Switch or close open tabs")
	}

	var settingsBtn *TipButton
	if cfg.OnSettings != nil {
		settingsBtn = TipButtonLabeled("Settings", theme.SettingsIcon(), cfg.OnSettings)
		settingsBtn.Importance = widget.LowImportance
		settingsBtn.SetToolTip("Preferences, vault, import/export, paths")
	}

	right := container.NewHBox()
	if tabsBtn != nil {
		right.Add(tabsBtn)
	}
	if settingsBtn != nil {
		right.Add(settingsBtn)
	}
	if cfg.ShowCursorAI && cfg.OnCursorAI != nil {
		label := "Cursor AI"
		if cfg.CursorAIOpen {
			label = "Hide Cursor"
		}
		cursorBtn := TipButtonLabeled(label, theme.MailSendIcon(), cfg.OnCursorAI)
		cursorBtn.Importance = widget.LowImportance
		cursorBtn.SetToolTip("AI troubleshooting pane — gather scrollback, ask Cursor, send commands to SSH")
		right.Add(cursorBtn)
	}

	topInner := container.NewBorder(nil, nil, launch, right, chromeSpacer())
	top := container.NewPadded(topInner)
	return container.New(&chromeMinHeight{h: toolbarH}, top)
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
	b.Importance = widget.LowImportance
	if tip != "" {
		b.SetToolTip(tip)
	}
	return b
}

func chromeSpacer() fyne.CanvasObject {
	r := canvas.NewRectangle(theme.Color(theme.ColorNameBackground))
	r.SetMinSize(fyne.NewSize(1, 1))
	return r
}

// OpsDock is the persistent bottom strip (button bar, send row, status).
type OpsDock struct {
	root      fyne.CanvasObject
	buttonRow fyne.CanvasObject
	sendRow   fyne.CanvasObject
	workCtx   *widget.Label
}

func (d *OpsDock) Content() fyne.CanvasObject {
	if d == nil {
		return nil
	}
	return d.root
}

// SetWorkContext updates the incident label in the bottom status strip.
func (c *AppChrome) SetWorkContext(text string) {
	if c == nil || c.opsDock == nil || c.opsDock.workCtx == nil {
		return
	}
	c.opsDock.workCtx.SetText(text)
	if text == "" {
		c.opsDock.workCtx.Hide()
	} else {
		c.opsDock.workCtx.Show()
	}
}

func (d *OpsDock) SetConnected(connected bool) {
	if d == nil {
		return
	}
	if d.buttonRow != nil {
		d.buttonRow.Show()
		if !connected {
			d.buttonRow.Hide()
		}
	}
	if d.sendRow != nil {
		d.sendRow.Show()
		if !connected {
			d.sendRow.Hide()
		}
	}
	if d.root != nil {
		d.root.Refresh()
	}
}

func newOpsDock(cfg AppChromeConfig) *OpsDock {
	buttonRow := newButtonBarRow(cfg.BarButtons, cfg.OnBarAction, cfg.OnBarEdit)
	// Freeform send uses a light Entry that looked like a white streak over the
	// terminal; keep macros only here. Chat/send remains available via Scripts.
	status := cfg.Status
	if status == nil {
		status = widget.NewLabel("")
		status.Importance = widget.LowImportance
	}
	left := container.NewHBox()
	workCtx := cfg.WorkContext
	if workCtx == nil {
		workCtx = widget.NewLabel("")
		workCtx.Hide()
	}
	workCtx.Importance = widget.MediumImportance
	left.Add(workCtx)
	left.Add(status)
	statusBar := container.NewBorder(nil, nil, left, nil, chromeSpacer())

	root := container.NewVBox(buttonRow, statusBar)
	d := &OpsDock{root: root, buttonRow: buttonRow, sendRow: nil, workCtx: workCtx}
	if cfg.OnSendChat != nil {
		sendRow := newCommandRow(cfg.Customers, cfg.OnAdhocConnect, cfg.OnSendChat)
		d.sendRow = sendRow
		root = container.NewVBox(buttonRow, sendRow, statusBar)
		d.root = root
	}
	d.SetConnected(cfg.ShowSendDock)
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
