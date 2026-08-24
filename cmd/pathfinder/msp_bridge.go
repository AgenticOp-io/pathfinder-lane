package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"

	"github.com/scottpeterman/pathfinderssh/internal/pfbridge"
	"github.com/scottpeterman/pathfinderssh/internal/ui"
)

// syncMSPBridge starts or stops the localhost Cursor IDE bridge from settings.
func (h *host) syncMSPBridge() {
	if h == nil {
		return
	}
	if h.mspBridge != nil {
		h.mspBridge.Stop()
		h.mspBridge = nil
	}
	if h.base.MSPBridgeDisabled {
		return
	}

	token := pfbridge.EnsureToken(h.base.MSPBridgeToken)
	if token != h.base.MSPBridgeToken {
		h.base.MSPBridgeToken = token
		ui.SetSettings(h.base)
		if h.settingsPath != "" {
			if err := ui.SaveSettings(h.settingsPath, h.base); err != nil {
				log.Printf("[msp-bridge] persist token: %v", err)
			}
		}
	}

	port := h.base.MSPBridgePort
	if port <= 0 {
		port = pfbridge.DefaultPort
	}
	srv := pfbridge.New(pfbridge.Config{
		Enabled:   true,
		Bind:      pfbridge.DefaultBind,
		Port:      port,
		Token:     token,
		AllowSend: h.base.MSPBridgeAllowSend,
		AppHome:   ui.GetAppHome(),
	}, h)
	if err := srv.Start(); err != nil {
		log.Printf("[msp-bridge] start failed: %v", err)
		return
	}
	h.mspBridge = srv
	log.Printf("[msp-bridge] listening on %s (allow_send=%v)", srv.Addr(), h.base.MSPBridgeAllowSend)
}

func (h *host) ListSessions() []pfbridge.Session {
	var out []pfbridge.Session
	fyne.DoAndWait(func() {
		out = h.listSessionsUI()
	})
	return out
}

func (h *host) ActiveSession() (pfbridge.Session, bool) {
	var sess pfbridge.Session
	var ok bool
	fyne.DoAndWait(func() {
		sess, ok = h.activeSessionUI()
	})
	return sess, ok
}

func (h *host) Scrollback(id string, maxChars int) (string, error) {
	var text string
	var err error
	fyne.DoAndWait(func() {
		text, err = h.scrollbackUI(id, maxChars)
	})
	return text, err
}

func (h *host) Send(id string, text string) error {
	var err error
	fyne.DoAndWait(func() {
		err = h.sendUI(id, text)
	})
	return err
}

func (h *host) listSessionsUI() []pfbridge.Session {
	if h.shell == nil {
		return nil
	}
	active := h.shell.ActiveTerminal()
	var out []pfbridge.Session
	for _, inst := range h.shell.Instances() {
		if inst == nil || inst.Applet() == nil {
			continue
		}
		ta, ok := inst.Applet().(*termApplet)
		if !ok || ta.sess == nil {
			continue
		}
		s := pfbridge.Session{
			ID:       strconv.Itoa(inst.ID()),
			Title:    inst.Title(),
			Customer: ta.customer,
			Folder:   ta.folder,
			Target:   ta.sess.TargetLabel(),
			Active:   active != nil && active.ID() == inst.ID(),
			Kind:     string(ui.KindTerminal),
		}
		out = append(out, s)
	}
	return out
}

func (h *host) activeSessionUI() (pfbridge.Session, bool) {
	if h.shell == nil {
		return pfbridge.Session{}, false
	}
	inst := h.shell.ActiveTerminal()
	if inst == nil {
		return pfbridge.Session{}, false
	}
	ta, ok := inst.Applet().(*termApplet)
	if !ok || ta.sess == nil {
		return pfbridge.Session{}, false
	}
	return pfbridge.Session{
		ID:       strconv.Itoa(inst.ID()),
		Title:    inst.Title(),
		Customer: ta.customer,
		Folder:   ta.folder,
		Target:   ta.sess.TargetLabel(),
		Active:   true,
		Kind:     string(ui.KindTerminal),
	}, true
}

func (h *host) scrollbackUI(id string, maxChars int) (string, error) {
	inst, err := h.terminalByID(id)
	if err != nil {
		return "", err
	}
	ta, ok := inst.Applet().(*termApplet)
	if !ok || ta.sess == nil {
		return "", fmt.Errorf("not a terminal session")
	}
	text := ta.sess.ScrollbackText()
	if maxChars > 0 && len(text) > maxChars {
		text = text[len(text)-maxChars:]
	}
	return text, nil
}

func (h *host) sendUI(id string, text string) error {
	if ok, why := h.allowSend(); !ok {
		return fmt.Errorf("%s", why)
	}
	text = ui.NormalizeTerminalSend(text)
	if strings.TrimSpace(text) == "" {
		return fmt.Errorf("empty text")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		if !h.shell.SendToActive(text) {
			return fmt.Errorf("no active terminal or send rejected")
		}
		h.shell.RefocusCurrentTerminal()
		return nil
	}
	inst, err := h.terminalByID(id)
	if err != nil {
		return err
	}
	ta, ok := inst.Applet().(*termApplet)
	if !ok || ta.sess == nil {
		return fmt.Errorf("not a terminal session")
	}
	if !ta.sess.SendUser([]byte(text)) {
		return fmt.Errorf("send rejected (disconnected or read-only)")
	}
	h.shell.Activate(inst)
	h.shell.RefocusCurrentTerminal()
	return nil
}

func (h *host) terminalByID(id string) (*ui.Instance, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		inst := h.shell.ActiveTerminal()
		if inst == nil {
			return nil, fmt.Errorf("no active terminal")
		}
		return inst, nil
	}
	want, err := strconv.Atoi(id)
	if err != nil || want <= 0 {
		return nil, fmt.Errorf("invalid session id %q", id)
	}
	for _, inst := range h.shell.Instances() {
		if inst != nil && inst.ID() == want {
			if _, ok := inst.Applet().(*termApplet); !ok {
				return nil, fmt.Errorf("session %d is not a terminal", want)
			}
			return inst, nil
		}
	}
	return nil, fmt.Errorf("session %d not found", want)
}
