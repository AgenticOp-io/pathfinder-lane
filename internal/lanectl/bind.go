package lanectl

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/scottpeterman/pathfinderssh/internal/crtapp"
	"github.com/scottpeterman/pathfinderssh/internal/crtbridge"
	"github.com/scottpeterman/pathfinderssh/internal/sessions"
	"github.com/scottpeterman/pathfinderssh/internal/vpnprov"
)

// PrepareConnect brings up the mapped customer VPN before Pathfinder (or
// any other client) dials. Auvik tunnels stay Pathfinder's own path.
// VPN failure is returned for logging; the caller should still connect.
func PrepareConnect(ctx context.Context, folder, name, host string) error {
	vpn, cfg := VPNTarget(folder, name, host)
	if vpn == "" {
		return nil
	}
	return vpnprov.Ensure(ctx, vpnprov.Bins{
		FortiBin:   cfg.VPNBin,
		FortiTools: cfg.VPNTools,
		WireGuard:  cfg.WGBin,
		Zscaler:    cfg.ZSABin,
	}, vpn)
}

// VPNTarget is the mapped customer VPN for this Pathfinder (or CRT) session.
// Empty means no VPN — the session stays on the default route.
func VPNTarget(folder, name, host string) (string, crtbridge.Settings) {
	cfg, err := crtbridge.LoadSettings(crtapp.Home())
	if err != nil {
		return "", crtbridge.Settings{}
	}
	return vpnForSession(cfg, folder, name, host), cfg
}

// DialerFor returns a TCP dialer bound to the customer VPN adapter when we
// can name it. Fail-open: unmapped sessions and unmatched adapters use the
// default route (a plain Dialer).
func DialerFor(folder, name, host string) *net.Dialer {
	vpn, _ := VPNTarget(folder, name, host)
	if vpn == "" {
		return &net.Dialer{}
	}
	return vpnprov.Dialer(vpn)
}

func vpnForSession(cfg crtbridge.Settings, folder, name, host string) string {
	customer := strings.TrimSpace(sessions.CustomerOfFolder(folder))
	if customer == "" {
		customer = cfg.HostFolder(name, host)
	}
	if customer == "" {
		customer = strings.TrimSpace(folder)
	}
	rel := ""
	if folder != "" && name != "" {
		rel = strings.ReplaceAll(folder, `\`, `/`) + "/" + name + ".ini"
	}
	return cfg.VPNTunnelForSession(rel, customer)
}

func BindHost(appHome, key, folder string) error {
	key, folder = strings.TrimSpace(key), strings.TrimSpace(folder)
	if key == "" || folder == "" {
		return fmt.Errorf("bind NAME FOLDER")
	}
	if appHome == "" {
		appHome = crtapp.Home()
	}
	s, err := crtbridge.LoadSettings(appHome)
	if err != nil {
		return err
	}
	s.HostBinds[key] = folder
	return crtbridge.SaveSettings(appHome, s)
}

func UnbindHost(appHome, key string) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return fmt.Errorf("unbind NAME")
	}
	if appHome == "" {
		appHome = crtapp.Home()
	}
	s, err := crtbridge.LoadSettings(appHome)
	if err != nil {
		return err
	}
	delete(s.HostBinds, key)
	for k := range s.HostBinds {
		if strings.EqualFold(k, key) {
			delete(s.HostBinds, k)
		}
	}
	return crtbridge.SaveSettings(appHome, s)
}

func MapSet(appHome, folder, target string, auvik bool) error {
	folder, target = strings.TrimSpace(folder), strings.TrimSpace(target)
	if folder == "" || target == "" {
		return fmt.Errorf("folder and target required")
	}
	if appHome == "" {
		appHome = crtapp.Home()
	}
	s, err := crtbridge.LoadSettings(appHome)
	if err != nil {
		return err
	}
	if auvik {
		s.AuvikTenants[folder] = target
	} else {
		s.VPNTunnels[folder] = target
	}
	return crtbridge.SaveSettings(appHome, s)
}

func InteractiveStdin() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
