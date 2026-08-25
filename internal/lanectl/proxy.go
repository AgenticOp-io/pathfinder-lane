package lanectl

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/scottpeterman/pathfinderssh/internal/auvik"
	"github.com/scottpeterman/pathfinderssh/internal/crtapp"
	"github.com/scottpeterman/pathfinderssh/internal/crtbridge"
	"github.com/scottpeterman/pathfinderssh/internal/vpnprov"
)

// RunProxy is OpenSSH ProxyCommand / PuTTY local proxy: ensure the mapped
// VPN (and Auvik tunnel when mapped), then splice stdin/stdout to the host.
// Logs go to stderr so they cannot corrupt the SSH stream.
func RunProxy(ctx context.Context, appHome, folder, host string, port int, in io.Reader, out io.Writer) error {
	if appHome == "" {
		appHome = crtapp.Home()
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return fmt.Errorf("proxy: host required")
	}
	if port <= 0 {
		port = 22
	}
	if in == nil {
		in = os.Stdin
	}
	if out == nil {
		out = os.Stdout
	}
	cfg, err := crtbridge.LoadSettings(appHome)
	if err != nil {
		cfg = crtbridge.Settings{Mode: crtbridge.AutoMixed}
	}
	rel := folder + "/" + host + ".ini"
	vpn := cfg.VPNTunnelForSession(rel, folder)
	if vpn != "" {
		vpnCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
		err := vpnprov.Ensure(vpnCtx, vpnprov.Bins{
			FortiBin:   cfg.VPNBin,
			FortiTools: cfg.VPNTools,
			WireGuard:  cfg.WGBin,
			Zscaler:    cfg.ZSABin,
		}, vpn)
		cancel()
		if err != nil {
			fmt.Fprintf(os.Stderr, "lane: vpn %s: %v — connecting to %s anyway\n", vpn, err, host)
		}
	}

	dialHost, dialPort := host, port
	if want := cfg.AuvikTenantForSession(rel, folder); want != "" && cfg.AuvikEnabled() {
		if local, ok := ensureAuvik(ctx, appHome, cfg, want, host, port); ok {
			dialHost, dialPort = "127.0.0.1", local
		}
	}

	d := &net.Dialer{}
	if dialHost != "127.0.0.1" && dialHost != "::1" && vpn != "" {
		d = vpnprov.Dialer(vpn)
	}
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(dialHost, strconv.Itoa(dialPort)))
	if err != nil {
		return fmt.Errorf("dial %s:%d: %w", dialHost, dialPort, err)
	}
	defer conn.Close()
	return spliceIO(in, out, conn)
}

func ensureAuvik(ctx context.Context, appHome string, cfg crtbridge.Settings, want, host string, port int) (int, bool) {
	tm, _ := auvik.LoadTenantMap(appHome)
	var tenants []auvik.Tenant
	if cfg.AuvikUser != "" && cfg.AuvikKey != "" {
		list, err := auvik.New(cfg.AuvikUser, cfg.AuvikKey, cfg.AuvikBase).ListTenants(ctx)
		if err == nil {
			tenants = list
		}
	}
	m, ok := crtbridge.ResolveAuvik(want, tenants, tm)
	if !ok || m.Domain == "" {
		fmt.Fprintf(os.Stderr, "lane: auvik %q not resolved — connecting to %s directly\n", want, host)
		return 0, false
	}
	tmgr := auvik.NewTunnelManager(cfg.TunnelBin)
	tmgr.SetAppHome(appHome)
	tmgr.SetCredentials(cfg.AuvikUser, cfg.AuvikKey, cfg.AuvikBase)
	local, err := tmgr.Ensure(ctx, m.Domain, host, port, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "lane: auvik tunnel %s: %v — connecting to %s directly\n", m.Domain, err, host)
		return 0, false
	}
	return local, true
}

func spliceIO(in io.Reader, out io.Writer, conn net.Conn) error {
	done := make(chan error, 2)
	go func() {
		_, err := io.Copy(conn, in)
		_ = conn.SetDeadline(time.Now())
		done <- err
	}()
	go func() {
		_, err := io.Copy(out, conn)
		done <- err
	}()
	err1 := <-done
	err2 := <-done
	if err1 != nil && err1 != io.EOF {
		return err1
	}
	if err2 != nil && err2 != io.EOF {
		return err2
	}
	return nil
}
