package crtbridge

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/scottpeterman/pathfinderssh/internal/auvik"
	"github.com/scottpeterman/pathfinderssh/internal/vpnprov"
)

// ensureVPN is the exclusive VPN bus; tests replace it.
var ensureVPN = func(ctx context.Context, cfg Settings, target string) error {
	return vpnprov.Ensure(ctx, vpnprov.Bins{
		FortiBin:   cfg.VPNBin,
		FortiTools: cfg.VPNTools,
		WireGuard:  cfg.WGBin,
		Zscaler:    cfg.ZSABin,
	}, target)
}

// Agent holds localhost listeners for proxied CRT sessions and splices
// them to AuvikTunnel and/or the original host after FortiClient is up.
type Agent struct {
	Opts    Options
	Tunnels *auvik.TunnelManager
	Log     *log.Logger
	cfg     Settings

	mu        sync.Mutex
	listeners map[int]net.Listener
}

// RunAgent syncs CRT sessions, serves localhost ports, and re-checks Auvik
// on a timer so new/removed customers pick up the right template automatically.
func RunAgent(ctx context.Context, opts Options) error {
	if opts.AppHome == "" {
		return fmt.Errorf("app home required")
	}
	_ = MigrateLegacyState(opts.AppHome)
	lg := log.Default()
	cfg := opts.settings()
	var tm *auvik.TunnelManager
	if cfg.AuvikEnabled() {
		tm = auvik.NewTunnelManager(cfg.TunnelBin)
		tm.SetAppHome(opts.AppHome)
		tm.SetCredentials(cfg.AuvikUser, cfg.AuvikKey, cfg.AuvikBase)
	}

	a := &Agent{
		Opts:      opts,
		Tunnels:   tm,
		Log:       lg,
		listeners: map[int]net.Listener{},
		cfg:       cfg,
	}
	_ = writeServePID(opts.AppHome)
	defer removeServePID(opts.AppHome)

	if _, err := Sync(ctx, opts); err != nil {
		lg.Printf("[crt-bridge] initial sync: %v", err)
	}
	if err := a.refreshListeners(); err != nil {
		lg.Printf("[crt-bridge] listeners: %v", err)
	}

	tick := time.NewTicker(5 * time.Minute)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			a.closeAll()
			return ctx.Err()
		case <-tick.C:
			if _, err := Sync(ctx, opts); err != nil {
				lg.Printf("[crt-bridge] sync: %v", err)
			}
			if err := a.refreshListeners(); err != nil {
				lg.Printf("[crt-bridge] listeners: %v", err)
			}
		}
	}
}

func (a *Agent) refreshListeners() error {
	st, err := LoadState(a.Opts.AppHome)
	if err != nil {
		return err
	}
	want := map[int]Session{}
	for _, s := range st.AllListeners() {
		if s.Proxied() && s.FrontPort > 0 {
			want[s.FrontPort] = s
		}
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	for port, ln := range a.listeners {
		if _, ok := want[port]; ok {
			continue
		}
		_ = ln.Close()
		delete(a.listeners, port)
	}
	for port, s := range want {
		if _, ok := a.listeners[port]; ok {
			continue
		}
		ln, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
		if err != nil {
			a.logf("listen %d: %v", port, err)
			continue
		}
		a.listeners[port] = ln
		go a.serve(ln, s)
	}
	return nil
}

func (a *Agent) serve(ln net.Listener, s Session) {
	for {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		go a.handle(c, s)
	}
}

func (a *Agent) handle(c net.Conn, s Session) {
	defer c.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	st, _ := LoadState(a.Opts.AppHome)
	live := s
	if found, ok := st.ListenerByFrontPort(s.FrontPort); ok {
		live = found
	}

	if live.VPNTunnel != "" {
		vpnCtx, vpnCancel := context.WithTimeout(context.Background(), 2*time.Minute)
		err := ensureVPN(vpnCtx, a.cfg, live.VPNTunnel)
		vpnCancel()
		if err != nil {
			a.logf("vpn %s (%s): %v - connecting to %s anyway", live.Customer, live.VPNTunnel, err, live.OriginalHost)
		}
	}

	var upstream net.Conn
	var err error
	if live.UseAuvik && live.Domain != "" && a.Tunnels != nil {
		local, terr := a.Tunnels.Ensure(ctx, live.Domain, live.DeviceIP, live.OriginalPort, 0)
		if terr != nil {
			a.logf("tunnel %s: %v — falling back to %s", live.Customer, terr, live.OriginalHost)
			upstream, err = net.DialTimeout("tcp", net.JoinHostPort(live.OriginalHost, strconv.Itoa(live.OriginalPort)), 20*time.Second)
		} else {
			upstream, err = net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(local)), 20*time.Second)
		}
	} else {
		upstream, err = net.DialTimeout("tcp", net.JoinHostPort(live.OriginalHost, strconv.Itoa(live.OriginalPort)), 20*time.Second)
	}
	if err != nil {
		a.logf("dial %s: %v", live.OriginalHost, err)
		return
	}
	defer upstream.Close()
	splice(c, upstream)
}

func splice(a, b net.Conn) {
	done := make(chan struct{}, 2)
	go func() { _, _ = io.Copy(a, b); done <- struct{}{} }()
	go func() { _, _ = io.Copy(b, a); done <- struct{}{} }()
	<-done
	_ = a.SetDeadline(time.Now())
	_ = b.SetDeadline(time.Now())
	<-done
}

func (a *Agent) closeAll() {
	a.mu.Lock()
	defer a.mu.Unlock()
	for p, ln := range a.listeners {
		_ = ln.Close()
		delete(a.listeners, p)
	}
}

func (a *Agent) logf(format string, args ...any) {
	if a.Log != nil {
		a.Log.Printf("[crt-bridge] "+format, args...)
	}
}
