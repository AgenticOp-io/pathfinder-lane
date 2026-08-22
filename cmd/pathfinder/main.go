// cmd/pathfinder/main.go
// The application shell: one window, three kinds of applet, tabs.
//
// This is the first thing in the tree that is an application rather than a
// harness. cmd/pfterm, cmd/pfconnect, cmd/crawlui and cmd/captureui stay --
// they are the fastest way to reproduce a bug in one view with the shell out of
// the way, and they mean the shell can be broken without breaking the three
// things that already work.
//
// What lives here and not in internal/ui: connecting. The shell hosts applets
// and knows nothing about dialers, vaults or crawlers; this file assembles
// them. That is the guard against TetherSSH's session manager, which reached
// 2,000 lines by becoming the place connections and dialogs both ended up.
//
//	go run ./cmd/pathfinder
//	go run ./cmd/pathfinder -vault ~/.pathfinderssh/vault.json -store ~/captures
//	go run ./cmd/pathfinder -vault ~/.pathfinderssh/vault.json -domain lab.local
//
// # What to check once it is up
//
//   - open a terminal, a crawl and a capture, all three at once; the crawl
//     table keeps updating while a terminal tab is in front
//   - detach any of them: same content, own window, still live. Its close box
//     re-docks it; the X in its own bar ends it
//   - two terminals with different font sizes, opened in either order, both
//     keep their own size -- this is what ui.ThemedAt is for
//   - switch tabs and type: the terminal takes focus back
//   - click a reached device in the crawl table: a session dialog opens
//     prefilled with that device
//   - close the window with a crawl running: it cancels rather than hanging
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"golang.org/x/crypto/ssh"

	"github.com/scottpeterman/pathfinderssh/internal/appinstall"
	"github.com/scottpeterman/pathfinderssh/internal/auvik"
	"github.com/scottpeterman/pathfinderssh/internal/buttons"
	"github.com/scottpeterman/pathfinderssh/internal/capture"
	"github.com/scottpeterman/pathfinderssh/internal/capturedial"
	"github.com/scottpeterman/pathfinderssh/internal/capturerun"
	"github.com/scottpeterman/pathfinderssh/internal/crawldial"
	"github.com/scottpeterman/pathfinderssh/internal/crawler"
	"github.com/scottpeterman/pathfinderssh/internal/crawlrun"
	"github.com/scottpeterman/pathfinderssh/internal/crtimport"
	"github.com/scottpeterman/pathfinderssh/internal/cursorapi"
	"github.com/scottpeterman/pathfinderssh/internal/evidence"
	helpdoc "github.com/scottpeterman/pathfinderssh/internal/help"
	"github.com/scottpeterman/pathfinderssh/internal/itglue"
	"github.com/scottpeterman/pathfinderssh/internal/mapweb"
	"github.com/scottpeterman/pathfinderssh/internal/mspauth"
	"github.com/scottpeterman/pathfinderssh/internal/policy"
	"github.com/scottpeterman/pathfinderssh/internal/product"
	"github.com/scottpeterman/pathfinderssh/internal/psasync"
	"github.com/scottpeterman/pathfinderssh/internal/pyrun"
	"github.com/scottpeterman/pathfinderssh/internal/recent"
	"github.com/scottpeterman/pathfinderssh/internal/scripts"
	"github.com/scottpeterman/pathfinderssh/internal/serialx"
	"github.com/scottpeterman/pathfinderssh/internal/sessiondial"
	"github.com/scottpeterman/pathfinderssh/internal/sessions"
	"github.com/scottpeterman/pathfinderssh/internal/storesearch"
	"github.com/scottpeterman/pathfinderssh/internal/term"
	"github.com/scottpeterman/pathfinderssh/internal/topo"
	"github.com/scottpeterman/pathfinderssh/internal/ui"
	"github.com/scottpeterman/pathfinderssh/internal/vault"
	"github.com/scottpeterman/pathfinderssh/internal/vaultcli"
	"github.com/scottpeterman/pathfinderssh/internal/workcontext"
)

func main() {
	var (
		vaultPath    = flag.String("vault", "", "vault file; defaults to the standard path, unlocked from the keyring if it can be")
		sessionsPath = flag.String("sessions", "", "session tree file; defaults to sessions.yaml beside the vault")
		storePath    = flag.String("store", "", "capture store root for the capture applet")
		appTheme     = flag.String("app", "dark", "application chrome: dark|light")
		domain       = flag.String("domain", "", "default domain suffix for crawl and capture")
		verbose      = flag.Bool("v", false, "log applet progress to stderr")
		doInstall   = flag.Bool("install", false, "copy into LocalAppData\\PathfinderSSH-MSP, create shortcuts, exit")
		doInstallGUI = flag.Bool("install-gui", false, "graphical install wizard (solo, Microsoft 365, or Google)")
		doUninstall = flag.Bool("uninstall", false, "remove LocalAppData\\PathfinderSSH-MSP install and shortcuts, exit")
		doEnroll    = flag.Bool("enroll", false, "open setup wizard (OAuth app registration), then exit if combined with -install")
		doSetup     = flag.String("setup", "", "access mode: solo (no cloud login), o365, google")
		noInstall   = flag.Bool("no-install", false, "run from this folder without copying to AppData")
		logoPath    = flag.String("logo", "", "override About/window logo PNG (also PATHFINDERSSH_LOGO or ~/.pathfinderssh/logo.png)")
	)
	flag.Parse()

	log.SetFlags(log.Ltime | log.Lmicroseconds)

	if strings.TrimSpace(*logoPath) != "" {
		ui.SetLogoPath(*logoPath)
	}

	if *doUninstall {
		if err := appinstall.Uninstall(); err != nil {
			fmt.Fprintf(os.Stderr, "uninstall: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Removed", appinstall.Root())
		return
	}

	setupMode := strings.TrimSpace(*doSetup)

	if *doInstallGUI {
		runInstallGUI(setupMode, version)
		return
	}

	if err := maybeSelfInstall(*doInstall, *noInstall); err != nil {
		log.Printf("install: %v", err)
		if *doInstall {
			os.Exit(1)
		}
	}

	if setupMode != "" && mspauth.HeadlessSetup(setupMode) {
		if err := mspauth.SaveSoloSetup(ui.GetAppHome()); err != nil {
			fmt.Fprintf(os.Stderr, "setup: %v\n", err)
			os.Exit(1)
		}
		if *doInstall || !*doEnroll {
			fmt.Println("Solo mode — no Microsoft/Google sign-in required.")
			if *doInstall {
				fmt.Println("Installed to", appinstall.ExePath())
			}
			if !*doEnroll {
				return
			}
		}
	}

	if *doInstall && !*doEnroll && setupMode == "" {
		fmt.Println("Installed to", appinstall.ExePath())
		return
	}
	if *doInstall && !*doEnroll && setupMode != "" && !mspauth.HeadlessSetup(setupMode) {
		fmt.Println("Installed to", appinstall.ExePath())
		fmt.Println("Run Pathfinder to finish", setupMode, "sign-in setup.")
		return
	}

	// One UI process only. A second Start() (or double-click while already
	// open) previously left two windows racing writes to settings.json.
	if ok, release := appinstall.TrySingleton(); !ok {
		fmt.Fprintln(os.Stderr, "PathfinderSSH MSP is already running.")
		os.Exit(0)
	} else {
		defer release()
	}

	// Settings come off disk before anything is built: the chrome variant
	// is read by app.New's theme and by every widget after it, so a
	// settings load that happened later would repaint a window that had
	// already been drawn in the other colour.
	//
	// A load failure is not fatal -- LoadSettings hands back working
	// defaults with it -- but it is not silent either, because the next
	// Save overwrites whatever could not be read. It is reported once the
	// window exists to report it in.
	settingsPath := ui.SettingsPath()
	base, settingsErr := ui.LoadSettings(settingsPath)

	// The flag wins over the file, but only when it was actually typed.
	// Its default is "dark", so an unconditional assignment would mean a
	// saved light theme lost every launch to a flag nobody passed.
	if flagWasSet("app") {
		base.AppTheme = ui.AppVariant(*appTheme)
	}
	ui.SetSettings(base)
	base = ui.CurrentSettings()

	// Layer ~/.pathfinderssh/themes over the built-ins and the embedded
	// pack. Nothing called this: theme_registry's init registers the
	// shipped themes and its comment says user themes arrive "at
	// LoadUserThemes() in main", but no main did, so the themes directory
	// has been read by nothing. It has to happen before the first theme
	// lookup, and it is only file reading -- no widget, no app.
	ui.LoadUserThemes()

	// app.New() before ANY widget. Fyne resolves the theme and driver
	// through the current app; building a widget first nil-derefs inside
	// Button.CreateRenderer and the panic names a layout function.
	a := app.New()
	ui.ApplyAppTheme(a, base.AppVariant())
	if icon := ui.AppIcon(); icon != nil {
		a.SetIcon(icon)
	}
	w := a.NewWindow("PathfinderSSH MSP")
	w.Resize(fyne.NewSize(1280, 820))

	h := &host{
		app:          a,
		win:          w,
		base:         base,
		settingsPath: settingsPath,
		vaultPath:    ui.ExpandHome(*vaultPath),
		verbose:      *verbose,
		forwards:     ui.NewForwardHub(),
		guardedSend:  true,
	}
	if h.vaultPath == "" {
		h.vaultPath = vaultcli.DefaultPath()
	}
	h.shell = ui.NewShell(a, w)
	h.initWorkContext()

	// Seed the dialogs so the first launch is not an empty form.
	h.lastCrawl.Params = crawlrun.Defaults()
	h.lastCrawl.Verbose = *verbose
	h.lastCapture.Params = capturerun.Defaults()
	h.lastCapture.Params.StorePath = ui.ExpandHome(*storePath)
	h.lastCapture.Verbose = *verbose
	if d := strings.TrimSpace(*domain); d != "" {
		h.lastCrawl.Params.Domains = []string{d}
		h.lastCapture.Params.Domains = []string{d}
	}
	h.node = sessions.Defaults()

	h.buildSessionTree(ui.ExpandHome(*sessionsPath))
	h.installShortcuts()

	h.mspEnrollOnStart = *doEnroll
	if p, ok := mspauth.ParseSetupMode(setupMode); ok && p.RequiresCloudLogin() {
		h.mspSetupPreset = p
		h.mspEnrollOnStart = true
	}

	// Try the keyring and the environment before the window is up, and
	// leave the vault LOCKED if neither has it. The old code called
	// vaultcli.Open here, which falls through to a terminal read -- fine
	// for a CLI, and in a window it blocks on a prompt nobody can see.
	h.unlockQuiet()
	ui.SetHelp(ui.HelpConfig{Version: version})

	w.SetContent(ui.WithTooltips(h.shell.Content(), w.Canvas()))
	w.SetMaster()
	w.SetCloseIntercept(func() {
		// Already tearing down — ignore further close-box events.
		if h.shuttingDown {
			return
		}
		// The close box stays live while a dialog is up, so without
		// this a second click stacks a second confirmation -- and
		// answering one of them then quits out from under the other,
		// which is the opposite of what a confirmation is for.
		if h.askingQuit {
			return
		}
		msg, ask := ui.ShutdownPrompt(h.shell.Busy())
		if !ask {
			h.shutdown()
			return
		}
		h.askingQuit = true
		d := dialog.NewConfirm("Quit PathfinderSSH MSP?", msg, func(ok bool) {
			if !ok {
				h.askingQuit = false
				return
			}
			// Leave askingQuit set until shutdown flips shuttingDown so a
			// second close-box click cannot open another dialog in between.
			h.shutdown()
		}, w)
		d.SetConfirmText("Quit")
		d.SetConfirmImportance(widget.DangerImportance)
		d.SetDismissText("Stay open")
		d.Show()
	})
	// Report an unreadable settings file now that there is a window to
	// report it in. The application is already running on the defaults;
	// this exists so that the next Save -- which will overwrite the file
	// that could not be read -- is not a surprise.
	if settingsErr != nil {
		log.Printf("[settings] %v", settingsErr)
		dialog.ShowError(settingsErr, w)
	}

	// The first-run vault warning goes here for the same reason: there has
	// to be a window for it to appear in. It is silent on every machine
	// that already has a vault.
	h.runMSPAccessSetup(func() {
		h.promptVaultUnlockIfNeeded()
		h.offerVaultCreate()
		h.offerFirstRunSetup()
	})

	h.auvikTunnels = auvik.NewTunnelManager(h.base.AuvikTunnelPath)
	h.startAuvikPeriodicSync()

	// Immediately before ShowAndRun, like an applet's Start: the watchdog
	// hands work to fyne.Do and needs a running driver.
	h.shell.StartFocusWatch()
	w.ShowAndRun()
}

// host owns everything the shell deliberately does not: the vault, the dialers,
// and the last values each dialog was filled with.
type host struct {
	app  fyne.App
	win  fyne.Window
	base ui.Settings

	shell *ui.Shell
	appChrome *ui.AppChrome

	// vault is the one unlocked vault for this app session. Every applet
	// shares it: the session dialog's credential picker, and both run
	// builders, which are handed the open vault rather than a path so
	// neither can decide to open one on its own.
	vault    *vault.Vault

	creds  []string
	lookup sessiondial.Lookup

	// defaultCred is the vault's default credential name, or "" when there
	// is none. Held so a dialog can SAY which credential a blank field
	// resolves to without being handed the credential itself.
	defaultCred string
	vaultPath   string
	verbose     bool

	// settingsPath is the file the settings dialog reads and writes. It is
	// held rather than recomputed so a run can only ever have one answer
	// for where its settings live.
	settingsPath string

	// tree is the saved inventory, docked down the left. The host owns the
	// FILE; the widget owns the display and hands back a tree to save.
	tree         *ui.SessionTree
	sessionsPath string

	node        sessions.Node
	lastCrawl   ui.CrawlLaunch
	lastCapture ui.CaptureLaunch
	lastSearch  ui.SearchLaunch

	// maps is the loopback viewer, started on the first map opened and
	// kept for the life of the app: one port, one token, and a browser tab
	// that keeps working when the next map is loaded into it.
	maps   *mapweb.Server
	mapDir string
	// mapCustomer is the last customer selected in the map picker / crawl map path.
	mapCustomer string

	// askingQuit is true while the quit confirmation is on screen. UI
	// goroutine only, like everything else the close intercept touches,
	// so a plain bool is the honest type -- an atomic here would suggest
	// a second writer that does not exist.
	askingQuit bool

	// shuttingDown is set once quit is committed. Prevents the close
	// intercept from re-entering when win.Close() would otherwise fire it
	// again (Fyne still honors SetCloseIntercept on programmatic Close).
	shuttingDown bool

	// scriptCancel stops an in-flight YAML script between steps.
	scriptCancel atomic.Pointer[context.CancelFunc]

	// forwards tracks SSH port forwards so session close / quit can stop them.
	forwards *ui.ForwardHub

	// guardedSend, when true, confirms before Send to all / customer.
	guardedSend bool

	// recorder captures keystrokes into scripts.yaml when non-nil and active.
	recorder *scripts.Recorder

	// Auvik periodic sync and tunnel helper.
	auvikSyncCancel context.CancelFunc
	auvikTunnels    *auvik.TunnelManager

	// Cursor AI side pane (Troubleshoot addon).
	cursorPaneVisible bool
	cursorPane        fyne.CanvasObject

	// MSP org enrollment + engineer sign-in (mspauth hooks).
	mspAuth       *mspauth.Authenticator
	mspEnrollment mspauth.Enrollment
	mspSession    mspauth.UserSession
	mspEnrollOnStart bool
	mspSetupPreset   mspauth.Provider

	// Engineer work context (augments PagerDuty / on-call systems).
	workCtx          workcontext.Context
	workCtxPath      string
	workContextLabel *widget.Label
}

// shutdown ends the application: applets first, then the map server, then the
// window.
//
// Split out of the close intercept because there are now two ways in -- the
// straight-through case and the confirmed one -- and the failure mode if they
// drift is that quitting past a warning skips a teardown that quitting
// without one performs.
func (h *host) shutdown() {
	if h.shuttingDown {
		return
	}
	h.shuttingDown = true
	h.askingQuit = false

	// Must clear the intercept before Close. Otherwise Close re-invokes the
	// intercept: Busy() still sees live sessions (OnClose is async), the quit
	// dialog comes back, and the app never leaves ShowAndRun.
	h.win.SetCloseIntercept(nil)

	h.shell.StopFocusWatch()

	// Tear the applets down before the window goes. Closing a transport
	// can block, so each instance's OnClose already runs on its own
	// goroutine -- this just makes sure they all start.
	h.shell.CloseAll()
	if h.forwards != nil {
		h.forwards.StopAll()
	}
	if h.auvikTunnels != nil {
		h.auvikTunnels.StopAll()
	}
	if h.auvikSyncCancel != nil {
		h.auvikSyncCancel()
	}
	// Stop answering the browser. A map left open in a tab after the
	// application exits should fail honestly rather than look live until
	// something is clicked.
	if h.maps != nil {
		_ = h.maps.Close()
	}

	// Last-resort exit: GLFW/Fyne has been observed to never return from
	// Quit when a serial/SSH close is wedged in the driver. Prefer a clean
	// Quit; force the process out if the run loop stays stuck.
	go func() {
		time.Sleep(2 * time.Second)
		os.Exit(0)
	}()

	h.win.Close()
	// Ensure the Fyne run loop exits even if a detached window or driver
	// quirk left the master window in a half-closed state.
	h.app.Quit()
}

func (h *host) logf() func(string, ...any) { return h.logfIf(false) }

// logfIf is logf with a per-run override.
//
// Both launch dialogs carry a "Log progress to stderr" checkbox and neither
// run path was reading it: every logf came from h.verbose, which is the -v
// flag and nothing else. So the checkbox did nothing, and the one thing it
// would have shown — the credential-resolution lines naming which vault entry
// was offered to which device — was unreachable from the GUI at all.
func (h *host) logfIf(runVerbose bool) func(string, ...any) {
	if !h.verbose && !runVerbose {
		return func(string, ...any) {}
	}
	return func(format string, args ...any) {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
}

// --- terminal --------------------------------------------------------------

// launchTerminal opens the session dialog and connects what comes out of it.
func (h *host) launchTerminal(start sessions.Node) {
	h.launchTerminalTitled("New session", "", start)
}

// launchTerminalTitled is the same dialog under a caller-chosen heading. A
// node picked out of the inventory is not a new session, and a dialog that
// says it is reads like the click did something other than it did.
//
// folder is the inventory path when the node came from the tree (so Save can
// write it back). Empty means ad-hoc / Quick Connect — Connect only.
func (h *host) launchTerminalTitled(title, folder string, start sessions.Node) {
	var d dialog.Dialog
	var form *ui.SessionForm
	oldLabel := start.Normalize().Label()

	opts := ui.SessionFormOptions{
		Node:              start,
		Credentials:       h.creds,
		DefaultCredential: h.defaultCred,
		VaultLocked:       h.lookup == nil,
		ListSerialPorts:   listPorts,
		ShowConnect:       true,
		OnConnect: func(n sessions.Node) {
			h.node = n
			d.Hide()
			h.connectSavingPassword(folder, oldLabel, n, func(saved sessions.Node) {
				if folder != "" {
					h.applyInventoryNode(folder, oldLabel, saved)
				}
			})
		},
	}
	if folder != "" {
		opts.ShowSave = true
		opts.OnSave = func(n sessions.Node) {
			// Save must NOT close the dialog and must NOT open a terminal.
			// Closing here left people with no session tab and looking like
			// "the terminal does not work". Persist, stay open, Connect opens.
			h.applyInventoryNode(folder, oldLabel, n)
			oldLabel = n.Normalize().Label()
			if form != nil {
				form.SetStatus("Saved. Click Connect to open the terminal.")
			}
		}
	}

	form = ui.NewSessionForm(opts)

	d = dialog.NewCustom(title, "Cancel", form.Content(), h.win)
	d.Resize(fyne.NewSize(760, 660))
	d.Show()
	// This dialog's only button is Cancel -- Connect lives inside the form
	// -- so Return goes to the form's own action rather than to the
	// dialog's.
	ui.EnterConfirms(h.win, form.Content(), form.Connect)
}

// applyInventoryNode writes an edited node back into the session tree.
func (h *host) applyInventoryNode(folder, oldLabel string, n sessions.Node) {
	if h.tree == nil || folder == "" {
		return
	}
	tr := h.tree.Tree()
	if err := tr.Replace(folder, oldLabel, n); err != nil {
		dialog.ShowError(err, h.win)
		return
	}
	h.tree.SetTree(tr)
	// Synchronous: credential: links from password-save must hit disk before
	// the next Connect, and an async save raced a wipe of that field.
	if err := sessions.SaveFile(h.sessionsPath, tr); err != nil {
		dialog.ShowError(fmt.Errorf("could not save %s: %w", h.sessionsPath, err), h.win)
	}
}

// --- session tree ----------------------------------------------------------

// buildSessionTree loads the inventory and docks it beside the tabs.
//
// A file that will not parse is reported and then ignored: starting with an
// empty tree beats refusing to start, and the file is left untouched so it can
// be fixed in an editor rather than overwritten by whatever this session does
// next.
func (h *host) buildSessionTree(path string) {
	h.sessionsPath = path
	if h.sessionsPath == "" {
		// Beside the vault, deliberately, so the legacy-directory fallback
		// vaultcli already does is inherited rather than reimplemented.
		h.sessionsPath = filepath.Join(filepath.Dir(h.vaultPath), "sessions.yaml")
	}

	tr, err := sessions.LoadFile(h.sessionsPath)
	if err != nil {
		log.Printf("[sessions] %s: %v", h.sessionsPath, err)
	}
	if changed, mErr := (&tr).EnsureMSPLayout(); mErr != nil {
		log.Printf("[sessions] MSP layout: %v", mErr)
	} else if changed {
		if sErr := sessions.SaveFile(h.sessionsPath, tr); sErr != nil {
			log.Printf("[sessions] save after MSP roots: %v", sErr)
		}
	}

	h.tree = ui.NewSessionTree(ui.SessionTreeOptions{
		Window: h.win,
		OnActivate: func(folder string, n sessions.Node) {
			go func() {
				if _, err := recent.Touch(recent.Path(ui.GetAppHome()), folder, n.Label(), n.Host); err != nil {
					log.Printf("[recent] %v", err)
				}
			}()
			// Double-click with a saved password (vault credential) skips the
			// form and dials immediately — SecureCRT-style auto-login.
			if h.canAutoLogin(n) {
				label := n.Normalize().Label()
				h.connect(folder, label, n, func(saved sessions.Node) {
					h.applyInventoryNode(folder, label, saved)
				})
				return
			}
			h.launchTerminalTitled(n.Label(), folder, n)
		},
		OnNew: func(folder string, apply func(sessions.Node)) {
			h.editSession("New session in "+folder, folder, sessions.Defaults(), apply)
		},
		OnEdit: func(folder string, n sessions.Node, apply func(sessions.Node)) {
			h.editSession("Edit "+n.Label(), folder, n, apply)
		},
		OnChanged: h.saveTree,
	})
	h.tree.SetTree(tr)
	h.shell.SetSide(h.tree.Content(), 0.25)
	h.buildChrome()
	if h.base.TroubleshootAddon {
		h.cursorPaneVisible = true
		h.applyCursorPane()
	}


	if err != nil {
		// After the window has content, or the error has nowhere to appear.
		fyne.Do(func() {
			dialog.ShowError(fmt.Errorf("could not read %s: %w", h.sessionsPath, err), h.win)
		})
	}
}

// editSession is the inventory's session dialog: Save writes it back to the
// tree; Connect dials and writes the tree after a successful dial (so a
// password-save credential: link is not wiped by a pre-connect write).
func (h *host) editSession(title, folder string, start sessions.Node, apply func(sessions.Node)) {
	var d dialog.Dialog
	oldLabel := start.Normalize().Label()

	form := ui.NewSessionForm(ui.SessionFormOptions{
		Node:              start,
		Credentials:       h.creds,
		DefaultCredential: h.defaultCred,
		VaultLocked:       h.lookup == nil,
		ListSerialPorts:   listPorts,
		ShowSave:          true,
		ShowConnect:       true,
		OnSave: func(n sessions.Node) {
			d.Hide()
			apply(n)
		},
		OnConnect: func(n sessions.Node) {
			d.Hide()
			h.connectSavingPassword(folder, oldLabel, n, apply)
		},
	})

	d = dialog.NewCustom(title, "Cancel", form.Content(), h.win)
	d.Resize(fyne.NewSize(760, 660))
	d.Show()
	ui.EnterConfirms(h.win, form.Content(), form.Connect)
}

// folderFor finds the folder holding this node, reporting false when the answer
// is not exactly one.
//
// Both failure directions matter and neither is an error: a session dialled ad
// hoc is in no folder, and a device name that appears in two site folders has
// no single right answer. Writing to a guess would edit the wrong device's
// session file, which is worse than not offering to write at all.
func (h *host) folderFor(n sessions.Node) (string, bool) {
	if h.tree == nil {
		return "", false
	}
	label := n.Normalize().Label()
	if label == "" {
		return "", false
	}
	target := n.Target()
	found := ""
	ambiguous := false
	var walk func(prefix string, folders []sessions.Folder)
	walk = func(prefix string, folders []sessions.Folder) {
		for _, f := range folders {
			path := f.Name
			if prefix != "" {
				path = sessions.JoinPath(prefix, f.Name)
			}
			for _, s := range f.Sessions {
				if s.Label() != label || s.Target() != target {
					continue
				}
				if found != "" {
					ambiguous = true
					return
				}
				found = path
			}
			walk(path, f.Folders)
			if ambiguous {
				return
			}
		}
	}
	walk("", h.tree.Tree().Folders)
	if ambiguous || found == "" {
		return "", false
	}
	return found, true
}

// rememberPastePacing persists a pacing pair chosen in the paste confirmation.
//
// Zero cannot be stored as-is for either field: on a node zero means "inherit
// the application setting", and the application default for the line delay is
// 25ms. So an operator who picked "No delay" and ticked remember would get 25ms
// back on the next connect — the setting silently not taking, which is the
// worst outcome of the three. Negative is the established spelling for an
// explicit off (see Node.PasteLineDelayMs), and ConsoleBaud now reads the same
// way.
func (h *host) rememberPastePacing(folder string, n sessions.Node, delayMs, baud int) {
	if delayMs <= 0 {
		delayMs = -1
	}
	if baud <= 0 {
		baud = -1
	}

	tr := h.tree.Tree()
	f, err := tr.FolderAt(folder)
	if err != nil {
		dialog.ShowError(err, h.win)
		return
	}
	label := n.Normalize().Label()
	j := f.SessionIndex(label)
	if j < 0 {
		dialog.ShowError(fmt.Errorf("no session called %q in %q", label, folder), h.win)
		return
	}

	// Read the node back out of the tree rather than editing the copy this
	// tab was dialled from. That copy is a snapshot taken at connect time,
	// and writing it back would revert anything edited in the session form
	// since -- an unrelated change lost as a side effect of a paste.
	upd := f.Sessions[j]
	upd.PasteLineDelayMs = delayMs
	upd.ConsoleBaud = baud
	if err := tr.Replace(folder, label, upd); err != nil {
		dialog.ShowError(err, h.win)
		return
	}
	h.tree.SetTree(tr)
	h.saveTree(tr)
	log.Printf("[paste] remembered pacing for %s: delay=%d baud=%d", label, delayMs, baud)
}

// saveTree writes the inventory off the UI thread. A large nested SecureCRT
// tree marshaled synchronously froze clicks for hundreds of milliseconds.
func (h *host) saveTree(tr sessions.Tree) {
	path := h.sessionsPath
	go func(tr sessions.Tree) {
		if err := sessions.SaveFile(path, tr); err != nil {
			fyne.Do(func() {
				dialog.ShowError(fmt.Errorf("could not save %s: %w", path, err), h.win)
			})
		}
	}(tr)
}

// --- the File menu ---------------------------------------------------------

// buildMenu puts import and export on a menu bar rather than on the tree panel.
//
// They belong here for the same reason the tree widget has no save button: the
// host owns the FILE and the widget owns the display. They are also the wrong
// shape for that panel — a quarter-width column of icons is for the things done
// constantly, and importing an estate is done once and then not again for
// months.
// tabsButton is the toolbar entry for switching and closing open tabs.
//
// When nothing is open the menu still shows a clear “No tabs open” line —
// an all-disabled Close menu looked blank on Windows. With sessions open it
// lists them so you can jump to one, then the close actions underneath.
// tabsButton kept for tests; ribbon uses showTabsMenu.
func (h *host) tabsButton() *ui.TipButton {
	var btn *ui.TipButton
	btn = ui.TipButtonLabeled("Tabs", theme.ListIcon(), func() {
		h.showTabsMenu(btn)
	})
	btn.Importance = widget.LowImportance
	btn.SetToolTip("Switch or close open tabs")
	return btn
}

// confirmClose asks before ending more than one session at once.
//
// One tab is the person closing the thing in front of them; several is a
// command whose effect they cannot see -- a crawl mid-run and a terminal
// sitting at a config prompt look identical from a menu. Closing exactly one
// skips the question, because a confirmation on every close is a confirmation
// nobody reads.
func (h *host) confirmClose(title string, n int, do func()) {
	if n <= 0 {
		return
	}
	if n == 1 {
		do()
		return
	}
	msg := fmt.Sprintf("Close %d open sessions? Anything still running will be stopped.", n)
	dialog.ShowConfirm(title, msg, func(ok bool) {
		if ok {
			do()
		}
	}, h.win)
}

func (h *host) buildChrome() {
	if h.shell == nil {
		return
	}
	if h.appChrome != nil {
		h.appChrome.SetConnected(h.hasConnectedTerminal())
		h.refreshVault()
		return
	}

	path := buttons.Path(ui.GetAppHome())
	btnFile, err := buttons.Load(path)
	if err != nil {
		btnFile = buttons.Defaults()
	}
	if len(btnFile.Buttons) > 0 {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			_ = buttons.Save(path, btnFile)
		}
	}

	h.appChrome = ui.BuildAppChrome(ui.AppChromeConfig{
		OnQuickConnect: func() { h.launchTerminal(h.node) },
		OnCrawl:        h.launchCrawl,
		OnCapture:      h.launchCapture,
		OnMap:          h.launchMap,
		OnSearch:       h.launchSearch,
		ScriptsMenu:    h.showScriptsMenu,
		TabsMenu:       h.showTabsMenu,
		OnSettings:     h.showSettings,
		Customers:      h.customerNames(),
		ShowSendDock:   h.hasConnectedTerminal(),
		OnSendChat: func(text string, mode ui.ChatSendMode, customer string) {
			h.sendChat(text, mode, customer)
		},
		BarButtons: btnFile.Buttons,
		OnBarAction: func(b buttons.Button, all bool) { h.barButtonAction(b, all) },
		OnBarEdit: func() {
			scriptPath := scripts.Path(ui.GetAppHome())
			dialog.ShowInformation("Button bar",
				"Edit "+path+" and restart Pathfinder.\n\n"+
					"Send a command:\n  label: Show run\n  send: show running-config\\n\n"+
					"Run a script (from "+scriptPath+"):\n  label: Backup\n  script: My Script Name\n\n"+
					"Optional: scope: all  (or check All tabs when clicking)",
				h.win)
		},
		Status:         h.shell.SummaryLabel(),
		WorkContext:    h.workContextLabel,
		ShowCursorAI:   h.base.TroubleshootAddon,
		CursorAIOpen:   h.cursorPaneVisible,
		OnCursorAI:     h.toggleCursorPane,
	})
	h.shell.SetTopChrome(h.appChrome.Top())
	h.shell.SetBottom(h.appChrome.Bottom())
	h.refreshVault()
}

func (h *host) refreshOpsChrome() {
	if h.shell == nil {
		return
	}
	h.buildChrome()
	// Do NOT RefocusCurrentTerminal here. This runs from the connect state
	// handler while the Connecting dialog may still be an overlay; focusing
	// under it makes the new SSH tab permanently ignore the keyboard.
	// Shell settle() + the focus watchdog reclaim focus once overlays clear.
}

func (h *host) hasConnectedTerminal() bool {
	if h.shell == nil {
		return false
	}
	for _, inst := range h.shell.Instances() {
		if inst == nil {
			continue
		}
		ta, ok := inst.Applet().(*termApplet)
		if ok && ta.sess != nil && ta.sess.Connected() {
			return true
		}
	}
	return false
}

// buildRibbon is replaced by buildChrome.
func (h *host) buildRibbon() { h.buildChrome() }

func (h *host) showTabsMenu(anchor fyne.CanvasObject) {
	open := h.shell.TabCount()
	current := h.shell.Current()
	items := make([]*fyne.MenuItem, 0, open+5)

	if open == 0 {
		none := fyne.NewMenuItem("No tabs open", nil)
		none.Disabled = true
		items = append(items, none)
	} else {
		for _, inst := range h.shell.Instances() {
			inst := inst
			label := inst.Title()
			if label == "" {
				label = "(untitled)"
			}
			if current != nil && inst == current {
				label = "• " + label
			}
			items = append(items, fyne.NewMenuItem(label, func() {
				h.shell.Activate(inst)
			}))
		}
		items = append(items, fyne.NewMenuItemSeparator())
	}

	closeTab := fyne.NewMenuItem("Close Tab", h.shell.CloseCurrent)
	closeTab.Disabled = current == nil

	closeOthers := fyne.NewMenuItem("Close Other Tabs", func() {
		h.confirmClose("Close other tabs?", open-1, func() {
			h.shell.CloseOthers(current)
		})
	})
	closeOthers.Disabled = current == nil || open <= 1

	closeAll := fyne.NewMenuItem("Close All Tabs", func() {
		h.confirmClose("Close all tabs?", open, h.shell.CloseAll)
	})
	closeAll.Disabled = open == 0

	items = append(items, closeTab, closeOthers, closeAll)
	items = append(items, fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Tile / untile terminals", func() { h.shell.ToggleTileLayout() }),
	)
	menu := fyne.NewMenu("", items...)
	widget.ShowPopUpMenuAtRelativePosition(
		menu, h.win.Canvas(), fyne.NewPos(0, anchor.Size().Height), anchor)
}

func (h *host) showScriptsMenu(anchor fyne.CanvasObject) {
	f := h.loadScripts()
	menuItems := make([]*fyne.MenuItem, 0, len(f.Scripts)+8)
	for _, sc := range f.Scripts {
		sc := sc
		menuItems = append(menuItems, fyne.NewMenuItem(sc.Name, func() {
			h.runNamedScript(sc.Name)
		}))
	}
	if len(f.Scripts) > 0 {
		menuItems = append(menuItems, fyne.NewMenuItemSeparator())
	}
	menuItems = append(menuItems,
		fyne.NewMenuItem("Run script…", h.runScriptDialog),
		fyne.NewMenuItem("Script editor…", h.openScriptEditor),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Start recording…", h.startScriptRecording),
		fyne.NewMenuItem("Stop recording…", h.stopScriptRecording),
		fyne.NewMenuItem("Run Python script…", h.runPythonScript),
	)
	m := fyne.NewMenu("", menuItems...)
	widget.ShowPopUpMenuAtRelativePosition(m, h.win.Canvas(), fyne.NewPos(0, anchor.Size().Height), anchor)
}

func (h *host) showSessionMenu(anchor fyne.CanvasObject, sess *ui.Session, inst *ui.Instance, label string) {
	items := []*fyne.MenuItem{
		fyne.NewMenuItem("Transfer files (SFTP)…", func() {
			h.openSFTPTab(sess, "SFTP — "+label)
		}),
		fyne.NewMenuItem("Port forwards…", func() {
			client, ok := sess.SSHClient()
			if !ok || client == nil {
				dialog.ShowInformation("Port forwards", "Port forwards require an SSH session.", h.win)
				return
			}
			ui.ShowPortForwardDialog(h.win, "Port forwards — "+label, client, h.forwards)
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Run script into this session…", func() {
			h.shell.Activate(inst)
			h.runScriptDialog()
		}),
		fyne.NewMenuItemSeparator(),
		fyne.NewMenuItem("Start / stop capture…", func() {
			_, msg := sess.ToggleLogging()
			if msg != "" {
				dialog.ShowInformation("Session Capture", msg, h.win)
			}
		}),
		fyne.NewMenuItem("Save scrollback…", func() { sess.PromptSaveScrollback() }),
		fyne.NewMenuItem("Pack ticket evidence…", func() { h.packSessionEvidence(sess, label) }),
		fyne.NewMenuItem("Document work to PagerDuty…", func() { h.documentWorkToIncident() }),
	}
	widget.ShowPopUpMenuAtRelativePosition(
		fyne.NewMenu("", items...), h.win.Canvas(), fyne.NewPos(0, anchor.Size().Height), anchor)
}

func (h *host) showImportMenu(anchor fyne.CanvasObject) {
	items := []*fyne.MenuItem{
		fyne.NewMenuItem("Session YAML…", h.importSessions),
		fyne.NewMenuItem("Topology map…", h.importMap),
		fyne.NewMenuItem("SecureCRT sessions…", h.importSecureCRT),
		fyne.NewMenuItem("Customer crawl seeds (CSV)…", h.importCustomerCrawlCSV),
	}
	widget.ShowPopUpMenuAtRelativePosition(
		fyne.NewMenu("", items...), h.win.Canvas(), fyne.NewPos(0, anchor.Size().Height), anchor)
}

// buildMenu is replaced by buildChrome.
func (h *host) buildMenu() { h.buildChrome() }

func (h *host) currentTerminal() *ui.Session {
	inst := h.shell.Current()
	if inst == nil {
		return nil
	}
	ta, ok := inst.Applet().(*termApplet)
	if !ok || ta.sess == nil {
		return nil
	}
	return ta.sess
}

func (h *host) toggleCurrentCapture() {
	sess := h.currentTerminal()
	if sess == nil {
		dialog.ShowInformation("Session Capture", "Open a terminal session first.", h.win)
		return
	}
	_, msg := sess.ToggleLogging()
	if msg != "" {
		dialog.ShowInformation("Session Capture", msg, h.win)
	}
}

func (h *host) saveCurrentScrollback() {
	sess := h.currentTerminal()
	if sess == nil {
		dialog.ShowInformation("Save Scrollback", "Open a terminal session first.", h.win)
		return
	}
	sess.PromptSaveScrollback()
}

func (h *host) openFileTransfer() {
	sess := h.currentTerminal()
	if sess == nil {
		dialog.ShowInformation("File Transfer", "Open an SSH terminal session first.", h.win)
		return
	}
	title := "SFTP"
	if inst := h.shell.Current(); inst != nil {
		title = "SFTP — " + inst.Title()
	}
	h.openSFTPTab(sess, title)
}

func (h *host) openSFTPTab(sess *ui.Session, title string) {
	client, ok := sess.SSHClient()
	if !ok || client == nil {
		dialog.ShowInformation("File Transfer", "SFTP requires an SSH session (not telnet or serial).", h.win)
		return
	}
	view, err := ui.NewSFTPView(h.win, client)
	if err != nil {
		dialog.ShowError(err, h.win)
		return
	}
	if title == "" {
		title = "SFTP"
	}
	h.shell.Open(ui.Mount{
		Kind:   ui.KindSFTP,
		Title:  title,
		Applet: view,
		Busy:   func() string { return "" },
	})
}

func (h *host) openPortForwards() {
	sess := h.currentTerminal()
	if sess == nil {
		dialog.ShowInformation("Port forwards", "Open an SSH terminal session first.", h.win)
		return
	}
	client, ok := sess.SSHClient()
	if !ok || client == nil {
		dialog.ShowInformation("Port forwards", "Port forwards require an SSH session.", h.win)
		return
	}
	title := "Port forwards"
	if inst := h.shell.Current(); inst != nil {
		title = "Port forwards — " + inst.Title()
	}
	ui.ShowPortForwardDialog(h.win, title, client, h.forwards)
}

func (h *host) loadScripts() scripts.File {
	path := scripts.Path(ui.GetAppHome())
	f, err := scripts.Load(path)
	if err != nil {
		log.Printf("[scripts] %v", err)
		return scripts.Defaults()
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		_ = scripts.Save(path, f)
	}
	return f
}

func (h *host) scriptSender() ui.ScriptSender {
	return scriptSend{h: h}
}

type scriptSend struct{ h *host }

func (s scriptSend) SendActive(text string) { s.h.shell.SendToActive(text) }
func (s scriptSend) SendAll(text string)    { s.h.shell.SendToAllTerminals(text) }

func (s scriptSend) WaitForPattern(ctx context.Context, pattern string, regex bool, timeout time.Duration) error {
	sess := s.h.activeScriptSession()
	if sess == nil {
		return fmt.Errorf("no active SSH session")
	}
	return sess.WaitForPattern(ctx, pattern, regex, timeout)
}

// activeScriptSession returns the Session mounted in the active terminal tab.
func (h *host) activeScriptSession() *ui.Session {
	if h == nil || h.shell == nil {
		return nil
	}
	cur := h.shell.ActiveTerminal()
	if cur == nil {
		return nil
	}
	if ta, ok := cur.Applet().(*termApplet); ok {
		return ta.sess
	}
	return nil
}

func (h *host) runScriptDialog() {
	ui.ShowRunScriptDialog(h.win, h.loadScripts(), h.scriptSender(), &h.scriptCancel)
}

func (h *host) openScriptEditor() {
	path := scripts.Path(ui.GetAppHome())
	ui.ShowScriptEditor(h.win, ui.ScriptEditorOptions{
		Path: path,
		File: h.loadScripts(),
		OnSaved: func(scripts.File) {
			// Menu is rebuilt on each Scripts button tap from disk.
		},
	})
}

func (h *host) runNamedScript(name string) {
	h.runNamedScriptScoped(name, false)
}

func (h *host) runNamedScriptScoped(name string, allTabs bool) {
	f := h.loadScripts()
	for _, sc := range f.Scripts {
		if sc.Name != name {
			continue
		}
		if allTabs {
			sc.Scope = "all"
		}
		if prev := h.scriptCancel.Load(); prev != nil {
			(*prev)()
		}
		ctx, cancel := context.WithCancel(context.Background())
		h.scriptCancel.Store(&cancel)
		go func(sc scripts.Script) {
			err := scripts.Run(ctx, sc, h.scriptSender())
			fyne.Do(func() {
				h.scriptCancel.Store(nil)
				if err != nil {
					dialog.ShowError(err, h.win)
				}
			})
		}(sc)
		return
	}
	dialog.ShowInformation("Scripts", "Script not found: "+name, h.win)
}

// installButtonBar, installOpsStrip, installScriptsBar — folded into buildRibbon.
func (h *host) installButtonBar()  {}
func (h *host) installOpsStrip()   {}
func (h *host) installScriptsBar() {}

// pickFile opens a read picker filtered to one set of extensions and hands the
// whole file to use. Reading here rather than in each caller keeps the three
// menu actions about what the bytes MEAN.
//
// A nil reader is a cancel, not a failure, and says nothing.
func (h *host) pickFile(exts []string, dir string, use func(path string, data []byte)) {
	d := dialog.NewFileOpen(func(r fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, h.win)
			return
		}
		if r == nil {
			return
		}
		defer r.Close()

		data, err := io.ReadAll(r)
		if err != nil {
			dialog.ShowError(fmt.Errorf("read %s: %w", r.URI().Name(), err), h.win)
			return
		}
		use(r.URI().Path(), data)
	}, h.win)

	d.SetFilter(storage.NewExtensionFileFilter(exts))
	if l := listerFor(dir); l != nil {
		d.SetLocation(l)
	}
	d.Resize(fyne.NewSize(820, 600))
	d.Show()
}

// listerFor turns a directory into something the picker can start in, or nil.
// A directory that has gone away is not worth a message — the picker opens
// wherever it would have opened anyway.
func listerFor(dir string) fyne.ListableURI {
	if strings.TrimSpace(dir) == "" {
		return nil
	}
	l, err := storage.ListerForURI(storage.NewFileURI(dir))
	if err != nil {
		return nil
	}
	return l
}

// importSessions merges another session file into this one.
//
// The file's own folders are kept. Which reader runs is decided by the shape of
// the document rather than by its name, so this also accepts the older
// terminal's file without the person having to say which kind it is — and says
// so when the file picked is a map.
func (h *host) importSessions() {
	h.pickFile([]string{".yaml", ".yml"}, filepath.Dir(h.sessionsPath), func(path string, data []byte) {
		folders, format, err := sessions.FoldersFrom(data)
		if err != nil {
			dialog.ShowError(fmt.Errorf("%s: %w", filepath.Base(path), err), h.win)
			return
		}
		tr := h.tree.Tree()
		h.applyImport(tr, format, tr.ImportFolders(folders))
	})
}

// importSecureCRT merges VanDyke Sessions\**\*.ini into the tree.
// Nested CRT folders become nested Pathfinder folders. Passwords are never
// read from CRT files. Parse/merge run off the UI thread — sync walk of ~800
// .ini files froze the window before the confirm dialog even appeared.
func (h *host) importSecureCRT() {
	cfg := crtimport.DefaultConfig()
	if cfg == "" {
		dialog.ShowInformation("SecureCRT", "No VanDyke Config folder found under AppData\\Roaming\\VanDyke\\Config.", h.win)
		return
	}
	prog := dialog.NewProgressInfinite("SecureCRT", "Reading session files…", h.win)
	prog.Show()
	go func() {
		list, err := crtimport.Import(cfg)
		if err != nil {
			fyne.Do(func() {
				prog.Hide()
				dialog.ShowError(err, h.win)
			})
			return
		}
		folders, supported, skipped := crtimport.Folders(list)
		top := sessions.TopLevelNamesFromFolders(folders)
		fyne.Do(func() {
			prog.Hide()
			if supported == 0 {
				dialog.ShowInformation("SecureCRT", fmt.Sprintf("No importable sessions in %s (%d skipped).", cfg, skipped), h.win)
				return
			}
			ui.ShowCRTImportWizard(h.win, cfg, supported, skipped, top, func(choice ui.CRTImportChoice) {
				prog2 := dialog.NewProgressInfinite("SecureCRT", "Importing into Customers / Unassigned…", h.win)
				prog2.Show()
				go func() {
					tr := h.tree.Tree()
					if choice.Replace {
						if err := (&tr).ResetMSPInventory(); err != nil {
							fyne.Do(func() {
								prog2.Hide()
								dialog.ShowError(err, h.win)
							})
							return
						}
					} else {
						_, _ = (&tr).EnsureMSPLayout()
					}
					sum := tr.ImportFolders(folders)
					if err := (&tr).OrganiseCRTImport(choice.CustomerRoot); err != nil {
						fyne.Do(func() {
							prog2.Hide()
							dialog.ShowError(err, h.win)
						})
						return
					}
					fyne.Do(func() {
						prog2.Hide()
						h.applyImport(tr, sessions.FormatNative, sum)
					})
				}()
			})
		})
	}()
}

func (h *host) sendChat(text string, mode ui.ChatSendMode, customer string) {
	if ok, reason := h.allowSend(); !ok {
		dialog.ShowInformation("Send blocked", reason, h.win)
		return
	}
	switch mode {
	case ui.ChatSendAll:
		h.confirmFanOut("Send to all open terminals?", func() {
			h.shell.SendToAllTerminals(text)
		})
	case ui.ChatSendCustomer:
		cust := customer
		h.confirmFanOut("Send to all open terminals for customer "+cust+"?", func() {
			h.shell.SendToMatching(text, func(meta ui.TerminalMeta) bool {
				return strings.EqualFold(meta.CustomerName(), cust)
			})
		})
	default:
		if !h.shell.SendToActive(text) {
			dialog.ShowInformation("Send", "No SSH session to send to.", h.win)
			return
		}
		h.shell.RefocusCurrentTerminal()
	}
}

func (h *host) confirmFanOut(msg string, do func()) {
	if !h.guardedSend {
		do()
		return
	}
	dialog.ShowConfirm("Guarded send", msg, func(ok bool) {
		if ok {
			do()
		}
	}, h.win)
}

func (h *host) sendButton(b buttons.Button) {
	h.sendButtonScoped(b, strings.EqualFold(b.Scope, "all"))
}

func (h *host) barButtonAction(b buttons.Button, all bool) {
	if strings.TrimSpace(b.Script) != "" {
		h.runNamedScriptScoped(b.Script, all || strings.EqualFold(b.Scope, "all"))
		return
	}
	h.sendButtonScoped(b, all)
}

func (h *host) sendButtonScoped(b buttons.Button, all bool) {
	text := strings.TrimRight(b.Send, "\r\n")
	if text == "" {
		return
	}
	// Enter key in the terminal is CR; append that (NormalizeTerminalSend also
	// maps any YAML \n to \r inside SendToActive / SendToAllTerminals).
	text += "\r"
	if ok, reason := h.allowSend(); !ok {
		dialog.ShowInformation("Send blocked", reason, h.win)
		return
	}
	if all {
		h.confirmFanOut("Send button \""+b.Label+"\" to all open terminals?", func() {
			h.shell.SendToAllTerminals(text)
		})
		return
	}
	if !h.shell.SendToActive(text) {
		dialog.ShowInformation("Send", "No SSH session to send to (or send was blocked).", h.win)
		return
	}
	h.shell.EnsureTerminalFocus()
}

func (h *host) allowSend() (bool, string) {
	p := policy.Policy{
		ReadOnly:          h.base.ReadOnlyMode,
		ChangeWindowStart: h.base.ChangeWindowStart,
		ChangeWindowEnd:   h.base.ChangeWindowEnd,
	}
	return p.Allow(time.Now())
}

func (h *host) startScriptRecording() {
	name := widget.NewEntry()
	name.SetText("Recorded " + time.Now().Format("15:04"))
	dialog.ShowForm("Start script recording", "Record", "Cancel", []*widget.FormItem{
		{Text: "Script name", Widget: name},
	}, func(ok bool) {
		if !ok {
			return
		}
		h.recorder = scripts.NewRecorder(name.Text)
		dialog.ShowInformation("Recording", "Keystrokes and button/script sends are being recorded.\nSession → Stop script recording… to save.", h.win)
	}, h.win)
}

func (h *host) stopScriptRecording() {
	if h.recorder == nil || !h.recorder.Active() {
		dialog.ShowInformation("Recording", "No recording in progress.", h.win)
		return
	}
	h.recorder.Stop()
	s := h.recorder.Script()
	h.recorder = nil
	if len(s.Steps) == 0 {
		dialog.ShowInformation("Recording", "Nothing was captured.", h.win)
		return
	}
	path := scripts.Path(ui.GetAppHome())
	f := h.loadScripts()
	f = scripts.Upsert(f, s)
	if err := scripts.Save(path, f); err != nil {
		dialog.ShowError(err, h.win)
		return
	}
	dialog.ShowInformation("Recording saved", fmt.Sprintf("Saved %q (%d steps) to %s", s.Name, len(s.Steps), path), h.win)
}

func (h *host) runPythonScript() {
	sess := h.currentTerminal()
	if sess == nil {
		dialog.ShowInformation("Python", "Open a terminal session first.", h.win)
		return
	}
	if ok, reason := h.allowSend(); !ok {
		dialog.ShowInformation("Send blocked", reason, h.win)
		return
	}
	open := dialog.NewFileOpen(func(uc fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, h.win)
			return
		}
		if uc == nil {
			return
		}
		path := uc.URI().Path()
		_ = uc.Close()
		ctx, cancel := context.WithCancel(context.Background())
		prev := h.scriptCancel.Swap(&cancel)
		if prev != nil && *prev != nil {
			(*prev)()
		}
		go func() {
			err := pyrun.Run(ctx, "", path, pyrun.Callbacks{
				Send: func(text string) error {
					if !sess.SendUser([]byte(text)) {
						return fmt.Errorf("send blocked by policy")
					}
					return nil
				},
				WaitFor: func(ctx context.Context, pattern string, timeout time.Duration) error {
					return sess.WaitForPattern(ctx, pattern, false, timeout)
				},
			})
			fyne.Do(func() {
				if err != nil && ctx.Err() == nil {
					dialog.ShowError(err, h.win)
				}
			})
		}()
	}, h.win)
	open.SetFilter(storage.NewExtensionFileFilter([]string{".py"}))
	open.Resize(fyne.NewSize(820, 600))
	open.Show()
}

func (h *host) installShortcuts() {
	c := h.win.Canvas()
	add := func(key fyne.KeyName, mod fyne.KeyModifier, fn func()) {
		sc := &desktop.CustomShortcut{KeyName: key, Modifier: mod}
		c.AddShortcut(sc, func(fyne.Shortcut) { fn() })
	}
	add(fyne.KeyW, fyne.KeyModifierControl, func() {
		h.shell.CloseCurrent()
	})
	add(fyne.KeyTab, fyne.KeyModifierControl, func() {
		h.shell.SelectNextTab()
	})
	add(fyne.KeyTab, fyne.KeyModifierControl|fyne.KeyModifierShift, func() {
		h.shell.SelectPrevTab()
	})
	add(fyne.KeyL, fyne.KeyModifierControl, func() {
		// Focus tree filter when present.
		if h.tree != nil {
			h.tree.FocusFilter()
		}
	})
}

func (h *host) toggleGuardedSend() {
	h.guardedSend = !h.guardedSend
	state := "off"
	if h.guardedSend {
		state = "on"
	}
	dialog.ShowInformation("Guarded multi-send", "Confirm before Send to all / customer is now "+state+".", h.win)
}

func (h *host) packTicketEvidence() {
	ticket := widget.NewEntry()
	ticket.SetPlaceHolder("ticket / change ID")
	dialog.ShowForm("Pack ticket evidence", "Save zip…", "Cancel", []*widget.FormItem{
		{Text: "Ticket", Widget: ticket},
	}, func(ok bool) {
		if !ok {
			return
		}
		var files []evidence.File
		for _, inst := range h.shell.Instances() {
			if inst == nil {
				continue
			}
			ta, ok := inst.Applet().(*termApplet)
			if !ok || ta.sess == nil {
				continue
			}
			text := ta.sess.ScrollbackText()
			if strings.TrimSpace(text) == "" {
				continue
			}
			files = append(files, evidence.File{
				Name:    inst.Title() + ".txt",
				Content: []byte(text),
			})
		}
		if len(files) == 0 {
			dialog.ShowInformation("Evidence pack", "No open terminals with scrollback.", h.win)
			return
		}
		suggested := filepath.Join(ui.GetLogsDir(), "evidence-"+time.Now().Format("20060102-150405")+".zip")
		save := dialog.NewFileSave(func(wc fyne.URIWriteCloser, err error) {
			if err != nil {
				dialog.ShowError(err, h.win)
				return
			}
			if wc == nil {
				return
			}
			path := wc.URI().Path()
			_ = wc.Close()
			if err := evidence.WriteZip(path, ticket.Text, files); err != nil {
				dialog.ShowError(err, h.win)
				return
			}
			dialog.ShowInformation("Evidence pack", fmt.Sprintf("Wrote %d scrollback(s) to %s", len(files), path), h.win)
		}, h.win)
		save.SetFileName(filepath.Base(suggested))
		save.Resize(fyne.NewSize(820, 600))
		save.Show()
	}, h.win)
}

func (h *host) packSessionEvidence(sess *ui.Session, title string) {
	if sess == nil {
		return
	}
	text := sess.ScrollbackText()
	if strings.TrimSpace(text) == "" {
		dialog.ShowInformation("Evidence pack", "No scrollback to save for this session.", h.win)
		return
	}
	ticket := widget.NewEntry()
	ticket.SetPlaceHolder("ticket / change ID")
	dialog.ShowForm("Pack ticket evidence", "Save zip…", "Cancel", []*widget.FormItem{
		{Text: "Ticket", Widget: ticket},
	}, func(ok bool) {
		if !ok {
			return
		}
		safe := strings.Map(func(r rune) rune {
			if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
				return r
			}
			return '_'
		}, title)
		suggested := filepath.Join(ui.GetLogsDir(), "evidence-"+safe+"-"+time.Now().Format("20060102-150405")+".zip")
		save := dialog.NewFileSave(func(wc fyne.URIWriteCloser, err error) {
			if err != nil {
				dialog.ShowError(err, h.win)
				return
			}
			if wc == nil {
				return
			}
			path := wc.URI().Path()
			_ = wc.Close()
			files := []evidence.File{{Name: title + ".txt", Content: []byte(text)}}
			if err := evidence.WriteZip(path, ticket.Text, files); err != nil {
				dialog.ShowError(err, h.win)
				return
			}
			dialog.ShowInformation("Evidence pack", "Wrote scrollback to "+path, h.win)
		}, h.win)
		save.SetFileName(filepath.Base(suggested))
		save.Resize(fyne.NewSize(820, 600))
		save.Show()
	}, h.win)
}

func (h *host) showCursorAccount() {
	ui.ShowCursorAccountDialog(h.win, h.base.CursorAPIKey)
}

// refreshTroubleshootChrome rebuilds the ribbon when the addon toggles.
func (h *host) refreshTroubleshootChrome() {
	if h.shell == nil {
		return
	}
	h.buildChrome()
	h.applyCursorPane()
	if c := h.shell.Content(); c != nil {
		c.Refresh()
	}
}

func (h *host) toggleCursorPane() {
	if !h.base.TroubleshootAddon {
		dialog.ShowInformation("Cursor AI",
			"Enable Troubleshoot addon in Settings → Tools first.", h.win)
		return
	}
	h.cursorPaneVisible = !h.cursorPaneVisible
	h.buildChrome()
	h.applyCursorPane()
}

func (h *host) applyCursorPane() {
	if h.shell == nil {
		return
	}
	if !h.base.TroubleshootAddon || !h.cursorPaneVisible {
		h.shell.SetRight(nil, 0)
		return
	}
	if h.cursorPane == nil {
		h.cursorPane = ui.NewCursorPane(h.win, h.cursorPaneHooks())
	}
	off := h.shell.RightOffset()
	if off <= 0 {
		off = 0.72
	}
	h.shell.SetRight(h.cursorPane, off)
}

func (h *host) activeSessionContext() string {
	inst := h.shell.Current()
	if inst == nil {
		return ui.FormatActiveContext("", "", "", "", false)
	}
	ta, ok := inst.Applet().(*termApplet)
	if !ok || ta.sess == nil {
		return ui.FormatActiveContext(inst.Title(), "", "", "", false)
	}
	return ui.FormatActiveContext(inst.Title(), ta.customer, ta.folder, ta.sess.TargetLabel(), true)
}

func (h *host) gatherScrollback(all bool) (string, error) {
	if !all {
		sess := h.currentTerminal()
		if sess == nil {
			return "", fmt.Errorf("no active terminal")
		}
		return sess.ScrollbackText(), nil
	}
	var b strings.Builder
	for _, inst := range h.shell.Instances() {
		if inst == nil {
			continue
		}
		ta, ok := inst.Applet().(*termApplet)
		if !ok || ta.sess == nil {
			continue
		}
		fmt.Fprintf(&b, "----- %s -----\n%s\n\n", inst.Title(), ta.sess.ScrollbackText())
	}
	if b.Len() == 0 {
		return "", fmt.Errorf("no terminal scrollbacks")
	}
	return b.String(), nil
}

func (h *host) sendToActiveSession(cmd string) error {
	if !h.shell.SendToActive(cmd) {
		return fmt.Errorf("no active terminal or send rejected")
	}
	return nil
}

func (h *host) askCursorAgent(prompt string) (string, error) {
	cli := cursorapi.New(h.base.CursorAPIKey)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	out, err := cli.CreateAgent(ctx, cursorapi.CreateAgentRequest{
		Prompt: cursorapi.CreatePrompt{Text: prompt},
		Name:   "PathfinderSSH troubleshoot",
	})
	if err != nil {
		return "", err
	}
	msg := fmt.Sprintf("Agent %s (%s) · run %s (%s)",
		out.Agent.ID, out.Agent.Status, out.Run.ID, out.Run.Status)
	if out.Agent.URL != "" {
		msg += "\n" + out.Agent.URL
	}
	return msg, nil
}

func (h *host) cursorPaneHooks() ui.CursorPaneHooks {
	return ui.CursorPaneHooks{
		APIKey:           h.base.CursorAPIKey,
		ActiveContext:    h.activeSessionContext,
		GatherScrollback: h.gatherScrollback,
		SendToActive:     h.sendToActiveSession,
		ScriptNames: func() []string {
			f := h.loadScripts()
			names := make([]string, 0, len(f.Scripts))
			for _, s := range f.Scripts {
				names = append(names, s.Name)
			}
			return names
		},
		RunScript: func(name string) error {
			h.runNamedScript(name)
			return nil
		},
		AskCursor: h.askCursorAgent,
	}
}

func (h *host) showTroubleshootAgent() {
	if !h.base.TroubleshootAddon {
		dialog.ShowInformation("Troubleshoot addon",
			"Enable it in Settings → Tools → Enable addon.", h.win)
		return
	}
	ui.ShowTroubleshootAgent(h.win, ui.TroubleshootHooks{
		Enabled: true,
		APIKey:  h.base.CursorAPIKey,
		ListSessions: func() []ui.TroubleshootSession {
			cur := h.shell.Current()
			out := make([]ui.TroubleshootSession, 0)
			for _, inst := range h.shell.Instances() {
				if inst == nil {
					continue
				}
				ta, ok := inst.Applet().(*termApplet)
				if !ok || ta.sess == nil {
					continue
				}
				out = append(out, ui.TroubleshootSession{
					Title:    inst.Title(),
					Customer: ta.customer,
					Folder:   ta.folder,
					Target:   ta.sess.TargetLabel(),
					Active:   cur != nil && inst == cur,
				})
			}
			return out
		},
		GatherScrollback: h.gatherScrollback,
		ScriptNames: func() []string {
			f := h.loadScripts()
			names := make([]string, 0, len(f.Scripts))
			for _, s := range f.Scripts {
				names = append(names, s.Name)
			}
			return names
		},
		RunScript: func(name string) error {
			h.runNamedScript(name)
			return nil
		},
		CheckCursor: func() (string, error) {
			cli := cursorapi.New(h.base.CursorAPIKey)
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			me, err := cli.Me(ctx)
			if err != nil {
				return "", err
			}
			who := strings.TrimSpace(me.UserEmail)
			if who == "" {
				who = "(email not returned)"
			}
			key := strings.TrimSpace(me.APIKeyName)
			if key == "" {
				key = "(unnamed key)"
			}
			return fmt.Sprintf("Cursor OK — user=%s key=%s", who, key), nil
		},
		LaunchCursor: func(prompt, repo, ref, name string) (string, error) {
			if strings.TrimSpace(repo) != "" {
				cli := cursorapi.New(h.base.CursorAPIKey)
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				defer cancel()
				req := cursorapi.CreateAgentRequest{
					Prompt: cursorapi.CreatePrompt{Text: prompt},
					Name:   strings.TrimSpace(name),
					Repos: []cursorapi.RepoSpec{{
						URL:         strings.TrimSpace(repo),
						StartingRef: strings.TrimSpace(ref),
					}},
				}
				out, err := cli.CreateAgent(ctx, req)
				if err != nil {
					return "", err
				}
				msg := fmt.Sprintf("Agent %s (%s) · run %s (%s)",
					out.Agent.ID, out.Agent.Status, out.Run.ID, out.Run.Status)
				if out.Agent.URL != "" {
					msg += "\n" + out.Agent.URL
				}
				return msg, nil
			}
			return h.askCursorAgent(prompt)
		},
	})
}

func (h *host) syncPSACustomers() {
	path := psasync.DefaultPath(ui.GetAppHome())
	_ = psasync.WriteExample(path)
	var src psasync.Source = psasync.FileSource{Path: path}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	list, err := src.ListCustomers(ctx)
	if err != nil {
		if os.IsNotExist(err) {
			src = psasync.StubSource{}
			list, err = src.ListCustomers(ctx)
		}
		if err != nil {
			dialog.ShowError(err, h.win)
			return
		}
	}
	tr := h.tree.Tree()
	root := product.CustomersRoot
	res := psasync.SyncFolderNames(root, list, func(name string) (string, error) {
		path, err := (&tr).CreateCustomer(root, name)
		if err != nil {
			if strings.Contains(err.Error(), "already exists") {
				return "", nil
			}
			return "", err
		}
		return path, nil
	})
	res.Source = src.Name()
	h.tree.SetTree(tr)
	h.saveTree(tr)
	msg := fmt.Sprintf("Source: %s\nFile: %s\nCreated: %d\nAlready present: %d", res.Source, path, len(res.Created), len(res.Existing))
	if len(res.Errors) > 0 {
		msg += "\nErrors: " + strings.Join(res.Errors, "; ")
	}
	dialog.ShowInformation("Customer list import", msg, h.win)
}

func (h *host) importFromAuvik() {
	cli := auvik.New(h.base.AuvikUsername, h.base.AuvikAPIKey, h.base.AuvikBaseURL)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	if err := cli.Verify(ctx); err != nil {
		dialog.ShowError(err, h.win)
		return
	}
	tenants, err := cli.ListTenants(ctx)
	if err != nil {
		dialog.ShowError(err, h.win)
		return
	}
	ui.ShowAuvikImportDialog(h.win, ui.AuvikImportOptions{
		Tenants:       tenants,
		CustomerNames: h.mspCustomerNames(),
		OnImport: func(tenantIDs []string, customerFolder string, imp auvik.ImportOptions) (auvik.ImportStats, error) {
			ctx2, cancel2 := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel2()
			devs, err := cli.ListDevices(ctx2, tenantIDs, 300)
			if err != nil {
				return auvik.ImportStats{}, err
			}
			var tenant *auvik.Tenant
			for i := range tenants {
				if tenants[i].ID == tenantIDs[0] {
					tenant = &tenants[i]
					break
				}
			}
			if tenant == nil {
				return auvik.ImportStats{}, fmt.Errorf("unknown tenant id %q", tenantIDs[0])
			}
			imp.DefaultUsername = strings.TrimSpace(imp.DefaultUsername)
			if imp.DefaultUsername == "" {
				imp.DefaultUsername = h.base.AuvikDefaultUsername
			}
			if strings.TrimSpace(imp.DefaultCredential) == "" {
				imp.DefaultCredential = h.base.AuvikDefaultCredential
			}
			customer := h.resolveMSPCustomer(customerFolder)
			if customer == "" {
				customer = h.resolveMSPCustomer(tenant.Name)
			}
			tr := h.tree.Tree()
			syncRes := auvik.SyncTenantTree(&tr, devs, auvik.SyncOptions{
				ImportOptions:     imp,
				Tenant:            *tenant,
				CustomerName:      customer,
				MoveToAuvikFolder: true,
				UseTunnelDefault:  h.base.AuvikAutoTunnel,
			})
			h.tree.SetTree(tr)
			h.saveTree(tr)
			st := auvik.ImportStats{
				Imported: syncRes.Created,
				Skipped:  syncRes.Skipped,
				NoIP:     syncRes.NoIP,
				Errors:   syncRes.Errors,
			}
			// Surface merge/update counts in the summary string via Errors note.
			if syncRes.Merged > 0 || syncRes.Updated > 0 || syncRes.Moved > 0 {
				st.Errors = append(st.Errors,
					fmt.Sprintf("merged %d, updated %d, moved %d", syncRes.Merged, syncRes.Updated, syncRes.Moved))
			}
			return st, nil
		},
	})
}

func (h *host) importFromITGlue() {
	if h.vault == nil {
		dialog.ShowInformation("IT Glue",
			"Unlock or create the credential vault first (Settings → File → Manage credentials).", h.win)
		return
	}
	cli := itglue.New(h.base.ITGlueAPIKey, h.base.ITGlueBaseURL)
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	if err := cli.Verify(ctx); err != nil {
		dialog.ShowError(err, h.win)
		return
	}
	orgs, err := cli.ListOrganizations(ctx)
	if err != nil {
		dialog.ShowError(err, h.win)
		return
	}
	ui.ShowITGlueImportDialog(h.win, ui.ITGlueImportOptions{
		Organizations: orgs,
		CustomerNames: h.mspCustomerNames(),
		OnImport: func(orgID string, imp itglue.ImportDialogOptions) (itglue.ImportResult, error) {
			ctx2, cancel2 := context.WithTimeout(context.Background(), 180*time.Second)
			defer cancel2()
			list, err := cli.ListPasswords(ctx2, orgID)
			if err != nil {
				return itglue.ImportResult{}, err
			}
			if imp.SSHFilter {
				list = itglue.FilterSSHPasswords(list)
			}
			full, err := cli.FetchPasswordSecrets(ctx2, list)
			if err != nil {
				return itglue.ImportResult{}, err
			}
			vst, err := itglue.SyncPasswordsToVault(h.vault, full, itglue.VaultSyncOptions{
				UpdateExisting: imp.UpdateVault,
			})
			if err != nil {
				return itglue.ImportResult{}, err
			}
			res := itglue.ImportResult{Vault: vst}
			if imp.LinkSessions {
				credNames, err := itglue.CredNamesFromVault(h.vault)
				if err != nil {
					res.Errors = append(res.Errors, err.Error())
				} else {
					customer := h.resolveMSPCustomer(strings.TrimSpace(imp.CustomerName))
					if customer == "" {
						for _, o := range orgs {
							if o.ID == orgID {
								customer = h.resolveMSPCustomer(o.Name)
								break
							}
						}
					}
					tr := h.tree.Tree()
					link := itglue.LinkSessions(&tr, full, credNames, itglue.LinkOptions{
						CustomerName: customer,
						OnlyEmpty:    imp.OnlyEmptyCreds,
					})
					res.Link = link
					h.tree.SetTree(tr)
					h.saveTree(tr)
				}
			}
			h.refreshVault()
			if len(vst.Errors) > 0 {
				res.Errors = append(res.Errors, vst.Errors...)
			}
			return res, nil
		},
	})
}

func (h *host) syncAuvikNow() {
	h.runAuvikSyncAll(true)
}

type auvikSyncAggregate struct {
	Tenants int
	Summary string
	Changed bool
}

func (h *host) auvikImportDefaults() auvik.ImportOptions {
	user, cred := h.mspInventoryDefaults()
	return auvik.ImportOptions{
		NetworkGearOnly:        true,
		RequireLoginAuthorized: true,
		DefaultUsername:        user,
		DefaultCredential:      cred,
	}
}

func (h *host) auvikSyncDefaults(tenant auvik.Tenant) auvik.SyncOptions {
	return auvik.SyncOptions{
		ImportOptions:     h.auvikImportDefaults(),
		Tenant:            tenant,
		CustomerName:      h.resolveMSPCustomer(tenant.Name),
		MoveToAuvikFolder: true,
		UseTunnelDefault:  h.base.AuvikAutoTunnel,
	}
}

func (h *host) runAuvikSyncAll(interactive bool) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		agg, err := h.auvikSyncPass(ctx)
		fyne.Do(func() {
			if interactive {
				if err != nil {
					dialog.ShowError(err, h.win)
					return
				}
				dialog.ShowInformation("Auvik sync", agg.Summary, h.win)
				return
			}
			if err != nil {
				log.Printf("[auvik] background sync: %v", err)
				return
			}
			if agg.Changed {
				log.Printf("[auvik] background sync: %s", agg.Summary)
			}
		})
	}()
}

func (h *host) auvikSyncPass(ctx context.Context) (auvikSyncAggregate, error) {
	cli := auvik.New(h.base.AuvikUsername, h.base.AuvikAPIKey, h.base.AuvikBaseURL)
	if err := cli.Verify(ctx); err != nil {
		return auvikSyncAggregate{}, err
	}
	tenants, err := cli.ListTenants(ctx)
	if err != nil {
		return auvikSyncAggregate{}, err
	}
	tr := h.tree.Tree()
	agg := auvikSyncAggregate{Tenants: len(tenants)}
	var parts []string
	for _, tenant := range tenants {
		devs, err := cli.ListDevices(ctx, []string{tenant.ID}, 300)
		if err != nil {
			parts = append(parts, tenant.Name+": "+err.Error())
			continue
		}
		res := auvik.SyncTenantTree(&tr, devs, h.auvikSyncDefaults(tenant))
		if res.Changed() {
			agg.Changed = true
			parts = append(parts, tenant.Name+": "+res.Summary())
		}
		if len(res.Errors) > 0 {
			parts = append(parts, tenant.Name+" errors: "+strings.Join(res.Errors, "; "))
		}
	}
	if agg.Changed {
		h.tree.SetTree(tr)
		h.saveTree(tr)
	}
	if len(parts) == 0 {
		agg.Summary = fmt.Sprintf("no changes across %d client(s)", agg.Tenants)
	} else {
		agg.Summary = strings.Join(parts, "\n")
	}
	return agg, nil
}

func (h *host) startAuvikPeriodicSync() {
	if h.auvikSyncCancel != nil {
		h.auvikSyncCancel()
		h.auvikSyncCancel = nil
	}
	if !h.base.AuvikSyncEnabled {
		return
	}
	interval := h.base.AuvikSyncIntervalMin
	if interval < 5 {
		interval = 60
	}
	ctx, cancel := context.WithCancel(context.Background())
	h.auvikSyncCancel = cancel
	go func() {
		time.Sleep(30 * time.Second)
		ticker := time.NewTicker(time.Duration(interval) * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				h.runAuvikSyncAll(false)
			}
		}
	}()
}

func (h *host) tryAuvikTunnelConnect(ctx context.Context, node sessions.Node, opts sessiondial.Options) (term.Transport, error) {
	if h.auvikTunnels == nil {
		h.auvikTunnels = auvik.NewTunnelManager(h.base.AuvikTunnelPath)
	}
	remotePort := node.Port
	if remotePort == 0 {
		remotePort = sessions.TransportSSH.DefaultPort()
	}
	local, err := h.auvikTunnels.Ensure(ctx, node.AuvikDomain, node.Host, remotePort, node.AuvikTunnelPort)
	if err != nil {
		return nil, err
	}
	tunOpts := opts
	tunOpts.SkipReachCheck = true
	tunNode := auvik.DialNodeViaTunnel(node, local)
	log.Printf("[auvik] tunnel localhost:%d → %s:%d (%s)", local, node.Host, remotePort, node.AuvikDomain)
	return sessiondial.Connect(ctx, tunNode, tunOpts)
}

// importMap turns a crawl's map.json into sessions under Customers/<name>/….
func (h *host) importMap() {
	dir := h.mapDir
	if dir == "" && h.lastCrawl.MapPath != "" {
		dir = filepath.Dir(h.lastCrawl.MapPath)
	}
	if dir == "" {
		if c := h.mapCustomer; c != "" {
			dir = ui.CustomerMapsDir(ui.GetAppHome(), c)
		} else {
			dir = ui.MapsRootDir(ui.GetAppHome())
		}
	}

	h.pickFile([]string{".json"}, dir, func(path string, data []byte) {
		h.mapDir = filepath.Dir(path)
		if cust := ui.InferCustomerFromMapsPath(path); cust != "" {
			h.mapCustomer = cust
		}
		if f := sessions.Sniff(data); f != sessions.FormatMap {
			dialog.ShowError(fmt.Errorf("%s is a %s, not a topology map", filepath.Base(path), f), h.win)
			return
		}
		h.askMapImport(path, mapFolderName(path), data)
	})
}

// mapFolderName is the site/crawl folder name under Customers/<customer>/
// unless the person overrides it. Prefer the crawl date filename; if the file
// is map.json, use the parent directory (often the customer — then we still
// ask for a site name).
func mapFolderName(path string) string {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	if strings.EqualFold(base, "map") {
		if dir := filepath.Base(filepath.Dir(path)); dir != "" && dir != "." && dir != string(filepath.Separator) {
			return dir
		}
	}
	return base
}

// askMapImport collects customer + folder and the leaves decision.
func (h *host) askMapImport(mapPath, defaultFolder string, data []byte) {
	customers := h.customerNames()
	inferred := ui.InferCustomerFromMapsPath(mapPath)
	if inferred == "" {
		inferred = h.mapCustomer
	}
	if inferred != "" {
		customers = mergeStrings(inferred, customers)
	}

	custSel := widget.NewSelect(customers, nil)
	if inferred != "" {
		custSel.SetSelected(inferred)
	} else if len(customers) > 0 {
		custSel.SetSelected(customers[0])
	}

	name := widget.NewEntry()
	// Avoid duplicating the customer name as the only folder under itself.
	folderDef := defaultFolder
	if inferred != "" && strings.EqualFold(folderDef, inferred) {
		folderDef = time.Now().Format("crawl-2006-01-02")
	}
	name.SetText(folderDef)
	name.SetPlaceHolder("Site or crawl name")

	leaves := widget.NewCheck("", nil)
	leafItem := widget.NewFormItem("Include leaves", leaves)
	leafItem.HintText = "devices a neighbour reported but the crawl never dialled"

	items := []*widget.FormItem{
		widget.NewFormItem("Customer", custSel),
		widget.NewFormItem("Folder under customer", name),
		leafItem,
	}
	d := dialog.NewForm("Import topology map", "Import", "Cancel", items,
		func(ok bool) {
			if !ok {
				return
			}
			nodes, err := sessions.NodesFromMap(data, leaves.Checked)
			if err != nil {
				dialog.ShowError(err, h.win)
				return
			}
			customer := strings.TrimSpace(custSel.Selected)
			if customer == "" {
				dialog.ShowError(fmt.Errorf("pick a customer"), h.win)
				return
			}
			folder := strings.TrimSpace(name.Text)
			if folder == "" {
				folder = "Imported"
			}
			path := sessions.JoinPath(product.CustomersRoot, customer, folder)

			tr := h.tree.Tree()
			if _, err := (&tr).EnsureMSPLayout(); err != nil {
				dialog.ShowError(err, h.win)
				return
			}
			if _, err := (&tr).CreateCustomer(product.CustomersRoot, customer); err != nil {
				// CreateCustomer fails if it already exists — that is fine.
				if !strings.Contains(strings.ToLower(err.Error()), "exist") {
					dialog.ShowError(err, h.win)
					return
				}
			}
			sum := tr.ImportFolders([]sessions.Folder{{Name: path, Sessions: nodes}})
			h.mapCustomer = customer
			_, _ = ui.EnsureCustomerMapsDir(ui.GetAppHome(), customer)
			h.applyImport(tr, sessions.FormatMap, sum)
		}, h.win)
	d.Show()
	ui.EnterConfirmsForm(h.win, items, d.Submit)
}

func mergeStrings(first string, rest []string) []string {
	out := []string{first}
	seen := map[string]bool{strings.ToLower(first): true}
	for _, s := range rest {
		s = strings.TrimSpace(s)
		if s == "" || seen[strings.ToLower(s)] {
			continue
		}
		seen[strings.ToLower(s)] = true
		out = append(out, s)
	}
	return out
}

// applyImport puts the merged tree back, saves it, and says what happened.
//
// SetTree deliberately does not fire OnChanged — that callback is the widget
// telling the host something changed, and this is the host telling the widget
// what is true. So the save is explicit, and it happens before the summary: a
// dialog saying 13 sessions were added, over a file that was never written, is
// the failure worth ruling out.
func (h *host) applyImport(tr sessions.Tree, format sessions.Format, sum sessions.ImportSummary) {
	h.tree.SetTree(tr)
	h.saveTree(tr)
	h.buildChrome()
	msg := sum.Describe()
	switch format {
	case sessions.FormatMap:
		msg += "\n\nSessions were filed under Customers/<customer>/<folder>."
	default:
		msg += "\n\nRe-import merges by address: existing sessions stay, only new ones are added."
		if sum.Skipped > 0 && sum.Added == 0 {
			msg += "\nTip: choose Replace inventory in the SecureCRT wizard to start fresh."
		}
	}
	dialog.ShowInformation("Imported "+format.String(), msg, h.win)
}

// exportSessions writes the whole tree to a file of the person's choosing.
//
// What leaves is what is on disk already: folders, sessions, and a credential
// REFERENCE by name. No password and no passphrase, because the model marks
// both yaml:"-" and a test fails if either reaches the bytes — so a file handed
// to somebody else carries a map of the estate and nothing that unlocks it.
func (h *host) exportSessions() {
	tr := h.tree.Tree()

	// Marshal BEFORE the picker. The save dialog creates the file it
	// returns, and a marshal that failed after that would leave an empty
	// file where the person believes their inventory is.
	data, err := sessions.MarshalTree(tr)
	if err != nil {
		dialog.ShowError(fmt.Errorf("render the session file: %w", err), h.win)
		return
	}
	count := len(tr.Nodes())

	d := dialog.NewFileSave(func(wc fyne.URIWriteCloser, err error) {
		if err != nil {
			dialog.ShowError(err, h.win)
			return
		}
		if wc == nil {
			return
		}
		defer wc.Close()

		if _, err := wc.Write(data); err != nil {
			dialog.ShowError(fmt.Errorf("write %s: %w", wc.URI().Name(), err), h.win)
			return
		}
		dialog.ShowInformation("Exported",
			fmt.Sprintf("Wrote %d session(s) to %s.", count, wc.URI().Name()), h.win)
	}, h.win)

	d.SetFileName("sessions.yaml")
	d.SetFilter(storage.NewExtensionFileFilter([]string{".yaml", ".yml"}))
	if l := listerFor(filepath.Dir(h.sessionsPath)); l != nil {
		d.SetLocation(l)
	}
	d.Resize(fyne.NewSize(820, 600))
	d.Show()
}

func (h *host) connect(folder, oldLabel string, n sessions.Node, persist func(sessions.Node)) {
	// The ctx is created HERE, not inside the dial goroutine, because the
	// Cancel button has to be able to reach it. The 90s ceiling stays as the
	// unattended bound; Cancel is the attended one.
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)

	// Fyne's dialog.Hide() runs the onClosed callback too, so the Cancel
	// button and the success path both arrive here. One CAS decides which of
	// them got there first, and the loser does nothing: without it, hiding
	// the dialog on success would cancel the context it just finished with
	// and log a cancellation that never happened.
	var settled atomic.Bool

	progress := dialog.NewCustom("Connecting", "Cancel",
		widget.NewLabel("Checking reachability of "+n.Target()+" …"), h.win)
	// A dial that cannot be escaped is worse than a slow one: until this
	// existed, a serial open that parked in the driver left a modal over the
	// whole window and the only exit was killing the process. Hot-plugging an
	// adapter on a crash cart is enough to produce that.
	progress.SetOnClosed(func() {
		if settled.CompareAndSwap(false, true) {
			log.Printf("[dial] cancelled by operator: %s", n.Target())
			cancel()
		}
	})
	progress.Show()

	// Pointer so AuthPrompt can attach a vault credential name onto the node
	// used for the eventual inventory write.
	node := n

	opts := sessiondial.Options{
		Credentials:   h.lookup,
		HostKeyPrompt: h.promptHostKey,
		OnNewHostKey: func(host, keyType, fingerprint string) {
			log.Printf("[hostkey] trusted on first contact: %s %s %s", host, keyType, fingerprint)
		},
		AuthPrompt: func(prompt string, echo bool) (string, error) {
			return h.promptSecret(folder, oldLabel, &node, prompt, echo)
		},
		Log: log.Printf,
	}

	// Dial off the UI goroutine. A device slow to answer, or a host-key
	// prompt waiting on a click, must not freeze the window -- and the
	// prompt cannot be answered by a frozen one.
	go func() {
		defer cancel()

		tp, err := sessiondial.Connect(ctx, node, opts)
		if err != nil && auvik.ShouldTryTunnel(node, err, h.base.AuvikAutoTunnel) {
			log.Printf("[auvik] direct dial failed, trying tunnel: %v", err)
			tp, err = h.tryAuvikTunnelConnect(ctx, node, opts)
		}
		fyne.Do(func() {
			if !settled.CompareAndSwap(false, true) {
				// The operator cancelled and has moved on. Connect closes
				// an abandoned transport itself, so there is nothing to
				// clean up here and nothing to report -- raising an error
				// dialog now would hand them back their own decision as a
				// fault.
				return
			}
			progress.Hide()
			if err != nil {
				log.Printf("[dial] %v", err)
				// After a failed handshake (auth, host key, unreachable), put
				// the session form back up once the error is acknowledged so
				// the operator can fix the password or settings.
				title := node.Label()
				if title == "" {
					title = "Session"
				}
				ed := dialog.NewError(err, h.win)
				ed.SetOnClosed(func() {
					h.launchTerminalTitled(title, folder, node)
				})
				ed.Show()
				return
			}
			// Persist after dial so password-link Credential is kept, and so
			// New session (Add) still works via the tree's apply callback.
			if persist != nil {
				persist(node)
			}
			h.mountTerminal(node, tp)
		})
	}()
}

// connectSavingPassword stores a form-typed password in the vault (unlocking
// first if needed) before dialling, so AuthPrompt is not the only save path.
// The session form password field is yaml:"-" and was previously discarded.
func (h *host) connectSavingPassword(folder, oldLabel string, n sessions.Node, persist func(sessions.Node)) {
	pw := strings.TrimSpace(n.Password)
	if pw == "" {
		h.connect(folder, oldLabel, n, persist)
		return
	}
	go func() {
		node := n
		if h.ensureVaultUnlockedBlocking() {
			h.persistDialPassword(folder, oldLabel, &node, pw)
			// Keep password on the in-memory node for this dial; vault link
			// covers the next one.
		} else {
			log.Printf("[vault] could not unlock — connecting with typed password only (not saved)")
		}
		fyne.Do(func() {
			h.connect(folder, oldLabel, node, persist)
		})
	}()
}

// canAutoLogin reports whether double-click should dial without opening the form.
// Requires a vault credential that resolves to a password (or key material).
func (h *host) canAutoLogin(n sessions.Node) bool {
	if n.Transport == sessions.TransportTelnet || n.Transport == sessions.TransportSerial {
		return true
	}
	if strings.TrimSpace(n.Password) != "" {
		return true
	}
	ref := strings.TrimSpace(n.Credential)
	if ref == "" || h.lookup == nil {
		return false
	}
	c, err := h.lookup(ref)
	if err != nil {
		return false
	}
	return strings.TrimSpace(c.Password) != "" || strings.TrimSpace(c.KeyPath) != ""
}

// mountTerminal builds the session and hands it to the shell. UI goroutine.
func (h *host) mountTerminal(n sessions.Node, tp term.Transport) {
	// The settings dance, and the reason it is a dance: Settings are
	// process-wide, and the terminal widget reads FontSize and
	// ScrollbackLines at construction. So install this session's values,
	// build, wrap at an EXPLICIT size, and put the base back immediately --
	// the override object holds the size, so the next tab opening cannot
	// change this one's.
	cfg := ui.SettingsFor(h.base, n)
	ui.SetSettings(cfg)

	sess := ui.NewSession()
	// Before Attach: anti-idle is read when the transport is attached, so
	// setting it afterwards silently does nothing until a reconnect.
	ui.ApplySession(sess, n)
	// Do NOT call SetTerminalTheme(cfg.TerminalThemeName()) here. That pinned
	// every session to a copy of the global palette, so Settings → Terminal
	// theme never updated open (or even "inheriting") tabs. ApplySession
	// already sets a session-specific theme when the node names one; otherwise
	// the widget inherits CurrentSettings().
	content := ui.ThemedAt(sess, cfg)

	ui.SetSettings(h.base)

	if err := sess.Attach(tp); err != nil {
		log.Printf("[attach] %v", err)
		tp.Close()
		dialog.ShowError(err, h.win)
		return
	}

	sess.SetSendGate(func() (bool, string) {
		return h.allowSend()
	})
	sess.SetOnUserSend(func(b []byte) {
		if h.recorder != nil && h.recorder.Active() {
			h.recorder.Note(string(b))
		}
	})
	sess.SetOnOutputTee(func(b []byte) {
		if h.recorder != nil && h.recorder.Active() {
			h.recorder.NoteOutput(string(b))
		}
	})

	captureBtn := ui.TipIconButtonLow("Start or stop capturing this session to a transcript", theme.MediaRecordIcon(), nil)
	sessionMenuBtn := ui.TipButtonLabeled("Session", theme.DocumentIcon(), func() {})
	refreshCaptureTip := func() {
		if sess.IsLogging() {
			captureBtn.SetToolTip("Stop capture (transcript is recording)")
			captureBtn.SetIcon(theme.MediaStopIcon())
		} else {
			captureBtn.SetToolTip("Start capture — write a transcript from now on")
			captureBtn.SetIcon(theme.MediaRecordIcon())
		}
	}
	captureBtn.OnTapped = func() {
		_, msg := sess.ToggleLogging()
		refreshCaptureTip()
		if msg != "" {
			dialog.ShowInformation("Session Capture", msg, h.win)
		}
	}
	refreshCaptureTip()

	folder := ""
	if f, ok := h.folderFor(n); ok {
		folder = f
	}
	customer := sessions.CustomerOfFolder(folder)

	inst := h.shell.Open(ui.Mount{
		Kind:  ui.KindTerminal,
		Title: n.Label(),
		Applet: &termApplet{
			content:  content,
			sess:     sess,
			folder:   folder,
			customer: customer,
			host:     n.Host,
		},
		Focus:   sess,
		Actions: []fyne.CanvasObject{sessionMenuBtn, captureBtn},
		// The terminal resolves its own canvas for focus-on-click and for
		// its context menu, and the driver's cache cannot tell it that it
		// has moved. Without this a detached session goes deaf on the
		// first click and its right-click menu opens on the main window.
		OnCanvasChange: sess.SetHostCanvas,
		// The window is not its final size when it appears, and the
		// terminal's resize is debounced — without this a detached
		// session can tell the far end the minimum window size and a
		// full-screen application redraws into a corner.
		OnPlaced: sess.ResyncSize,
		OnClose: func() {
			if err := sess.Close(); err != nil {
				log.Printf("[close] %v", err)
			}
			// Skip UI refresh during app quit — fyne.Do into a dying
			// run loop has been observed to delay / stall Quit.
			if h.shuttingDown {
				return
			}
			fyne.Do(h.refreshOpsChrome)
		},
		// Only a live transport is worth stopping somebody for. A tab
		// whose session already died is a scrollback buffer, and
		// warning about those is how a person learns to click through
		// the warning without reading it.
		Busy: func() string {
			if sess.Connected() {
				return "connected"
			}
			return ""
		},
	})
	sessionMenuBtn.OnTapped = func() {
		h.showSessionMenu(sessionMenuBtn, sess, inst, n.Label())
	}
	inst.SetStatus(n.Target())

	// The terminal's right-click menu carries the same two bulk actions, for
	// the case where the pointer is already in the terminal. It is handed
	// functions rather than the shell: it must not learn what a tab is.
	sess.SetTabHooks(
		func() { h.confirmClose("Close all tabs?", h.shell.TabCount(), h.shell.CloseAll) },
		func() {
			h.confirmClose("Close other tabs?", h.shell.TabCount()-1, func() { h.shell.CloseOthers(inst) })
		},
		h.shell.TabCount,
	)

	// The paste confirmation offers to remember a pacing override, but only
	// when there is somewhere to remember it. A session dialled from a map
	// click or typed into the form is not in the inventory, so the hook stays
	// nil and the dialog hides the checkbox rather than promising a save that
	// has no file to land in.
	if folder, ok := h.folderFor(n); ok {
		sess.SetPasteRememberFunc(func(delayMs, baud int) {
			h.rememberPastePacing(folder, n, delayMs, baud)
		})
	}

	sess.SetStateChangeHandler(func(st ui.ConnectionState) {
		log.Printf("[state] %s %s", n.Label(), st)
		// The handler can fire from the read loop, so hop threads before
		// touching a widget.
		fyne.Do(func() {
			inst.SetStatus(n.Target() + " — " + st.String())
			h.refreshOpsChrome()
			if st == ui.StateConnected {
				h.shell.EnsureTerminalFocus()
			}
		})
	})
	sess.SetErrorHandler(func(err error) { log.Printf("[error] %s: %v", n.Label(), err) })

	// Attach emits StateConnected before the handler above is registered, so
	// the bottom button dock would stay hidden until disconnect.
	h.refreshOpsChrome()
	// Force keyboard onto this terminal once the Connecting overlay is gone.
	// Without this, settle can finish under the dialog and leave focus on the
	// session tree — a live prompt that ignores every key.
	h.shell.EnsureTerminalFocus()

	// Do not Canvas.Focus / GrabFocus here. The Connecting dialog's Hide may
	// still leave an overlay on this frame; focusing under an overlay lands
	// keys in the wrong focus manager and the terminal looks dead. Shell
	// settle() and EnsureTerminalFocus wait for overlays to clear first.
}

// termApplet is the whole adapter. A terminal has no redraw loop to gate -- it
// repaints from its read loop -- so Start and Stop are genuinely nothing, and
// the teardown that matters is the Mount's OnClose.
type termApplet struct {
	content  fyne.CanvasObject
	sess     *ui.Session
	folder   string
	customer string
	host     string
}

func (t *termApplet) Content() fyne.CanvasObject { return t.content }
func (t *termApplet) Start()                     {}
func (t *termApplet) Stop()                      {}

func (t *termApplet) FolderPath() string {
	if t == nil {
		return ""
	}
	return t.folder
}
func (t *termApplet) CustomerName() string {
	if t == nil {
		return ""
	}
	return t.customer
}

// SendBytes implements the button-bar / send-to-all hook.
func (t *termApplet) SendBytes(b []byte) {
	if t == nil || t.sess == nil || len(b) == 0 {
		return
	}
	if !t.sess.SendUser(b) {
		log.Printf("[button] send failed (%d bytes) connected=%v", len(b), t.sess.Connected())
	}
}

// --- crawl -----------------------------------------------------------------

func (h *host) launchCrawl() {
	h.launchCrawlPrefill("", nil)
}

func (h *host) launchCrawlPrefill(customer string, seedHosts []string) {
	h.lastCrawl.Params.VaultPath = h.runVaultPath()
	// Maps always live under app home (~/.pathfinderssh/maps/…), not beside a
	// relocated vault, so the Map picker and crawl writer agree.
	home := ui.GetAppHome()
	root := product.CustomersRoot
	if h.tree != nil {
		tr := h.tree.Tree()
		if _, err := (&tr).EnsureMSPLayout(); err != nil {
			log.Printf("[sessions] MSP layout: %v", err)
		}
		h.tree.SetTree(tr)
	}
	if c := strings.TrimSpace(customer); c != "" {
		h.mapCustomer = c
		_, _ = ui.EnsureCustomerMapsDir(home, c)
	}
	ui.ShowCrawlWizard(h.win, ui.CrawlWizardOptions{
		Prev:             h.lastCrawl,
		Sessions:         h.crawlSeedOptions(),
		Customers:        h.customerNames(),
		CustomersRoot:    root,
		HomeDir:          home,
		PrefillCustomer:  customer,
		PrefillSeedHosts: seedHosts,
		CreateCustomer: func(name string) (string, error) {
			if h.tree == nil {
				return "", fmt.Errorf("no session tree loaded")
			}
			tr := h.tree.Tree()
			path, err := tr.CreateCustomer(root, name)
			if err != nil {
				return "", err
			}
			h.tree.SetTree(tr)
			h.saveTree(tr)
			if _, err := ui.EnsureCustomerMapsDir(home, name); err != nil {
				log.Printf("[maps] create customer maps dir: %v", err)
			}
			h.mapCustomer = name
			return path, nil
		},
	}, func(l ui.CrawlLaunch) {
		if !l.ManualCreds {
			l.Params.VaultPath = h.runVaultPath()
		}
		h.lastCrawl = l
		if c := ui.InferCustomerFromMapsPath(l.MapPath); c != "" {
			h.mapCustomer = c
		}
		h.startCrawl(l)
	})
}

// importCustomerCrawlCSV walks new-customer seed import: template → CSV → sessions → optional crawl.
func (h *host) importCustomerCrawlCSV() {
	home := ui.GetAppHome()
	if h.vaultPath != "" {
		home = filepath.Dir(h.vaultPath)
	}
	root := product.CustomersRoot
	ui.ShowCustomerCrawlImportWizard(h.win, ui.CustomerCrawlImportOptions{
		ExistingCustomers: h.customerNames(),
		HomeDir:           home,
		CreateCustomer: func(name string) (string, error) {
			if h.tree == nil {
				return "", fmt.Errorf("no session tree loaded")
			}
			tr := h.tree.Tree()
			if _, err := (&tr).EnsureMSPLayout(); err != nil {
				return "", err
			}
			path, err := tr.CreateCustomer(root, name)
			if err != nil {
				if strings.Contains(strings.ToLower(err.Error()), "already exists") {
					h.tree.SetTree(tr)
					return sessions.CustomerPath(root, name), nil
				}
				return "", err
			}
			h.tree.SetTree(tr)
			h.saveTree(tr)
			_, _ = ui.EnsureCustomerMapsDir(home, name)
			return path, nil
		},
	}, func(imp ui.CustomerCrawlImport) {
		if h.tree == nil {
			dialog.ShowError(fmt.Errorf("no session tree loaded"), h.win)
			return
		}
		tr := h.tree.Tree()
		var folders []sessions.Folder
		for rel, nodes := range imp.Folders {
			path := sessions.JoinPath(root, imp.Customer, rel)
			folders = append(folders, sessions.Folder{Name: path, Sessions: nodes})
		}
		sum := tr.ImportFolders(folders)
		h.mapCustomer = imp.Customer
		_, _ = ui.EnsureCustomerMapsDir(home, imp.Customer)
		h.applyImport(tr, sessions.FormatNative, sum)
		if imp.StartCrawl {
			h.launchCrawlPrefill(imp.Customer, imp.SeedHosts)
		}
	})
}

func (h *host) customerNames() []string {
	if h.tree == nil {
		return nil
	}
	tr := h.tree.Tree()
	return tr.ListCustomers(product.CustomersRoot)
}

// crawlSeedOptions lists inventory hosts tagged with their customer (folder
// directly under 3_Customers).
func (h *host) crawlSeedOptions() []ui.CrawlSeedOption {
	if h.tree == nil {
		return nil
	}
	root := product.CustomersRoot
	seen := map[string]bool{}
	var out []ui.CrawlSeedOption
	h.tree.Tree().WalkSessions(func(folder string, n sessions.Node) {
		n = n.Normalize()
		if n.Transport == sessions.TransportSerial {
			return
		}
		host := strings.TrimSpace(n.Host)
		if host == "" || seen[strings.ToLower(host)] {
			return
		}
		seen[strings.ToLower(host)] = true
		customer := customerOfPath(root, folder)
		label := n.Label()
		if label == "" || strings.EqualFold(label, host) {
			label = host
		} else {
			label = label + " (" + host + ")"
		}
		out = append(out, ui.CrawlSeedOption{Label: label, Host: host, Customer: customer})
	})
	return out
}

// customerOfPath returns the customer leaf under root for a folder path.
func customerOfPath(root, folder string) string {
	parts := sessions.SplitPath(folder)
	rootParts := sessions.SplitPath(root)
	if len(parts) <= len(rootParts) {
		return ""
	}
	for i := range rootParts {
		if !strings.EqualFold(parts[i], rootParts[i]) {
			return ""
		}
	}
	return parts[len(rootParts)]
}

func (h *host) startCrawl(l ui.CrawlLaunch) {
	l.MapPath = strings.TrimSpace(ui.ExpandHome(l.MapPath))
	if l.MapPath == "" {
		cust := strings.TrimSpace(h.mapCustomer)
		if cust == "" {
			cust = "Unassigned"
		}
		l.MapPath = filepath.Join(ui.CustomerMapsDir(ui.GetAppHome(), cust), time.Now().Format("crawl-2006-01-02.json"))
		log.Printf("[crawl] MapPath was empty; defaulting to %s", l.MapPath)
	}
	if c := ui.InferCustomerFromMapsPath(l.MapPath); c != "" {
		h.mapCustomer = c
	}
	h.lastCrawl = l

	run := crawlrun.New()
	view := ui.NewCrawlView(run)

	// The loop the shell exists to close: a device in a crawl result is a
	// device you can open a session to without retyping it.
	view.OnConnect = func(d crawlrun.DeviceRow) {
		n := sessions.Defaults()
		n.Name = d.Display()
		n.Host = d.Name
		n.Transport = sessions.TransportSSH
		h.launchTerminal(n)
	}

	ctx, cancel := context.WithCancel(context.Background())

	var stop *widget.Button
	stop = widget.NewButtonWithIcon("Stop", theme.MediaStopIcon(), func() {
		cancel()
		stop.SetText("Stopping…")
		stop.Disable()
	})

	title := "crawl"
	if len(l.Params.Seeds) > 0 {
		title = "crawl " + l.Params.Seeds[0]
	}

	// running is this crawl's own answer to "is it still doing something",
	// asked at shutdown. The Stop button's state is not that answer: it is
	// disabled by cancellation as well as by completion, and a crawl that
	// has been asked to stop is still dialing until the in-flight devices
	// drain.
	var running atomic.Bool

	inst := h.shell.Open(ui.Mount{
		Kind:    ui.KindCrawl,
		Title:   title,
		Applet:  view,
		Actions: []fyne.CanvasObject{stop},
		Busy: func() string {
			if running.Load() {
				return "running"
			}
			return ""
		},
		// Cancel on close as well as on Stop. Closing the tab of a
		// running crawl and leaving it dialing devices in the background
		// is the kind of thing that is only noticed by the lockout
		// counter on somebody's TACACS server.
		OnClose: cancel,
	})
	inst.SetStatus(fmt.Sprintf("%d starting device(s) · depth %d", len(l.Params.Seeds), l.Params.Depth))

	if l.LastRun != "" {
		if prev, err := crawlrun.LoadSnapshot(l.LastRun); err == nil {
			view.Compare(prev)
		} else {
			inst.SetStatus("comparison unavailable: " + err.Error())
		}
	}

	logf := h.logfIf(l.Verbose)

	// outputs accumulates what this run actually wrote, and is read by the
	// deferred summary. A crawl that produced no file has to SAY so: the
	// map and the snapshot are the durable half of the work, the counters
	// look identical either way, and a blank path field reads exactly like
	// a write that failed.
	var outputs []string
	running.Store(true)
	go func() {
		defer func() {
			// First, so that every later line -- including a dialog
			// raised from the deferred summary -- already reports
			// the run as over.
			running.Store(false)
			run.Finish()
			c := run.Counts()
			summary := fmt.Sprintf(
				"%d reached · %d failed · %d not dialed · %d new host keys · %.2f tries/device",
				c.Reached, c.Failed, c.NotDialed, c.NewHostKeys, c.AttemptsPerReached())
			if len(outputs) > 0 {
				summary += "  ·  " + strings.Join(outputs, ", ")
			} else {
				summary += "  ·  nothing written"
			}
			fyne.Do(func() {
				stop.SetText("Done")
				stop.Disable()
				inst.SetStatus(summary)
			})
		}()

		built, err := crawldial.Build(l.Params, crawldial.Options{
			// The OPEN vault, not a path. Build must never try to
			// unlock one from here: it runs on this goroutine and
			// has nowhere to ask for a master password.
			//
			// Nil for a manual run, and not merely unused: Build's
			// no-credentials guard tests this field, so leaving an
			// open vault here would let a manual run with every
			// field blank past the guard and fail device by device
			// instead of before the first dial.
			Vault: h.runVault(l.ManualCreds),
			Static: crawldial.StaticCreds{
				Username: l.Auth.Username, Password: l.Auth.Password, KeyPath: l.Auth.KeyPath,
			},
			Log:     crawler.Logf(logf),
			CredLog: logf,
			Emit:    run.Emit(),
		})
		if err != nil {
			// A dialog, not just the status line. The commonest
			// cause is a locked vault, and that is a thing to fix
			// and retry rather than a result to read.
			h.reportRunError(inst, err)
			return
		}
		defer built.Close()

		devices := built.Crawler.CrawlContext(ctx, l.Params.Seeds)

		// The same two writes cmd/crawl makes, for the same reasons: the
		// fold is the only place SysName reaches the binding store, and
		// the snapshot is what makes the NEXT run's comparison real.
		crawldial.Fold(built.Bindings, devices, l.Params.Domains, crawler.Logf(logf))

		// Both writes report, and a failure raises a dialog rather than
		// a log line. logf is a no-op without -v, so the previous
		// version of this could fail to write the map and end with a
		// summary that looked like a clean run.
		if l.MapPath != "" {
			if err := writeMap(devices, l.Params, l.MapPath); err != nil {
				log.Printf("pathfinder: write map: %v", err)
				h.reportRunError(inst, err)
			} else {
				outputs = append(outputs, "map → "+l.MapPath)
				mapPath := l.MapPath
				fyne.Do(func() {
					h.mapDir = filepath.Dir(mapPath)
					if c := ui.InferCustomerFromMapsPath(mapPath); c != "" {
						h.mapCustomer = c
					}
					dialog.ShowConfirm("Map saved",
						"Topology map written to:\n"+mapPath+"\n\nOpen it now?",
						func(ok bool) {
							if !ok {
								return
							}
							files, err := mapweb.ListMaps(filepath.Dir(mapPath))
							if err != nil {
								dialog.ShowError(err, h.win)
								return
							}
							base := filepath.Base(mapPath)
							for _, f := range files {
								if strings.EqualFold(f.Name, base) {
									h.openMap(f)
									return
								}
							}
							// Fresh write may not be listed if parse rejected empty —
							// open by path directly.
							h.openMap(mapweb.MapFile{Path: mapPath, Name: base})
						}, h.win)
				})
			}
		} else {
			log.Printf("[crawl] no MapPath — topology not saved")
			fyne.Do(func() {
				dialog.ShowInformation("No map file", "This crawl had no map output path, so nothing was written under maps/.", h.win)
			})
		}
		if l.SaveRun != "" {
			if err := run.Snapshot(l.Params.Seeds, l.Params.Domains).Save(l.SaveRun); err != nil {
				logf("pathfinder: %v", err)
				h.reportRunError(inst, err)
			} else {
				outputs = append(outputs, "run → "+l.SaveRun)
			}
		}
	}()
}

// writeMap renders the topology and writes it, reporting every failure.
//
// The marshal error was previously discarded by an `if err == nil` with no
// else, so a map that could not be encoded wrote nothing and said nothing.
func writeMap(devices []*topo.Device, p crawlrun.Params, path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("map path is empty")
	}
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return fmt.Errorf("map path is a directory — need a .json filename under it")
	}
	m := topo.Generate(devices, crawldial.MapOptions(p))
	data, err := topo.MarshalMap(m)
	if err != nil {
		return fmt.Errorf("render map: %w", err)
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// reportRunError puts a failure in front of the user instead of leaving it in
// the status line.
//
// Called from a run goroutine, so it hops threads. A dialog because a status
// line is easy to miss and, at the end of a run, about to be overwritten by
// the final counts — and because these are all things to fix and retry rather
// than results to read.
// --- map -------------------------------------------------------------------

// launchMap opens the per-customer map picker.
func (h *host) launchMap() {
	home := ui.GetAppHome()
	root := ui.MapsRootDir(home)
	_ = os.MkdirAll(root, 0o755)

	initial := h.mapCustomer
	if initial == "" && h.lastCrawl.MapPath != "" {
		initial = ui.InferCustomerFromMapsPath(h.lastCrawl.MapPath)
	}
	dir := h.mapDir
	if dir == "" && h.lastCrawl.MapPath != "" {
		dir = filepath.Dir(h.lastCrawl.MapPath)
	}

	ui.ShowCustomerMapDialog(h.win, ui.MapDialogOptions{
		MapsRoot:        root,
		Customers:       h.customerNames(),
		InitialCustomer: initial,
		InitialDir:      dir,
	}, func(l ui.MapLaunch) {
		h.mapDir = l.Dir
		h.mapCustomer = l.Customer
		h.openMap(l.File)
	})
}

// openMap loads a map into the viewer and opens it in the browser.
//
// Rendering lives outside this process on purpose: a browser already has a
// graph engine, a zoom, a scroll and a print dialog, and none of that has to
// be built or maintained here. What stays in the application is the part only
// the application can do — knowing which device a node is, and opening a
// session to it.
func (h *host) openMap(f mapweb.MapFile) {
	data, err := os.ReadFile(f.Path)
	if err != nil {
		dialog.ShowError(err, h.win)
		return
	}

	if h.maps == nil {
		srv, err := mapweb.Serve(mapweb.Options{
			OnConnect: h.mapConnect,
			Log:       h.logf(),
		})
		if err != nil {
			dialog.ShowError(fmt.Errorf("start the map viewer: %w", err), h.win)
			return
		}
		h.maps = srv
	}

	if err := h.maps.SetMap(f.Name, data); err != nil {
		dialog.ShowError(fmt.Errorf("%s: %w", f.Name, err), h.win)
		return
	}

	u, err := url.Parse(h.maps.URL())
	if err != nil {
		dialog.ShowError(err, h.win)
		return
	}
	if err := h.app.OpenURL(u); err != nil {
		// Not fatal: the server is up and the URL is valid, so give it
		// to the person rather than dropping the map on the floor.
		dialog.ShowInformation("Open this in a browser",
			h.maps.URL(), h.win)
	}
}

// mapConnect is the click-to-session loop, arriving from the browser.
//
// Two things about it are deliberate. It runs on the map server's HTTP
// goroutine, so everything it touches goes through fyne.Do. And it opens the
// session DIALOG rather than dialing: a click that arrives over HTTP should
// end in a form the person confirms, not in an SSH connection nobody asked
// for. That is also what makes the loopback surface safe to leave running.
func (h *host) mapConnect(n mapweb.NodeRef) {
	node := sessions.Defaults()
	node.Name = n.Name
	node.Transport = sessions.TransportSSH

	// Prefer the address. A neighbour-reported name is often not resolvable
	// from here, and his lab has no DNS at all — the IP fallback is the
	// path that actually works.
	if n.IP != "" {
		node.Host = n.IP
	} else {
		node.Host = n.Name
	}

	fyne.Do(func() {
		folder := ""
		title := "Map: " + n.Name
		if c := strings.TrimSpace(h.mapCustomer); c != "" {
			folder = sessions.JoinPath(product.CustomersRoot, c)
			title = c + ": " + n.Name
		}
		h.launchTerminalTitled(title, folder, node)
		h.win.RequestFocus()
	})
}

func (h *host) reportRunError(inst *ui.Instance, err error) {
	fyne.Do(func() {
		dialog.ShowError(err, h.win)
		inst.SetStatus("⚠  " + err.Error())
	})
}

// --- capture ---------------------------------------------------------------

// launchSearch collects a query and runs it against a capture store.
//
// The store field is seeded from the last capture launch in this process, then
// from the -store flag. A search dialog that opens with a blank store path
// makes the operator type a path to a folder the app already knows about,
// which is how a feature ends up unused.
func (h *host) launchSearch() {
	if h.lastSearch.StorePath == "" {
		h.lastSearch.StorePath = h.lastCapture.Params.StorePath
	}
	ui.ShowSearchDialog(h.win, h.lastSearch, capturedial.KnownTypes(), func(l ui.SearchLaunch) {
		h.lastSearch = l
		h.startSearch(l)
	})
}

// startSearch opens one search tab and keeps it usable for further searches.
//
// The tab outlives any single run. What made the first version awkward was
// that the run WAS the tab: the query lived in a modal reached only from the
// main window's toolbar, every run opened another tab, and the Stop button
// doubled as the completion signal by disabling itself — which is unreadable
// by design, since a greyed control means "you cannot press this" rather than
// "your answer is ready". A detached window could not reach the toolbar at
// all, so the only way to ask a second question was to close the window.
//
// So the two jobs are split. The progress bar in the view says whether a scan
// is running, and one always-enabled button is the control: Stop while a scan
// is in flight, New search when it is not.
func (h *host) startSearch(l ui.SearchLaunch) {
	// A dialog rather than a status line, and before any tab is opened:
	// there is nothing yet to put a status on, and an empty search tab
	// with a quiet failure in it reads as a store with nothing in it.
	store, err := capture.OpenFileStore(l.StorePath)
	if err != nil {
		dialog.ShowError(err, h.win)
		return
	}
	matcher, err := storesearch.NewLiteral(l.Query, l.CaseSensitive)
	if err != nil {
		dialog.ShowError(err, h.win)
		return
	}

	// store and matcher are built here only to fail fast: a bad path or an
	// empty query becomes a dialog before any tab exists, rather than an
	// empty search tab that reads like a store with nothing in it. start()
	// builds its own from whichever launch it is given.
	_ = matcher

	view := ui.NewSearchView(store)
	// The store keys on the canonical device name, which is the same
	// identity the binding store and the session tree use — so a hit
	// becomes a session through exactly the path a map click already
	// takes. There is no address here: capture files under the name, and
	// resolving it is the dialer's job, not the view's.
	view.OnConnect = func(device string) { h.mapConnect(mapweb.NodeRef{Name: device}) }

	// run holds the state one tab carries across many searches. The cancel
	// func is per-run and must be replaced on every start, which is the
	// whole reason this is a struct and not a closed-over variable: the
	// Stop button is built once and has to cancel whichever run is current
	// when it is pressed, not the first one.
	var run struct {
		mu      sync.Mutex
		cancel  context.CancelFunc
		running bool
		last    ui.SearchLaunch
		// gen identifies the current run. A finishing goroutine
		// compares its own generation against this before touching the
		// view, because a superseded scan still returns — with whatever
		// partial result it had — and installing that would blank the
		// hits its replacement has already put on screen.
		gen int
	}
	run.last = l

	var action *widget.Button
	var inst *ui.Instance

	idle := func() {
		action.SetIcon(theme.SearchIcon())
		action.SetText("New search")
		action.Enable()
	}
	busy := func() {
		action.SetIcon(theme.MediaStopIcon())
		action.SetText("Stop")
		action.Enable()
	}

	// start executes one search into the existing view. Callable repeatedly.
	var start func(ui.SearchLaunch)
	start = func(l ui.SearchLaunch) {
		st, err := capture.OpenFileStore(l.StorePath)
		if err != nil {
			dialog.ShowError(err, h.win)
			return
		}
		m, err := storesearch.NewLiteral(l.Query, l.CaseSensitive)
		if err != nil {
			dialog.ShowError(err, h.win)
			return
		}

		ctx, cancel := context.WithCancel(context.Background())
		run.mu.Lock()
		// Cancel anything still in flight before adopting the new run.
		// Without this a slow scan and its replacement both reach
		// SetResult, and the later answer can be overwritten by the
		// earlier one arriving last.
		if run.cancel != nil {
			run.cancel()
		}
		run.gen++
		gen := run.gen
		run.cancel, run.running, run.last = cancel, true, l
		run.mu.Unlock()

		view.Reset()
		view.SetMatcher(m)
		view.SetRunning(true)
		busy()
		inst.SetTitle("search " + l.Query)
		inst.SetStatus(filepath.Base(l.StorePath))

		go func() {
			res, err := storesearch.Search(ctx, st, m, storesearch.Options{
				Types:      l.Types,
				Limit:      l.Limit,
				OnProgress: view.SetProgress,
			})

			run.mu.Lock()
			current := gen == run.gen
			if current {
				run.running = false
			}
			run.mu.Unlock()

			// A superseded run owns nothing. Its replacement has
			// already reset the view, set the title and taken the
			// button; every line below would undo one of those.
			if !current {
				return
			}

			view.SetRunning(false)
			// A cancelled search still carries the hits it found
			// before it was stopped, so the result is installed
			// either way and the cancellation is reported as status
			// rather than as failure.
			view.SetResult(res)
			if err != nil && ctx.Err() == nil {
				view.SetError(err)
			}
			fyne.Do(func() {
				idle()
				if err == nil || ctx.Err() != nil {
					inst.SetStatus(res.Summary())
				}
			})
		}()
	}

	action = widget.NewButtonWithIcon("Stop", theme.MediaStopIcon(), func() {
		run.mu.Lock()
		running, last := run.running, run.last
		cancel := run.cancel
		run.mu.Unlock()

		if running {
			if cancel != nil {
				cancel()
			}
			return
		}
		// Idle: ask for the next query, seeded with this tab's last
		// one, and run it here instead of opening another tab.
		ui.ShowSearchDialog(h.win, last, capturedial.KnownTypes(), func(next ui.SearchLaunch) {
			h.lastSearch = next
			start(next)
		})
	})

	inst = h.shell.Open(ui.Mount{
		Kind: ui.KindSearch,
		// The query IS the title. Several searches coexist the way two
		// crawls already do, and a tab strip of tabs all called
		// "search" is a tab strip nobody can navigate.
		Title:   "search " + l.Query,
		Applet:  view,
		Actions: []fyne.CanvasObject{action},
		OnClose: func() {
			run.mu.Lock()
			c := run.cancel
			run.mu.Unlock()
			if c != nil {
				c()
			}
		},
		// A scan in flight is worth mentioning; a tab holding the hits
		// from a finished one is not. The lock is the same one every
		// other reader of this struct takes -- the answer is a bool
		// read, so it is held for no longer than that.
		Busy: func() string {
			run.mu.Lock()
			defer run.mu.Unlock()
			if run.running {
				return "searching"
			}
			return ""
		},
	})

	start(l)
}

func (h *host) launchCapture() {
	h.lastCapture.Params.VaultPath = h.runVaultPath()
	// Offer the inventory this window is already showing. Without it the
	// session field opens blank and the tree is only reachable by typing
	// a path to a file that is on screen — and a device source nobody
	// finds is a device source nobody uses. It stays only an offer: no
	// pattern means no sessions are selected.
	if h.lastCapture.Params.SessionFile == "" {
		h.lastCapture.Params.SessionFile = h.sessionsPath
	}
	ui.ShowCaptureDialog(h.win, h.lastCapture, capturedial.KnownTypes(), func(l ui.CaptureLaunch) {
		if !l.ManualCreds {
			l.Params.VaultPath = h.runVaultPath()
		}
		h.lastCapture = l
		h.startCapture(l)
	})
}

func (h *host) startCapture(l ui.CaptureLaunch) {
	run := capturerun.New()

	// A browser without a run is first class: opening the store to read
	// last night's config is the expected use once captures are scheduled.
	var browser capture.Browser
	if l.Params.StorePath != "" {
		if store, err := capture.OpenFileStore(l.Params.StorePath); err == nil {
			browser = store
		} else {
			log.Printf("[store] %s: %v", l.Params.StorePath, err)
		}
	}

	view := ui.NewCaptureView(run, browser)

	ctx, cancel := context.WithCancel(context.Background())

	var stop *widget.Button
	stop = widget.NewButtonWithIcon("Stop", theme.MediaStopIcon(), func() {
		cancel()
		stop.SetText("Stopping…")
		stop.Disable()
	})

	title := "capture"
	if l.Params.StorePath != "" {
		title = "capture " + filepath.Base(l.Params.StorePath)
	}

	// Set only where the run actually starts, below: a capture opened to
	// browse a store has no devices to visit and returns before then, and
	// a browser is not something to be warned about.
	var running atomic.Bool

	inst := h.shell.Open(ui.Mount{
		Kind:    ui.KindCapture,
		Title:   title,
		Applet:  view,
		Actions: []fyne.CanvasObject{stop},
		OnClose: cancel,
		Busy: func() string {
			if running.Load() {
				return "running"
			}
			return ""
		},
	})

	if !l.Params.HasDeviceSource() {
		msg := "no devices — browsing the store"
		if browser == nil {
			msg = "no devices and no store"
		}
		inst.SetStatus(msg)
		stop.Disable()
		return
	}

	logf := h.logfIf(l.Verbose)
	running.Store(true)
	go func() {
		defer func() {
			// First, so a dialog raised from anything below already
			// reports the run as over.
			running.Store(false)
			run.Finish()
			c := run.Counts()
			fyne.Do(func() {
				stop.SetText("Done")
				stop.Disable()
				inst.SetStatus(fmt.Sprintf(
					"%d stored · %d unchanged · %d not applicable · %d failed · %d new host key(s)",
					c.Stored, c.Unchanged, c.NotApplicable, c.Failed, c.NewHostKeys))
			})
		}()

		built, err := capturedial.Build(l.Params, capturedial.Options{
			Vault: h.runVault(l.ManualCreds),
			Static: capturedial.StaticCreds{
				Username: l.Auth.Username, Password: l.Auth.Password, KeyPath: l.Auth.KeyPath,
			},
			Log:     logf,
			CredLog: logf,
			Emit:    run.Emit(),
		})
		if err != nil {
			h.reportRunError(inst, err)
			return
		}
		defer built.Close()

		// What is about to happen, before it happens. A capture is the
		// case where being told afterwards is too late, and the CGNAT
		// notes in particular are decisions rather than trivia.
		plan := fmt.Sprintf("%d device(s) x %d type(s) -> %s",
			len(built.Devices), len(built.Specs), l.Params.StorePath)
		if len(built.Skipped) > 0 {
			// Sessions a pattern matched that capture cannot visit.
			// Without this the shell shows a smaller device count
			// than the pattern matched and says nothing about the
			// difference, which reads as the pattern being wrong.
			plan += fmt.Sprintf("  ·  %d matched session(s) skipped: %s",
				len(built.Skipped), strings.Join(capturedial.SkippedLines(built.Skipped), "; "))
		}
		if len(built.Notes) > 0 {
			var notes []string
			for id, n := range built.Notes {
				notes = append(notes, id+": "+n)
			}
			plan += "  ·  " + strings.Join(notes, "; ")
		}
		fyne.Do(func() { inst.SetStatus(plan) })

		built.Engine.Capture(ctx, built.Devices)
	}()
}

// --- vault and prompts -----------------------------------------------------

// unlockQuiet tries the keyring and the environment and gives up silently.
func (h *host) unlockQuiet() {
	if h.vaultPath == "" {
		h.refreshVault()
		return
	}
	v, err := vaultcli.OpenQuiet(h.vaultPath)
	if err != nil {
		if !errors.Is(err, vaultcli.ErrNeedsPassword) {
			log.Printf("[vault] %v", err)
		}
		h.refreshVault()
		return
	}
	h.adopt(v)
}

// promptVaultUnlockIfNeeded asks once at startup when an encrypted vault exists
// but the OS keyring did not unlock it. Choosing Remember stores the password
// for silent unlock next launch.
func (h *host) promptVaultUnlockIfNeeded() {
	if h.vault != nil {
		return
	}
	path := h.vaultPath
	if _, err := os.Stat(path); err != nil {
		if def := vaultcli.DefaultPath(); def != "" {
			if _, err2 := os.Stat(def); err2 == nil {
				path = def
				h.vaultPath = def
			} else {
				return
			}
		} else {
			return
		}
	}
	h.showUnlockVaultDialog(true)
}

// version is stamped at build time:
//
//	go build -ldflags "-X main.version=$(git describe --tags --always --dirty)"
//
// Left as "dev" otherwise, which is what an unstamped local build honestly is.
var version = "0.93"

// showAbout opens the About box.
//
// The paths are here rather than in the dialog because this is the only place
// that knows which vault and which inventory this run resolved to — and those
// two answers are the first thing worth having when somebody reports that a
// credential or a session "isn't there".
// showSettings opens the settings dialog and applies what comes back.
//
// Two things happen on save and they are deliberately separate. The chrome
// variant is applied to the LIVE app, because it is the one setting a person
// can see is wrong the moment they change it. Everything else is installed as
// the new base and takes effect on the next tab: the terminal reads its font
// size and scrollback when it constructs its grid, and reaching into the open
// ones would mean re-measuring a running session's geometry underneath the
// device on the far end.
//
// The write is last and its failure is reported, not swallowed. A settings
// dialog that appears to work and silently does not persist is worse than one
// that says the disk is full.
func (h *host) showSettings() {
	// Prefer disk so a prior save is not overwritten by a stale in-memory base
	// (e.g. after another code path wrote settings.json).
	if loaded, err := ui.LoadSettings(h.settingsPath); err == nil {
		h.base = loaded
		ui.SetSettings(loaded)
	}
	ui.ShowSettings(h.win, ui.SettingsFormOptions{
		Settings:               h.base,
		Paths:                  h.hostPaths(),
		MSPIntegrationsEnabled: h.mspIntegrationsEnabled(),
		MSPActions:             h.mspIntegrationActions(),
		OnManageVault:          h.manageVault,
		OnImportSessions:       h.importSessions,
		OnExportSessions:       h.exportSessions,
		OnImportMap:            h.importMap,
		OnImportCRT:            h.importSecureCRT,
		OnImportCrawlCSV:       h.importCustomerCrawlCSV,
		OnHelpQuickstart:       func() { ui.ShowHelp(h.win, helpdoc.TopicQuickstart) },
		OnHelpContents:         func() { ui.ShowHelp(h.win, "") },
		OnAbout:                h.showAbout,
		OnSave: func(s ui.Settings) {
			// Always reinstall the chrome theme so TreeExpandStyle icon remaps
			// take effect (Icon() reads CurrentSettings).
			ui.ApplyAppTheme(h.app, s.AppVariant())
			wasAddon := h.base.TroubleshootAddon
			h.base = s
			ui.SetSettings(s)
			if s.TroubleshootAddon && !wasAddon {
				h.cursorPaneVisible = true
			}
			if !s.TroubleshootAddon {
				h.cursorPaneVisible = false
			}
			h.refreshTroubleshootChrome()
			// Persist synchronously. Async save looked responsive but lost
			// theme changes when the app locked up or quit before the write.
			if err := ui.SaveSettings(h.settingsPath, s); err != nil {
				dialog.ShowError(err, h.win)
				return
			}
			// Re-read so h.base matches what will load on next start.
			if loaded, err := ui.LoadSettings(h.settingsPath); err == nil {
				h.base = loaded
				ui.SetSettings(loaded)
				ui.ApplyAppTheme(h.app, loaded.AppVariant())
			}
			if h.tree != nil {
				h.tree.RefreshView()
			}
			h.refreshOpenTerminalThemes()
			h.refreshOpenTerminalScrollback()
			if h.auvikTunnels != nil {
				h.auvikTunnels.BinPath = h.base.AuvikTunnelPath
			}
			h.startAuvikPeriodicSync()
		},
	})
}

// refreshOpenTerminalScrollback applies Settings.ScrollbackLines to open
// terminals (session-specific overrides already baked into the widget at open
// keep their higher value if they set one — we only raise/set the global cap).
func (h *host) refreshOpenTerminalScrollback() {
	n := h.base.ScrollbackLines
	if n <= 0 {
		return
	}
	for _, inst := range h.shell.Instances() {
		ta, ok := inst.Applet().(*termApplet)
		if !ok || ta.sess == nil {
			continue
		}
		ta.sess.SetMaxHistoryLines(n)
	}
}

// refreshOpenTerminalThemes rebuilds palettes on open terminals after Settings
// save. Sessions with an empty override inherit the new global theme; sessions
// with their own node theme keep that override but still refresh the cache.
func (h *host) refreshOpenTerminalThemes() {
	for _, inst := range h.shell.Instances() {
		ta, ok := inst.Applet().(*termApplet)
		if !ok || ta.sess == nil {
			continue
		}
		// Re-apply override (or "" to inherit CurrentSettings) so the palette
		// cache and TextGrid rebuild against the theme just saved.
		ta.sess.SetTerminalTheme(ta.sess.TerminalThemeName())
	}
}

// hostPaths are the files this run resolved, for the settings dialog's
// read-only Paths page and for the About box. Only the host knows them: the ui
// package has no business deciding where a vault lives.
func (h *host) hostPaths() []ui.AboutDetail {
	vaultPath := h.vaultPath
	if h.vault != nil {
		vaultPath = h.vault.Path()
	}
	return []ui.AboutDetail{
		{Label: "License", Value: "GNU GPL v3.0 — free software; you may redistribute under the same terms"},
		{Label: "Based on", Value: "PathfinderSSH by Scott Peterman (https://github.com/scottpeterman/pathfinderssh)"},
		{Label: "MSP source", Value: "https://github.com/AgenticOp-io/pathfinderssh-msp"},
		{Label: "Vault", Value: vaultPath},
		{Label: "Sessions", Value: h.sessionsPath},
		{Label: "Captures", Value: h.lastCapture.Params.StorePath},
	}
}

// flagWasSet reports whether a flag appeared on the command line, as opposed
// to holding its default. flag has no other way to tell the two apart, and the
// difference decides whether the command line overrides the settings file.
func flagWasSet(name string) bool {
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func (h *host) showAbout() {
	ui.ShowAbout(h.win, ui.AboutInfo{
		Name:    ui.DefaultAppName,
		Tagline: "MSP fork of PathfinderSSH (GPL-3.0). Upstream by Scott Peterman.",
		Version: version,
		Details: h.hostPaths(),
	})
}

// manageVault opens the credential manager against the session's vault.
//
// It refuses rather than prompting when the vault is locked: unlocking is
// showVaultDialog's job and it is the only place a master password is asked
// for, which is what keeps that question out of every other code path.
func (h *host) manageVault() {
	if h.vault == nil {
		dialog.ShowInformation("Vault",
			"No vault is unlocked. Open Vault → Unlock first.", h.win)
		return
	}
	// refreshVault rebuilds the credential list, the lookup and the
	// toolbar together, so a credential added here is offered by the next
	// dialog that opens without anything else being told.
	ui.ShowVaultManager(h.win, h.vault, h.refreshVault)
}

// runInstallGUI opens the graphical installer (solo / Microsoft 365 / Google).
func runInstallGUI(setupPreset, ver string) {
	a := app.NewWithID("com.pathfinder.installer")
	ui.LoadUserThemes()
	base, _ := ui.LoadSettings(ui.SettingsPath())
	ui.SetSettings(base)
	ui.ApplyAppTheme(a, base.AppVariant())
	if icon := ui.AppIcon(); icon != nil {
		a.SetIcon(icon)
	}
	w := a.NewWindow("Install PathfinderSSH")
	w.Resize(fyne.NewSize(640, 520))
	w.CenterOnScreen()

	home := ui.GetAppHome()
	auth := mspauth.NewAuthenticator(home)
	ui.ShowInstallWizard(w, ui.InstallWizardOptions{
		Version:     ver,
		PresetSetup: setupPreset,
		Home:        home,
		Enroll:      auth.EnrollAndVerify,
	})
	w.ShowAndRun()
}

// maybeSelfInstall copies pathfinder into LocalAppData and relaunches from
// there. Any Windows launch outside the install dir installs itself — no
// START.bat or PowerShell launcher is required.
//
// Never relaunches when this process is already the AppData binary (path
// compare is prefix-safe). A false "not installed" check used to Start() a
// second copy of the same exe → two windows and racing settings.json writes.
func maybeSelfInstall(force, skip bool) error {
	if runtime.GOOS != "windows" || skip {
		return nil
	}
	exe, _ := os.Executable()
	destWant := appinstall.ExePath()
	if appinstall.SameFile(exe, destWant) && !force {
		return nil
	}
	if appinstall.RunningInstalled() && !force {
		return nil
	}
	dest, _, err := appinstall.Ensure()
	if err != nil {
		return err
	}
	if err := appinstall.CreateShortcuts(dest); err != nil {
		log.Printf("shortcuts: %v", err)
	}
	if force {
		return nil
	}
	if appinstall.SameFile(exe, dest) || appinstall.RunningInstalled() {
		return nil
	}
	// Relaunch the installed binary so AppData is the running copy.
	args := make([]string, 0, len(os.Args))
	for _, a := range os.Args[1:] {
		switch a {
		case "-install", "-no-install", "-uninstall":
			continue
		}
		args = append(args, a)
	}
	cmd := exec.Command(dest, args...)
	cmd.Dir = appinstall.Root()
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("relaunch installed binary: %w", err)
	}
	os.Exit(0)
	return nil
}

// installDesktopShortcuts copies this binary into LocalAppData and refreshes
// Desktop / Start Menu shortcuts.
func (h *host) installDesktopShortcuts() {
	dest, _, err := appinstall.Ensure()
	if err != nil {
		dialog.ShowError(err, h.win)
		return
	}
	if err := appinstall.CreateShortcuts(dest); err != nil {
		dialog.ShowError(err, h.win)
		return
	}
	dialog.ShowInformation("Installed",
		fmt.Sprintf("Pathfinder is at:\n%s\n\nDesktop and Start Menu shortcuts point at this executable.", dest),
		h.win)
}

func (h *host) uninstallDesktop() {
	dialog.ShowConfirm("Uninstall",
		"Remove the LocalAppData PathfinderSSH MSP copy and Desktop/Start Menu shortcuts?\n\nYour sessions.yaml and vault under ~/.pathfinderssh are kept.",
		func(ok bool) {
			if !ok {
				return
			}
			if err := appinstall.Uninstall(); err != nil {
				dialog.ShowError(err, h.win)
				return
			}
			dialog.ShowInformation("Uninstalled", "AppData install and shortcuts removed. You can close this window.", h.win)
		}, h.win)
}

// runMSPAccessSetup gates startup on MSP enrollment and engineer sign-in.
func (h *host) runMSPAccessSetup(onDone func()) {
	if onDone == nil {
		onDone = func() {}
	}
	auth := mspauth.NewAuthenticator(ui.GetAppHome())
	h.mspAuth = auth

	finish := func(enroll mspauth.Enrollment, sess mspauth.UserSession) {
		h.mspEnrollment = enroll
		h.mspSession = sess
		who := strings.TrimSpace(sess.Email)
		if who == "" {
			who = sess.Name
		}
		if who != "" {
			log.Printf("[mspauth] session: %s (%s)", who, enroll.Provider)
		}
		onDone()
	}

	if h.mspEnrollOnStart {
		h.showMSPEnrollWizard(auth, finish)
		return
	}

	enroll, found, err := mspauth.LoadEnrollment()
	if err != nil {
		dialog.ShowError(err, h.win)
		onDone()
		return
	}
	if !found {
		h.showMSPEnrollWizard(auth, finish)
		return
	}
	h.mspEnrollment = enroll
	sess, ok, err := auth.CurrentSession(enroll)
	if err != nil {
		dialog.ShowError(err, h.win)
		onDone()
		return
	}
	if mspauth.LoginRequired(enroll, sess, ok) {
		ui.ShowMSPSetupDialog(h.win, ui.MSPSetupOptions{
			Mode:        ui.MSPSetupLogin,
			Enrollment:  enroll,
			Authenticate: auth.SignIn,
			OnComplete:  finish,
		})
		return
	}
	finish(enroll, sess)
}

func (h *host) showMSPEnrollWizard(auth *mspauth.Authenticator, finish func(mspauth.Enrollment, mspauth.UserSession)) {
	ui.ShowMSPSetupDialog(h.win, ui.MSPSetupOptions{
		Mode:           ui.MSPSetupEnroll,
		PresetProvider: h.mspSetupPreset,
		Enroll:         auth.EnrollAndVerify,
		OnComplete:     finish,
	})
}

// offerFirstRunSetup prompts for SecureCRT import when the session tree is empty.
func (h *host) offerFirstRunSetup() {
	if h.tree == nil {
		return
	}
	if n := len(h.tree.Tree().Nodes()); n > 0 {
		return
	}
	msg := "No sessions yet.\n\nImport SecureCRT now? You will pick which folder is your customer list; nested folders are kept.\n\nYou can also use File → Import SecureCRT later."
	dialog.ShowConfirm("First-run setup", msg, func(ok bool) {
		if !ok {
			return
		}
		h.importSecureCRT()
	}, h.win)
}

// offerVaultCreate raises the first-run warning when this run found no vault.
//
// It fires only when nothing exists at the path this run resolved, so it asks
// once in the life of a machine rather than once per launch, and "Not now" is
// recorded so it does not come back. That restraint is deliberate: a vault is
// not REQUIRED to use the application -- a session carries its own credentials
// and a crawl accepts a static username and password from its launch form --
// so this is a warning about what degrades without one, and a warning that
// returns every launch is a warning that gets clicked through.
//
// The way back after declining is the Vault button, which offers creation
// whenever the file is missing.
func (h *host) offerVaultCreate() {
	check := ui.VaultCheck{
		Path:     h.vaultPath,
		Unlocked: h.vault != nil,
		Declined: h.base.VaultPromptDeclined,
	}
	if _, err := os.Stat(check.Path); err == nil {
		check.Present = true
	}
	if !check.ShouldOffer() {
		return
	}

	body := widget.NewLabel(ui.VaultSetupWarning(check.Path))
	body.Wrapping = fyne.TextWrapWord

	d := dialog.NewCustomConfirm(ui.VaultSetupTitle, "Create vault…", "Not now",
		body, func(create bool) {
			if !create {
				h.declineVaultPrompt()
				return
			}
			h.showCreateVaultDialog(check.Path)
		}, h.win)
	d.Resize(fyne.NewSize(620, 420))
	d.Show()
}

// declineVaultPrompt records that the first-run warning was answered.
//
// A write failure is logged rather than raised: the answer has been honoured
// for this run, and the whole cost of the file not being written is being asked
// once more. An error dialog about a suppression flag, on top of the dialog it
// suppresses, is worse than the thing it reports.
func (h *host) declineVaultPrompt() {
	h.base.VaultPromptDeclined = true
	ui.SetSettings(h.base)
	if err := ui.SaveSettings(h.settingsPath, h.base); err != nil {
		log.Printf("[settings] could not record the vault prompt answer: %v", err)
	}
}

// showCreateVaultDialog creates a new vault and adopts it.
//
// The CONFIRM field is not friction. A new master password has nothing to be
// checked against, so a typo is unrecoverable in the worst way available: the
// vault opens for nobody, and every credential added afterwards is encrypted to
// a password no one knows.
func (h *host) showCreateVaultDialog(start string) {
	path := widget.NewEntry()
	path.SetText(start)
	path.SetPlaceHolder(vaultcli.DefaultPath())

	state := widget.NewLabel("")
	state.Wrapping = fyne.TextWrapWord
	checkPath := func(p string) {
		p = ui.ExpandHome(p)
		if p == "" {
			p = vaultcli.DefaultPath()
		}
		if _, err := os.Stat(p); err == nil {
			state.SetText("a vault already exists there — cancel and unlock it instead")
			return
		}
		state.SetText("will be created: " + p)
	}
	path.OnChanged = checkPath
	checkPath(start)

	pass := widget.NewPasswordEntry()
	confirm := widget.NewPasswordEntry()

	// Off by default, for the same reason as the unlock dialog: filing a
	// master password in the OS keyring is a decision that outlives this
	// window.
	remember := widget.NewCheck("Remember in the OS keyring", nil)

	status := widget.NewLabel("")
	status.Wrapping = fyne.TextWrapWord

	items := []*widget.FormItem{
		widget.NewFormItem("Vault", path),
		widget.NewFormItem("", state),
		widget.NewFormItem("Master password", pass),
		widget.NewFormItem("Confirm", confirm),
		widget.NewFormItem("", remember),
		widget.NewFormItem("", status),
	}

	// Reopened rather than rebuilt on a refused form, the way the credential
	// editor does it: the widgets are the same objects, so what was typed
	// survives. Making somebody retype a master password twice because they
	// mistyped it once is how a dialog gets abandoned.
	var show func()
	show = func() {
		d := dialog.NewForm("Create vault", "Create", "Cancel", items, func(ok bool) {
			if !ok {
				return
			}
			p := ui.ExpandHome(path.Text)
			if p == "" {
				p = vaultcli.DefaultPath()
			}

			form := ui.VaultCreateForm{Path: p, Master: pass.Text, Confirm: confirm.Text}
			if errs := form.Validate(); len(errs) > 0 {
				status.SetText("⚠  " + ui.ProblemText(errs))
				show()
				return
			}

			// Making the directory is part of honouring "create a
			// vault here" -- the same reasoning as writeMap. 0o700
			// because of what is about to be in it; NTFS ignores
			// the mode, so on Windows this is the directory and
			// nothing more.
			if dir := filepath.Dir(p); dir != "" && dir != "." {
				if err := os.MkdirAll(dir, 0o700); err != nil {
					status.SetText("⚠  " + err.Error())
					show()
					return
				}
			}

			v := vault.New(p)
			if err := v.Create(pass.Text); err != nil {
				status.SetText("⚠  " + err.Error())
				show()
				return
			}
			status.SetText("")
			h.vaultPath = p
			h.adopt(v)

			// A vault now exists, so the first-run warning has been
			// answered by events. Clearing the flag keeps the
			// settings file honest rather than carrying a decline
			// that no longer describes anything.
			if h.base.VaultPromptDeclined {
				h.base.VaultPromptDeclined = false
				ui.SetSettings(h.base)
				if err := ui.SaveSettings(h.settingsPath, h.base); err != nil {
					log.Printf("[settings] %v", err)
				}
			}

			if remember.Checked {
				if err := vaultcli.KeyringSet(p, pass.Text); err != nil {
					dialog.ShowError(fmt.Errorf("created, but the keyring refused the password: %w", err), h.win)
				}
			}
			dialog.ShowInformation("Vault created", ui.VaultCreatedNote(p), h.win)
		}, h.win)
		d.Resize(fyne.NewSize(600, 380))
		d.Show()
		ui.EnterConfirmsForm(h.win, items, d.Submit)
	}
	show()
}

// showVaultDialog is the only place a master password is ever asked for.
//
// Three states, and which one it is decides the question. An unlocked vault is
// asked whether to lock; a vault file that is present is asked for its master
// password; a path with no file behind it is offered CREATION, because on a
// fresh machine the alternative is a closed loop -- the dialog cannot unlock a
// file that does not exist, and until this existed the only way to make one was
// pfvault init, which somebody who installed a GUI has no reason to have found.
func (h *host) showVaultDialog() {
	if h.vault != nil {
		dialog.ShowConfirm("Vault",
			fmt.Sprintf("%s is unlocked with %d credential(s).\n\nLock it?",
				h.vault.Path(), len(h.creds)),
			func(ok bool) {
				if !ok {
					return
				}
				h.vault.Lock()
				h.vault = nil
				h.refreshVault()
			}, h.win)
		return
	}

	if _, err := os.Stat(h.vaultPath); err != nil {
		h.showCreateVaultDialog(h.vaultPath)
		return
	}
	h.showUnlockVaultDialog(false)
}

// showUnlockVaultDialog asks for the master password of a vault that is there.
// When rememberDefault is true (startup), Remember in keyring is pre-checked.
func (h *host) showUnlockVaultDialog(rememberDefault bool) {
	// Prefill with a path that EXISTS. h.vaultPath comes from -vault, and a
	// flag naming a file that is not there is the most likely reason
	// someone is opening this dialog in the first place -- making them
	// type a password before being told the file is missing is a wasted
	// round trip. DefaultPath already prefers the current app directory
	// and falls back to the legacy one, so it finds a vault that moved.
	start := h.vaultPath
	if _, err := os.Stat(start); err != nil {
		if def := vaultcli.DefaultPath(); def != "" {
			if _, err := os.Stat(def); err == nil {
				start = def
			}
		}
	}

	path := widget.NewEntry()
	path.SetText(start)
	path.SetPlaceHolder(vaultcli.DefaultPath())

	found := widget.NewLabel("")
	found.Wrapping = fyne.TextWrapWord
	checkPath := func(p string) {
		p = ui.ExpandHome(p)
		if p == "" {
			p = vaultcli.DefaultPath()
		}
		if _, err := os.Stat(p); err != nil {
			found.SetText("no vault file at that path yet")
			return
		}
		found.SetText("vault file found")
	}
	path.OnChanged = checkPath
	checkPath(start)

	pass := widget.NewPasswordEntry()

	remember := widget.NewCheck("Remember in the OS keyring (unlock silently next time)", nil)
	if rememberDefault {
		remember.SetChecked(true)
	}

	items := []*widget.FormItem{
		widget.NewFormItem("Vault", path),
		widget.NewFormItem("", found),
		widget.NewFormItem("Master password", pass),
		widget.NewFormItem("", remember),
	}

	d := dialog.NewForm("Unlock vault", "Unlock", "Cancel", items, func(ok bool) {
		if !ok {
			return
		}
		p := ui.ExpandHome(path.Text)
		if p == "" {
			p = vaultcli.DefaultPath()
		}
		// A path typed into this field that has no file behind it is
		// not an error to report -- the person has already said which
		// file they mean, so the useful answer is to offer to make it.
		if _, statErr := os.Stat(p); statErr != nil {
			h.showCreateVaultDialog(p)
			return
		}
		v, err := vaultcli.OpenWith(p, pass.Text)
		if err != nil {
			// Distinguishable on purpose: a wrong password and a
			// missing file need different next actions, and the
			// error already says which it was.
			dialog.ShowError(err, h.win)
			return
		}
		h.vaultPath = p
		h.adopt(v)
		if remember.Checked {
			if err := vaultcli.KeyringSet(p, pass.Text); err != nil {
				dialog.ShowError(fmt.Errorf("unlocked, but the keyring refused it: %w", err), h.win)
			}
		}
	}, h.win)
	d.Resize(fyne.NewSize(560, 300))
	d.Show()
	ui.EnterConfirmsForm(h.win, items, d.Submit)
}

// adopt takes ownership of an unlocked vault and rebuilds everything derived
// from it.
func (h *host) adopt(v *vault.Vault) {
	if h.vault != nil && h.vault != v {
		h.vault.Lock()
	}
	h.vault = v
	h.vaultPath = v.Path()
	log.Printf("[vault] %s unlocked, %d credential(s)", v.Path(), len(v.Names()))
	h.refreshVault()
}

// refreshVault rebuilds the credential list, the dialer lookup and the toolbar
// button from whatever h.vault currently is.
//
// One function so the three can never disagree -- a picker offering names from
// a vault that has since been locked is worse than an empty picker, because
// the failure arrives at connect time instead of at click time.
func (h *host) refreshVault() {
	if h.vault == nil {
		h.creds = nil
		h.lookup = nil
		h.defaultCred = ""
		return
	}

	v := h.vault
	h.creds = v.Names()
	h.defaultCred = v.DefaultName()
	h.lookup = func(ref string) (sessiondial.Credential, error) {
		// An EMPTY ref asks what this store uses when a session names
		// nothing. It is not a lookup failure: "" names nothing, so it
		// cannot be missing, and a session that says nothing about auth
		// is the ordinary state of every node produced by a map import.
		if strings.TrimSpace(ref) == "" {
			c, ok := v.Default()
			if !ok {
				return sessiondial.Credential{}, nil
			}
			return dialCredential(c), nil
		}
		c, err := v.Get(ref)
		if err != nil {
			return sessiondial.Credential{}, err
		}
		return dialCredential(c), nil
	}
}

// dialCredential is the ONE vault-to-dial conversion.
//
// One function because there are two ways a credential reaches a connection --
// named by the session, or standing behind a blank field as the store default
// -- and two conversions would be two places for the auth type or a passphrase
// to be dropped from one path and not the other.
func dialCredential(c vault.Credential) sessiondial.Credential {
	return sessiondial.Credential{
		Username:      c.Username,
		AuthType:      authTypeName(c),
		Password:      c.Password,
		KeyPath:       c.KeyPath,
		KeyPassphrase: c.KeyPassphrase,
	}
}

// runVaultPath is the path a run should record, and it is empty unless the
// vault is actually open.
//
// That emptiness is load-bearing: Build treats an empty VaultPath as "use the
// static credentials from the form", which is the honest behaviour for a
// locked vault. A path with no open vault behind it would send Build looking
// for a master password it cannot ask for.
func (h *host) runVaultPath() string {
	if h.vault == nil {
		return ""
	}
	return h.vaultPath
}

// runVault is the vault a run should use, and it is nil when the run asked for
// manual credentials.
//
// Paired with runVaultPath: a manual run has to look to Build exactly like a
// run with no vault at all, or the static credentials it collected are never
// reached.
func (h *host) runVault(manual bool) *vault.Vault {
	if manual {
		return nil
	}
	return h.vault
}

func authTypeName(c vault.Credential) string {
	switch c.Method() {
	case vault.AuthPublicKey:
		return sessions.AuthPublicKey
	case vault.AuthPassword:
		return sessions.AuthPassword
	default:
		return ""
	}
}

// promptHostKey asks in the GUI rather than on stderr: by the time this fires
// there is a window on screen and nobody is watching the terminal the app was
// launched from. Called on the dial goroutine, so it hops to the UI goroutine
// to show the dialog and blocks on the answer.
func (h *host) promptHostKey(hostname string, remote net.Addr, key ssh.PublicKey) (bool, error) {
	answer := make(chan bool, 1)
	msg := fmt.Sprintf("%s (%s)\n\n%s\n%s\n\nAccept and remember this key?",
		hostname, remote, key.Type(), ssh.FingerprintSHA256(key))
	fyne.Do(func() {
		dialog.ShowConfirm("Unknown host key", msg, func(ok bool) { answer <- ok }, h.win)
	})
	select {
	case ok := <-answer:
		return ok, nil
	case <-time.After(60 * time.Second):
		// A prompt nobody answers must resolve to no. Timing out into
		// yes would make the policy meaningless in exactly the case
		// where it matters.
		return false, fmt.Errorf("host key prompt timed out")
	}
}

// promptSecret answers password and keyboard-interactive challenges the node
// did not supply material for. When the prompt looks like a password (not an
// OTP/passphrase), a "Save password" checkbox stores the secret in the vault
// and links this session to that credential — unlocking/creating the vault
// first when needed (the previous silent skip when locked is why saves never
// stuck).
func (h *host) promptSecret(folder, oldLabel string, n *sessions.Node, prompt string, echo bool) (string, error) {
	type answer struct {
		text string
		save bool
	}
	ch := make(chan answer, 1)
	label := strings.TrimSpace(prompt)
	if label == "" {
		label = "Password"
	}
	canSave := !echo && passwordPromptSavable(label)

	fyne.Do(func() {
		field := widget.NewPasswordEntry()
		if echo {
			field = widget.NewEntry()
		}
		field.SetPlaceHolder(label)

		items := []*widget.FormItem{widget.NewFormItem(label, field)}
		var saveBox *widget.Check
		if canSave {
			saveBox = widget.NewCheck("Save password for next time", nil)
			saveBox.SetChecked(true)
			items = append(items, widget.NewFormItem("", saveBox))
		}

		d := dialog.NewForm("Authentication", "Send", "Cancel", items,
			func(ok bool) {
				if !ok {
					ch <- answer{}
					return
				}
				save := saveBox != nil && saveBox.Checked
				ch <- answer{text: field.Text, save: save}
			}, h.win)
		d.Resize(fyne.NewSize(460, 240))
		d.Show()
		ui.EnterConfirmsForm(h.win, items, d.Submit)
	})

	select {
	case a := <-ch:
		if a.save && strings.TrimSpace(a.text) != "" {
			if h.ensureVaultUnlockedBlocking() {
				h.persistDialPassword(folder, oldLabel, n, a.text)
			} else {
				log.Printf("[vault] save skipped — vault was not unlocked")
			}
		}
		return a.text, nil
	case <-time.After(120 * time.Second):
		return "", fmt.Errorf("authentication prompt timed out")
	}
}

// ensureVaultUnlockedBlocking unlocks or creates the vault on the UI thread and
// blocks the caller (dial goroutine) until the operator answers. Used when a
// password save was requested.
func (h *host) ensureVaultUnlockedBlocking() bool {
	if h.vault != nil {
		return true
	}
	done := make(chan bool, 1)
	fyne.Do(func() {
		path := h.vaultPath
		if path == "" {
			path = vaultcli.DefaultPath()
		}
		pass := widget.NewPasswordEntry()
		pass.SetPlaceHolder("master password")
		remember := widget.NewCheck("Remember in the OS keyring", nil)
		remember.SetChecked(true)

		_, missing := os.Stat(path)
		title := "Unlock vault to save password"
		action := "Unlock"
		if missing != nil {
			title = "Create vault to save password"
			action = "Create"
		}
		items := []*widget.FormItem{
			widget.NewFormItem("Master password", pass),
			widget.NewFormItem("", remember),
		}
		d := dialog.NewForm(title, action, "Don't save", items, func(ok bool) {
			if !ok {
				done <- false
				return
			}
			master := pass.Text
			if len(master) < 8 {
				dialog.ShowError(fmt.Errorf("master password must be at least 8 characters"), h.win)
				done <- false
				return
			}
			if missing != nil {
				if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
					dialog.ShowError(err, h.win)
					done <- false
					return
				}
				v := vault.New(path)
				if err := v.Create(master); err != nil {
					dialog.ShowError(err, h.win)
					done <- false
					return
				}
				h.vaultPath = path
				h.adopt(v)
			} else {
				v, err := vaultcli.OpenWith(path, master)
				if err != nil {
					dialog.ShowError(err, h.win)
					done <- false
					return
				}
				h.vaultPath = path
				h.adopt(v)
			}
			if remember.Checked {
				if err := vaultcli.KeyringSet(path, master); err != nil {
					log.Printf("[vault] keyring: %v", err)
				}
			}
			done <- true
		}, h.win)
		d.Resize(fyne.NewSize(480, 260))
		d.Show()
		ui.EnterConfirmsForm(h.win, items, d.Submit)
	})
	select {
	case ok := <-done:
		return ok && h.vault != nil
	case <-time.After(180 * time.Second):
		return false
	}
}

// passwordPromptSavable reports whether a challenge is a password worth storing
// (not an OTP, token, or key passphrase).
func passwordPromptSavable(prompt string) bool {
	p := strings.ToLower(prompt)
	for _, skip := range []string{
		"passphrase", "otp", "totp", "token", "verification", "authenticator",
		"one-time", "one time", "mfa", "2fa", "pin",
	} {
		if strings.Contains(p, skip) {
			return false
		}
	}
	return true
}

// persistDialPassword writes the typed password into the vault and stamps
// Credential onto n so the post-connect inventory write links the session.
func (h *host) persistDialPassword(folder, oldLabel string, n *sessions.Node, password string) {
	if h.vault == nil || n == nil || strings.TrimSpace(password) == "" {
		return
	}

	credName := strings.TrimSpace(n.Credential)
	if credName == "" {
		credName = vaultCredNameFor(*n)
	}

	username := strings.TrimSpace(n.Username)
	if existing, err := h.vault.Get(credName); err == nil {
		existing.Password = password
		if username != "" {
			existing.Username = username
		}
		if strings.TrimSpace(existing.AuthType) == "" {
			existing.AuthType = "password"
		}
		if err := h.vault.Update(existing); err != nil {
			log.Printf("[vault] update credential %q: %v", credName, err)
			return
		}
		credName = existing.Name
	} else {
		c, err := h.vault.Add(vault.Credential{
			Name:     credName,
			Username: username,
			AuthType: "password",
			Password: password,
		})
		if err != nil {
			if errors.Is(err, vault.ErrDuplicateName) {
				credName = uniqueVaultCredName(h.vault, credName)
				c, err = h.vault.Add(vault.Credential{
					Name:     credName,
					Username: username,
					AuthType: "password",
					Password: password,
				})
			}
			if err != nil {
				log.Printf("[vault] add credential %q: %v", credName, err)
				return
			}
		}
		credName = c.Name
	}

	n.Credential = credName
	n.Password = "" // dial uses vault via Credential; do not keep inline after save
	n.AuthType = sessions.AuthPassword
	log.Printf("[vault] stored credential %q for %s (folder=%q)", credName, n.Label(), folder)

	// Refresh lookup so this same dial (and the next double-click) resolve it.
	fyne.Do(func() { h.refreshVault() })

	// Write credential: into sessions.yaml immediately when we know the folder,
	// so a failed dial after auth still leaves auto-login wired.
	if folder == "" || h.tree == nil {
		return
	}
	done := make(chan struct{})
	fyne.Do(func() {
		defer close(done)
		label := oldLabel
		if label == "" {
			label = n.Normalize().Label()
		}
		tr := h.tree.Tree()
		f, err := tr.FolderAt(folder)
		if err != nil {
			log.Printf("[vault] link session: %v", err)
			return
		}
		j := f.SessionIndex(label)
		if j < 0 {
			j = f.SessionIndex(n.Normalize().Label())
		}
		if j < 0 {
			log.Printf("[vault] link session: no %q in %q", label, folder)
			return
		}
		upd := f.Sessions[j]
		upd.Credential = credName
		upd.Password = ""
		upd.AuthType = sessions.AuthPassword
		if username != "" {
			upd.Username = username
		}
		replLabel := f.Sessions[j].Label()
		if err := tr.Replace(folder, replLabel, upd); err != nil {
			log.Printf("[vault] link session replace: %v", err)
			return
		}
		h.tree.SetTree(tr)
		if err := sessions.SaveFile(h.sessionsPath, tr); err != nil {
			log.Printf("[vault] link session save: %v", err)
			return
		}
		log.Printf("[vault] linked session %q → credential %q", replLabel, credName)
	})
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		log.Printf("[vault] link session timed out waiting for UI thread")
	}
}

// vaultCredNameFor picks a stable vault entry name for a dialled session.
func vaultCredNameFor(n sessions.Node) string {
	n = n.Normalize()
	if label := strings.TrimSpace(n.Label()); label != "" {
		if u := strings.TrimSpace(n.Username); u != "" {
			return u + "@" + label
		}
		return label
	}
	if u := strings.TrimSpace(n.Username); u != "" && strings.TrimSpace(n.Host) != "" {
		return u + "@" + n.Host
	}
	if host := strings.TrimSpace(n.Host); host != "" {
		return host
	}
	return "saved-password"
}

func uniqueVaultCredName(v *vault.Vault, base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "saved-password"
	}
	if _, err := v.Get(base); err != nil {
		return base
	}
	for i := 2; i < 1000; i++ {
		cand := fmt.Sprintf("%s-%d", base, i)
		if _, err := v.Get(cand); err != nil {
			return cand
		}
	}
	return fmt.Sprintf("%s-%d", base, time.Now().Unix())
}

// listPorts feeds the serial dropdown. On macOS prefer a /dev/cu.* entry over
// the matching /dev/tty.*: the tty device blocks on open until carrier is
// asserted, which a console cable with no modem control never does, so the app
// appears to hang.
func listPorts() []string {
	if ports, err := serialx.ListDetailed(); err == nil {
		out := make([]string, 0, len(ports))
		for _, p := range ports {
			out = append(out, p.Name)
		}
		return out
	}
	names, err := serialx.List()
	if err != nil {
		log.Printf("[serial] listing ports: %v", err)
		return nil
	}
	return names
}
