// Port-forward manager dialog for an open SSH session.
package ui

import (
	"fmt"
	"strings"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/crypto/ssh"

	"github.com/scottpeterman/pathfinderssh/internal/portfwd"
)

// ForwardHub tracks live forwards so session close can tear them down.
type ForwardHub struct {
	mu   sync.Mutex
	byID map[string]*portfwd.Handle
	next int
}

// NewForwardHub returns an empty hub.
func NewForwardHub() *ForwardHub {
	return &ForwardHub{byID: map[string]*portfwd.Handle{}}
}

// Add registers a handle and returns its id.
func (h *ForwardHub) Add(handle *portfwd.Handle) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.next++
	id := fmt.Sprintf("%d", h.next)
	h.byID[id] = handle
	return id
}

// Stop closes one forward.
func (h *ForwardHub) Stop(id string) {
	h.mu.Lock()
	handle := h.byID[id]
	delete(h.byID, id)
	h.mu.Unlock()
	_ = handle.Close()
}

// StopAll closes every forward.
func (h *ForwardHub) StopAll() {
	h.mu.Lock()
	all := make([]*portfwd.Handle, 0, len(h.byID))
	for id, handle := range h.byID {
		all = append(all, handle)
		delete(h.byID, id)
	}
	h.mu.Unlock()
	for _, handle := range all {
		_ = handle.Close()
	}
}

// List returns id + summary lines for the UI.
func (h *ForwardHub) List() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]string, 0, len(h.byID))
	for id, handle := range h.byID {
		if handle == nil {
			continue
		}
		s := handle.Spec
		line := fmt.Sprintf("%s  %s  listen %s", id, s.Kind, s.ListenAddr)
		if s.TargetAddr != "" {
			line += " → " + s.TargetAddr
		}
		out = append(out, line)
	}
	return out
}

// ShowPortForwardDialog manages local/remote/dynamic forwards on sshClient.
func ShowPortForwardDialog(w fyne.Window, title string, sshClient *ssh.Client, hub *ForwardHub) {
	if w == nil || sshClient == nil {
		return
	}
	if hub == nil {
		hub = NewForwardHub()
	}
	if title == "" {
		title = "Port forwards"
	}

	kind := widget.NewSelect([]string{"Local", "Remote", "Dynamic (SOCKS5)"}, nil)
	kind.SetSelected("Local")
	listen := widget.NewEntry()
	listen.SetPlaceHolder("127.0.0.1:8080")
	listen.SetText("127.0.0.1:8080")
	target := widget.NewEntry()
	target.SetPlaceHolder("remote-host:80 (local/remote only)")
	target.SetText("127.0.0.1:80")

	profiles, _ := portfwd.LoadProfiles(GetAppHome())
	profileNames := []string{"(custom)"}
	profileByName := map[string]portfwd.Profile{}
	for _, p := range profiles {
		profileNames = append(profileNames, p.Name)
		profileByName[p.Name] = p
	}
	profileSel := widget.NewSelect(profileNames, nil)
	profileSel.SetSelected("(custom)")
	profileSel.OnChanged = func(name string) {
		p, ok := profileByName[name]
		if !ok {
			return
		}
		switch p.Kind {
		case "remote":
			kind.SetSelected("Remote")
		case "dynamic":
			kind.SetSelected("Dynamic (SOCKS5)")
		default:
			kind.SetSelected("Local")
		}
		listen.SetText(p.ListenAddr)
		target.SetText(p.TargetAddr)
	}

	status := widget.NewLabel("")
	list := widget.NewList(
		func() int { return len(hub.List()) },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			lines := hub.List()
			if i >= 0 && i < len(lines) {
				o.(*widget.Label).SetText(lines[i])
			}
		},
	)
	refresh := func() { list.Refresh(); status.SetText(fmt.Sprintf("%d active", len(hub.List()))) }

	start := widget.NewButtonWithIcon("Start", theme.MediaPlayIcon(), func() {
		spec := portfwd.Spec{ListenAddr: listen.Text, TargetAddr: target.Text}
		switch kind.Selected {
		case "Remote":
			spec.Kind = portfwd.Remote
		case "Dynamic (SOCKS5)":
			spec.Kind = portfwd.Dynamic
			spec.TargetAddr = ""
		default:
			spec.Kind = portfwd.Local
		}
		handle, err := portfwd.Start(sshClient, spec)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		hub.Add(handle)
		refresh()
	})
	saveProfile := widget.NewButton("Save as profile…", func() {
		nameEntry := widget.NewEntry()
		nameEntry.SetPlaceHolder("Profile name")
		dialog.ShowCustomConfirm("Save forward profile", "Save", "Cancel", nameEntry, func(ok bool) {
			if !ok || strings.TrimSpace(nameEntry.Text) == "" {
				return
			}
			kindStr := "local"
			switch kind.Selected {
			case "Remote":
				kindStr = "remote"
			case "Dynamic (SOCKS5)":
				kindStr = "dynamic"
			}
			p := portfwd.Profile{
				Name:       strings.TrimSpace(nameEntry.Text),
				Kind:       kindStr,
				ListenAddr: listen.Text,
				TargetAddr: target.Text,
			}
			all, _ := portfwd.LoadProfiles(GetAppHome())
			all = portfwd.UpsertProfile(all, p)
			if err := portfwd.SaveProfiles(GetAppHome(), all); err != nil {
				dialog.ShowError(err, w)
				return
			}
			status.SetText("Saved profile " + p.Name)
		}, w)
	})
	stopSel := -1
	list.OnSelected = func(i widget.ListItemID) { stopSel = int(i) }
	stopBtn := widget.NewButtonWithIcon("Stop selected", theme.MediaStopIcon(), func() {
		lines := hub.List()
		if stopSel < 0 || stopSel >= len(lines) {
			status.SetText("Select a forward to stop")
			return
		}
		// id is first token
		id := ""
		for _, c := range lines[stopSel] {
			if c == ' ' {
				break
			}
			id += string(c)
		}
		hub.Stop(id)
		stopSel = -1
		refresh()
	})

	form := widget.NewForm(
		widget.NewFormItem("Profile", profileSel),
		widget.NewFormItem("Type", kind),
		widget.NewFormItem("Listen", listen),
		widget.NewFormItem("Target", target),
	)
	body := container.NewBorder(
		container.NewVBox(form, container.NewHBox(start, saveProfile, stopBtn), status),
		nil, nil, nil, list,
	)
	refresh()
	d := dialog.NewCustom(title, "Close", body, w)
	d.Resize(fyne.NewSize(560, 420))
	d.Show()
}
