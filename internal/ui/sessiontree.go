// internal/ui/sessiontree.go
//
// The session tree: the saved inventory, docked down the left of the shell.
//
// A search entry on top, a folder tree under it, and a row of actions at the
// bottom. It renders a sessions.Tree and produces two kinds of event — "open
// this one" and "the inventory changed" — and it does neither of those things
// itself. It cannot connect, it cannot read or write a file, and it does not
// know it is in a split pane rather than a dialog or a tab: Content() returns a
// CanvasObject and the host decides where that goes. That is what keeps the
// choice between docked, modal and applet a one-line decision in the host.
//
// # Why the filter forces folders open
//
// A fyne.Tree remembers which branches the person opened. That is right when
// nothing is filtered and wrong the instant something is: a search that returns
// three devices inside shut folders looks like a search that found nothing. So
// filtering opens every folder it leaves, and clearing the filter hands control
// back rather than collapsing what the person had open.
//
// # Editing goes through the host
//
// New and Edit call back with the node and a function to apply the result,
// because the session form needs a vault credential list that this widget has
// no business holding. Folder work and delete are local, since they need
// nothing but the tree itself.
//
// # Opening a session is a double click
//
// Every session tree anybody has used opens on a double click, and a button
// that did the same job was one more thing on screen explaining a gesture the
// person already had. Single click still only selects, which is what makes the
// gesture safe on a list of production routers: nothing dials until the second
// click.
//
// Fyne has no double-click hook on widget.Tree, so the row content is a widget
// of its own that answers the mouse directly (see sessionRow). Two consequences
// worth knowing before touching it:
//
//   - the row now owns selection, because intercepting the click means the
//     toolkit's own tree node never sees it
//   - clicking a row no longer moves keyboard focus to the tree, since that
//     was also something the tree node did on tap. In this application that is
//     an improvement: picking a session out of the inventory while a terminal
//     is live used to take the keyboard away from it
package ui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/scottpeterman/pathfinderssh/internal/product"
	"github.com/scottpeterman/pathfinderssh/internal/sessions"
)

// SessionTreeOptions are the callbacks the host installs.
type SessionTreeOptions struct {
	// Window is needed for the confirmations this widget raises itself.
	Window fyne.Window

	// OnActivate fires when a session row is double clicked. It is
	// deliberately not a connect: the map surface already established that
	// this opens a prefilled dialog and the person presses Connect, and a
	// tree where a click dials a production router would be worse, not
	// better.
	OnActivate func(folder string, n sessions.Node)

	// OnEdit and OnNew hand the session form to the host, which owns the
	// credential list. apply is called with the edited node, or not at all.
	OnEdit func(folder string, n sessions.Node, apply func(sessions.Node))
	OnNew  func(folder string, apply func(sessions.Node))

	// OnChanged fires after any structural edit. The host saves; this widget
	// never touches a file.
	OnChanged func(sessions.Tree)
}

// SessionTree is the docked inventory panel.
type SessionTree struct {
	opts SessionTreeOptions

	tree sessions.Tree
	view TreeView

	search  *widget.Entry
	tw      *widget.Tree
	status  *widget.Label
	content fyne.CanvasObject

	selected string

	// Drag-and-drop: source uid while dragging, hover target for drop.
	draggingUID   string
	dropTargetUID string

	// filterDebounce coalesces keystrokes so each character does not rebuild
	// and fully Refresh a nested SecureCRT-sized tree.
	filterMu       sync.Mutex
	filterDebounce *time.Timer

	// sessionTotal is cached on each rebuild; avoids flattening Nodes() on
	// every status update.
	sessionTotal int
}

// NewSessionTree builds the panel. Call it after app.New(): it constructs
// widgets, and a widget built before the app exists nil-derefs inside
// CreateRenderer with a panic that names a layout function.
func NewSessionTree(o SessionTreeOptions) *SessionTree {
	t := &SessionTree{opts: o}
	t.view = BuildTreeView(t.tree, "")
	t.build()
	return t
}

// Content is the object to put in the split. It does not know it is in one.
func (t *SessionTree) Content() fyne.CanvasObject { return t.content }

// Tree is the current inventory.
func (t *SessionTree) Tree() sessions.Tree { return t.tree }

// SetTree replaces the inventory and redraws. It does NOT fire OnChanged —
// this is the host telling the widget what is true, not the widget telling the
// host something changed.
func (t *SessionTree) SetTree(tr sessions.Tree) {
	t.tree = tr
	t.refresh()
}

// Selected is the highlighted row, if it is a session.
//
// NOT WIRED. It was the Connect button's reader until opening became a double
// click, which carries its own uid. It survives a cull because it is the only
// way a host can ask this panel what is picked — which is what any keyboard
// shortcut or context menu will need first.
func (t *SessionTree) Selected() (folder string, n sessions.Node, ok bool) {
	row, ok := t.view.Rows[t.selected]
	if !ok || row.IsFolder {
		return "", sessions.Node{}, false
	}
	return row.Folder, row.Node, true
}

// SelectedFolder is the folder of the highlighted row, whether a folder or a
// session is selected. It is what "add here" means.
func (t *SessionTree) SelectedFolder() string {
	if row, ok := t.view.Rows[t.selected]; ok {
		return row.Folder
	}
	if len(t.tree.Folders) > 0 {
		return t.tree.Folders[0].Name
	}
	return ""
}

// FocusFilter puts the keyboard in the session tree filter box (Ctrl+L).
func (t *SessionTree) FocusFilter() {
	if t == nil || t.search == nil || t.opts.Window == nil {
		return
	}
	t.opts.Window.Canvas().Focus(t.search)
}

func (t *SessionTree) build() {
	t.search = widget.NewEntry()
	t.search.SetPlaceHolder("Filter by folder/session name (Ctrl+L)")
	t.search.OnChanged = func(string) { t.scheduleFilterRefresh() }

	t.status = widget.NewLabel("")
	t.status.Wrapping = fyne.TextWrapWord

	t.tw = widget.NewTree(
		func(uid widget.TreeNodeID) []widget.TreeNodeID { return t.view.Children[uid] },
		// Delegated, because the root has to answer true or fyne.Tree never
		// descends and nothing renders at all. See TreeView.IsBranch.
		func(uid widget.TreeNodeID) bool { return t.view.IsBranch(uid) },
		func(branch bool) fyne.CanvasObject {
			// One template per kind, reused for every row.
			return newSessionRow(t)
		},
		func(uid widget.TreeNodeID, branch bool, obj fyne.CanvasObject) {
			row, ok := obj.(*sessionRow)
			if !ok {
				return
			}
			row.set(uid, t.view.Rows[uid])
		},
	)

	// Selecting a row selects it and nothing else. Firing the connect dialog
	// on selection would mean a row could not be picked in order to edit or
	// delete it without a dialog appearing first — and every stray click on a
	// list of production routers would raise one. Opening is the second click,
	// handled by the row itself.
	t.tw.OnSelected = func(uid widget.TreeNodeID) { t.selected = uid }
	t.tw.OnUnselected = func(uid widget.TreeNodeID) {
		// Guarded: moving between rows unselects the old one, and depending on
		// the order those two callbacks run in, an unguarded clear would wipe
		// the selection that had just been made.
		if uid == t.selected {
			t.selected = ""
		}
	}

	// Connect, new session/folder/customer, edit, delete.
	// Drag a row onto a folder to move it (sessions or folders).
	actions := container.NewGridWithColumns(6,
		TipIconButton("Connect selected session", theme.LoginIcon(), t.connectSelected),
		TipIconButton("New session", theme.ContentAddIcon(), t.newSession),
		TipIconButton("New folder", theme.FolderNewIcon(), t.newFolder),
		TipIconButton("New customer", theme.AccountIcon(), t.newCustomer),
		TipIconButton("Edit selected", theme.DocumentCreateIcon(), t.editSelected),
		TipIconButton("Delete selected", theme.DeleteIcon(), t.deleteSelected),
	)

	bottom := container.NewVBox(actions, t.status)
	t.content = container.NewBorder(t.search, bottom, nil, nil, t.tw)
	t.refresh()
}

// RefreshView redraws the inventory from the current tree and filter.
// Call after theme/icon settings change so branch expand glyphs update.
func (t *SessionTree) RefreshView() {
	if t == nil {
		return
	}
	t.refresh()
}

// scheduleFilterRefresh rebuilds the view after typing pauses. Fyne's Tree
// Refresh walks every open node; doing that per keystroke freezes large trees.
func (t *SessionTree) scheduleFilterRefresh() {
	t.filterMu.Lock()
	defer t.filterMu.Unlock()
	if t.filterDebounce != nil {
		t.filterDebounce.Stop()
	}
	t.filterDebounce = time.AfterFunc(120*time.Millisecond, func() {
		fyne.Do(func() { t.refresh() })
	})
}

// refresh rebuilds the view from the tree and the current filter.
func (t *SessionTree) refresh() {
	filter := ""
	if t.search != nil {
		filter = t.search.Text
	}
	t.view = BuildTreeView(t.tree, filter)
	t.sessionTotal = t.view.TotalSessions

	if t.tw != nil {
		if strings.TrimSpace(filter) != "" {
			// OpenAllBranches marks every visible branch open and Refresh()s
			// once. Looping OpenBranch was N full tree refreshes (pathological
			// on nested SecureCRT imports).
			t.tw.OpenAllBranches()
		} else {
			t.tw.Refresh()
		}
		// A selection that the filter removed must go, or Selected() answers
		// with a row that is not on screen and Edit acts on the invisible one.
		if _, ok := t.view.Rows[t.selected]; !ok && t.selected != "" {
			t.tw.UnselectAll()
			t.selected = ""
		}
	}
	t.setStatus(filter)
}

func (t *SessionTree) setStatus(filter string) {
	if t.status == nil {
		return
	}
	total := t.sessionTotal
	switch {
	case total == 0 && filter == "":
		t.status.SetText("Empty — File → Import SecureCRT (pick customer list folder)")
	case filter == "":
		t.status.SetText(fmt.Sprintf("%d sessions", total))
	case t.view.Matched == 0:
		t.status.SetText(fmt.Sprintf("No match in %d sessions", total))
	default:
		t.status.SetText(fmt.Sprintf("%d of %d", t.view.Matched, total))
	}
}

// changed redraws and tells the host to save.
func (t *SessionTree) changed() {
	t.refresh()
	if t.opts.OnChanged != nil {
		t.opts.OnChanged(t.tree)
	}
}

// ── actions ──────────────────────────────────────────────────────────

// rowTapped is the single click. It selects, and that is all it does.
//
// The toolkit's own tree node normally does this; the row widget intercepts the
// click in order to see the second one, so the selection has to be made here
// instead. Tree.Select is a no-op when the row is already selected — critical,
// because Select otherwise full-Refresh()es the tree on every click.
func (t *SessionTree) rowTapped(uid widget.TreeNodeID) {
	if uid == "" || uid == t.selected {
		return
	}
	if t.tw != nil {
		t.tw.Select(uid)
	} else {
		t.selected = uid
	}
}

// rowActivated is the second click. On a session it hands the node to the
// host, which opens a prefilled dialog — it still does not dial, which is the
// contract the map surface settled on. On a folder it opens or shuts it, which
// is what a double click on a folder means everywhere else.
func (t *SessionTree) rowActivated(uid widget.TreeNodeID) {
	if t.tw != nil {
		t.tw.Select(uid)
	}
	row, ok := t.view.Rows[uid]
	if !ok {
		return
	}
	if row.IsFolder {
		if t.tw != nil {
			t.tw.ToggleBranch(uid)
		}
		return
	}
	if t.opts.OnActivate != nil {
		t.opts.OnActivate(row.Folder, row.Node)
	}
}

func (t *SessionTree) connectSelected() {
	folder, n, ok := t.Selected()
	if !ok {
		t.error(fmt.Errorf("select a session to connect"))
		return
	}
	if t.opts.OnActivate != nil {
		t.opts.OnActivate(folder, n)
	}
}

func (t *SessionTree) newSession() {
	if t.opts.OnNew == nil {
		return
	}
	folder := t.SelectedFolder()
	if folder == "" {
		t.error(fmt.Errorf("make a folder first"))
		return
	}
	t.opts.OnNew(folder, func(n sessions.Node) {
		if err := t.tree.Add(folder, n); err != nil {
			t.error(err)
			return
		}
		t.changed()
	})
}

func (t *SessionTree) newFolder() {
	entry := widget.NewEntry()
	entry.SetPlaceHolder("Site, role, vendor…")
	parent := t.SelectedFolder()
	hint := "Name"
	if parent != "" {
		hint = "Name (under " + sessions.PathLeaf(parent) + ")"
	}
	items := []*widget.FormItem{widget.NewFormItem(hint, entry)}
	d := dialog.NewForm("New folder", "Create", "Cancel", items,
		func(ok bool) {
			if !ok {
				return
			}
			name := strings.TrimSpace(entry.Text)
			path := name
			if parent != "" && name != "" {
				path = sessions.JoinPath(parent, name)
			}
			if err := t.tree.AddFolder(path); err != nil {
				t.error(err)
				return
			}
			t.changed()
		}, t.opts.Window)
	d.Show()
	EnterConfirmsForm(t.opts.Window, items, d.Submit)
}

// newCustomer creates a customer folder under the built-in Customers root.
func (t *SessionTree) newCustomer() {
	entry := widget.NewEntry()
	entry.SetPlaceHolder("Customer name")
	items := []*widget.FormItem{widget.NewFormItem("Customer", entry)}
	d := dialog.NewForm("New customer", "Create", "Cancel", items,
		func(ok bool) {
			if !ok {
				return
			}
			if _, err := t.tree.CreateCustomer(product.CustomersRoot, entry.Text); err != nil {
				t.error(err)
				return
			}
			if _, err := EnsureCustomerMapsDir(GetAppHome(), entry.Text); err != nil {
				// Session folder still exists; maps dir can be created on first crawl.
				t.status.SetText("Customer created; maps folder: " + err.Error())
			}
			t.changed()
			t.status.SetText("Customer created under " + product.CustomersRoot)
		}, t.opts.Window)
	d.Show()
	EnterConfirmsForm(t.opts.Window, items, d.Submit)
}

// applyDrop moves a dragged session or folder onto a destination folder.
func (t *SessionTree) applyDrop(srcUID, dstUID string) {
	src, ok := t.view.Rows[srcUID]
	if !ok {
		return
	}
	dst, ok := t.view.Rows[dstUID]
	if !ok || !dst.IsFolder {
		t.error(fmt.Errorf("drop onto a folder to move"))
		return
	}
	destPath := dst.Folder
	if src.IsFolder {
		if src.Folder == destPath {
			return
		}
		if err := t.tree.MoveFolder(src.Folder, destPath); err != nil {
			t.error(err)
			return
		}
		t.changed()
		t.status.SetText("Moved folder into " + sessions.PathLeaf(destPath))
		return
	}
	if src.Folder == destPath {
		return
	}
	if err := t.tree.Move(src.Folder, src.Label, destPath); err != nil {
		t.error(err)
		return
	}
	t.changed()
	t.status.SetText("Moved " + src.Label + " into " + sessions.PathLeaf(destPath))
}

func (t *SessionTree) editSelected() {
	if _, ok := t.view.Rows[t.selected]; !ok {
		t.error(fmt.Errorf("select a session or a folder first"))
		return
	}
	t.editRow(t.selected)
}

// rowEdit is the row's own overflow action. It selects first, so the dialog
// that opens and the row that is highlighted behind it are the same one — and
// so that whatever the person does next with the action bar acts on the row
// they just used, not on whichever one was selected before.
func (t *SessionTree) rowEdit(uid widget.TreeNodeID) {
	t.rowTapped(uid)
	t.editRow(uid)
}

// editRow opens the editor for one row: the session form for a session, the
// rename dialog for a folder.
func (t *SessionTree) editRow(uid widget.TreeNodeID) {
	row, ok := t.view.Rows[uid]
	if !ok {
		return
	}
	if row.IsFolder {
		t.renameFolder(row.Folder)
		return
	}
	if t.opts.OnEdit == nil {
		return
	}
	folder, label := row.Folder, row.Label
	t.opts.OnEdit(folder, row.Node, func(n sessions.Node) {
		if err := t.tree.Replace(folder, label, n); err != nil {
			t.error(err)
			return
		}
		t.changed()
	})
}

func (t *SessionTree) renameFolder(path string) {
	entry := widget.NewEntry()
	entry.SetText(sessions.PathLeaf(path))
	items := []*widget.FormItem{widget.NewFormItem("Name", entry)}
	d := dialog.NewForm("Rename folder", "Rename", "Cancel", items,
		func(ok bool) {
			if !ok {
				return
			}
			if err := t.tree.RenameFolder(path, entry.Text); err != nil {
				t.error(err)
				return
			}
			t.changed()
		}, t.opts.Window)
	d.Show()
	EnterConfirmsForm(t.opts.Window, items, d.Submit)
}

func (t *SessionTree) deleteSelected() {
	row, ok := t.view.Rows[t.selected]
	if !ok {
		t.error(fmt.Errorf("select a session or a folder first"))
		return
	}

	if !row.IsFolder {
		dialog.ShowConfirm("Delete session",
			fmt.Sprintf("Remove %q from %q?", row.Label, row.Folder),
			func(yes bool) {
				if !yes {
					return
				}
				if err := t.tree.Remove(row.Folder, row.Label); err != nil {
					t.error(err)
					return
				}
				t.changed()
			}, t.opts.Window)
		return
	}

	// The model refuses a non-empty folder unless forced. Ask with the count
	// in the question, so "delete" and "delete 40 devices" are not one click
	// apart with the same wording.
	count := len(t.view.Children[row.UID])
	msg := fmt.Sprintf("Remove the folder %q?", row.Folder)
	if count > 0 {
		msg = fmt.Sprintf("Remove %q and the %d session(s) in it?", row.Folder, count)
	}
	dialog.ShowConfirm("Delete folder", msg, func(yes bool) {
		if !yes {
			return
		}
		if err := t.tree.RemoveFolder(row.Folder, true); err != nil {
			t.error(err)
			return
		}
		t.changed()
	}, t.opts.Window)
}

func (t *SessionTree) error(err error) {
	if err == nil {
		return
	}
	if t.opts.Window != nil {
		dialog.ShowError(err, t.opts.Window)
		return
	}
	if t.status != nil {
		t.status.SetText(err.Error())
	}
}

// ── the row ──────────────────────────────────────────────────────────

// sessionRow is one line in the tree: an icon, a label, and a right-aligned
// detail, in a widget that answers the mouse.
//
// It exists because widget.Tree has no double-click hook. Fyne dispatches a
// click to the DEEPEST object under the cursor that implements any of the
// pointer interfaces, so a row that implements them wins over the toolkit's own
// tree node — which means this type has to do the selecting as well, since the
// tree node no longer gets the chance. That is the whole trade: one small widget
// in exchange for the gesture everybody already knows.
//
// Selection happens on MouseDown rather than on Tapped. A widget that can be
// double clicked has its single Tapped held back until the double-click window
// expires, so selecting there would leave a quarter of a second between clicking
// a row and seeing it highlight — small, and enough to make the panel feel
// broken.
type sessionRow struct {
	widget.BaseWidget

	tree *SessionTree
	uid  widget.TreeNodeID

	icon   *widget.Icon
	label  *widget.Label
	detail *widget.Label
	more   *TipButton
	right  *fyne.Container
	box    *fyne.Container
}

// The interfaces this row depends on, asserted rather than assumed: none of
// them are called by name anywhere in this package, so a signature that drifted
// would otherwise show up as a row that quietly stopped responding.
var (
	_ fyne.Tappable       = (*sessionRow)(nil)
	_ fyne.DoubleTappable = (*sessionRow)(nil)
	_ fyne.Draggable      = (*sessionRow)(nil)
	_ desktop.Mouseable   = (*sessionRow)(nil)
	_ desktop.Hoverable   = (*sessionRow)(nil)
)

func newSessionRow(t *SessionTree) *sessionRow {
	r := &sessionRow{tree: t}

	// Truncate rather than wrap: a wrapped hostname makes rows different
	// heights and the tree's own row measurement stops matching what is drawn.
	r.label = widget.NewLabel("")
	r.label.Truncation = fyne.TextTruncateEllipsis

	// Folder rows only, and deliberately NOT truncated. A truncated label's
	// MinSize collapses to the width of the ellipsis, which is why every row
	// in this tree used to end in a "..." that was the session count rather
	// than a control. The folder count is short and bounded, so it fits.
	r.detail = widget.NewLabel("")

	// Session rows only. It reads r.uid when it is pressed rather than
	// capturing it: rows are reused as the tree scrolls, so a captured uid
	// would edit whichever session happened to be drawn in this row first.
	r.more = TipIconButtonLow("Edit session", theme.MoreHorizontalIcon(), func() {
		if r.tree != nil && r.uid != "" {
			r.tree.rowEdit(r.uid)
		}
	})

	r.icon = widget.NewIcon(theme.FolderIcon())

	// Both live in the same slot and one of them is hidden. Stack skips
	// hidden children when it measures, so a session row does not reserve
	// width for a folder count it will never show.
	r.right = container.NewStack(r.detail, r.more)
	r.box = container.NewBorder(nil, nil, r.icon, r.right, r.label)

	r.ExtendBaseWidget(r)
	return r
}

func (r *sessionRow) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(r.box)
}

// set fills the row in for whichever uid the tree is currently drawing. Rows
// are reused, so every field is written every time — a field left alone keeps
// the previous row's value.
func (r *sessionRow) set(uid widget.TreeNodeID, row TreeRow) {
	r.uid = uid
	r.label.SetText(row.Label)
	r.detail.SetText(row.Detail)
	if row.IsFolder {
		r.icon.SetResource(theme.FolderIcon())
		r.icon.Show()
		r.detail.Show()
		r.more.Hide()
		return
	}
	// No icon on a session row. Every session carried the same one, so it
	// separated nothing from nothing and spent a column of a quarter-width
	// pane doing it. The indent already says the row is inside a folder.
	r.icon.Hide()
	r.detail.Hide()
	r.more.Show()
}

// MouseDown selects immediately. See the type comment for why this is not in
// Tapped.
func (r *sessionRow) MouseDown(*desktop.MouseEvent) {
	if r.tree != nil && r.uid != "" {
		r.tree.rowTapped(r.uid)
	}
}

func (r *sessionRow) MouseUp(*desktop.MouseEvent) {}

// Tapped selects too. On this platform MouseDown has already done it and this
// is a no-op that arrives late; it is here so the row still selects under a
// driver that sends taps without mouse events.
func (r *sessionRow) Tapped(*fyne.PointEvent) {
	if r.tree != nil && r.uid != "" {
		r.tree.rowTapped(r.uid)
	}
}

func (r *sessionRow) DoubleTapped(*fyne.PointEvent) {
	if r.tree != nil && r.uid != "" {
		r.tree.rowActivated(r.uid)
	}
}

// Dragged starts a move; drop onto another folder row to relocate.
func (r *sessionRow) Dragged(*fyne.DragEvent) {
	if r.tree == nil || r.uid == "" {
		return
	}
	if r.tree.draggingUID == "" {
		r.tree.draggingUID = r.uid
		r.tree.status.SetText("Drop onto a folder to move…")
	}
}

func (r *sessionRow) DragEnd() {
	if r.tree == nil {
		return
	}
	src, dst := r.tree.draggingUID, r.tree.dropTargetUID
	r.tree.draggingUID = ""
	r.tree.dropTargetUID = ""
	if src != "" && dst != "" && src != dst {
		r.tree.applyDrop(src, dst)
		return
	}
	if src != "" {
		r.tree.status.SetText("")
	}
}

func (r *sessionRow) MouseIn(*desktop.MouseEvent) {
	if r.tree == nil || r.tree.draggingUID == "" || r.uid == "" {
		return
	}
	if row, ok := r.tree.view.Rows[r.uid]; ok && row.IsFolder {
		r.tree.dropTargetUID = r.uid
	}
}

func (r *sessionRow) MouseMoved(*desktop.MouseEvent) {}

func (r *sessionRow) MouseOut() {
	if r.tree != nil && r.tree.dropTargetUID == r.uid {
		r.tree.dropTargetUID = ""
	}
}
