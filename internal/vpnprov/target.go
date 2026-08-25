// Package vpnprov is the last-mile VPN bus: official CLIs only
// (FortiClient, WireGuard, Zscaler Client Connector). It does not
// reimplement those protocols or pass passwords on the command line.
package vpnprov

import "strings"

const (
	FortiClient = "forticlient"
	WireGuard   = "wireguard"
	Zscaler     = "zscaler"
)

// Target is one mapped customer VPN. Kind is the vendor; Name is what
// that vendor's CLI expects (Forti connection, WireGuard interface, or
// zpa / zpa:partner-user).
type Target struct {
	Kind string
	Name string
}

// ParseTarget reads installer map values.
//
//	aspire-ssl              → forticlient / aspire-ssl  (legacy)
//	forticlient:aspire-ssl  → forticlient / aspire-ssl
//	wireguard:acme          → wireguard / acme
//	wg:acme                 → wireguard / acme
//	zscaler:zpa             → zscaler / zpa
//	zscaler:zpa:user@x      → zscaler / zpa:user@x
//	zpa                     → zscaler / zpa
func ParseTarget(raw string) Target {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Target{}
	}
	low := strings.ToLower(raw)
	switch {
	case low == "zpa" || strings.HasPrefix(low, "zpa:"):
		return Target{Kind: Zscaler, Name: raw}
	case strings.HasPrefix(low, "wireguard:"):
		return Target{Kind: WireGuard, Name: strings.TrimSpace(raw[len("wireguard:"):])}
	case strings.HasPrefix(low, "wg:"):
		return Target{Kind: WireGuard, Name: strings.TrimSpace(raw[len("wg:"):])}
	case strings.HasPrefix(low, "zscaler:"):
		name := strings.TrimSpace(raw[len("zscaler:"):])
		if name == "" {
			name = "zpa"
		}
		return Target{Kind: Zscaler, Name: name}
	case strings.HasPrefix(low, "forticlient:"):
		return Target{Kind: FortiClient, Name: strings.TrimSpace(raw[len("forticlient:"):])}
	case strings.HasPrefix(low, "forti:"):
		return Target{Kind: FortiClient, Name: strings.TrimSpace(raw[len("forti:"):])}
	default:
		return Target{Kind: FortiClient, Name: raw}
	}
}

// FormatTarget is Folder=value form. Bare Forti names stay bare for older maps.
func FormatTarget(t Target) string {
	t.Kind = strings.ToLower(strings.TrimSpace(t.Kind))
	t.Name = strings.TrimSpace(t.Name)
	if t.Name == "" {
		return ""
	}
	switch t.Kind {
	case WireGuard:
		return "wireguard:" + t.Name
	case Zscaler:
		return "zscaler:" + t.Name
	case FortiClient, "":
		return t.Name
	default:
		return t.Kind + ":" + t.Name
	}
}

func (t Target) String() string {
	return FormatTarget(t)
}

// ShortName is the vendor-side label used for SuggestFolder.
func (t Target) ShortName() string {
	name := t.Name
	if t.Kind == Zscaler {
		if _, rest, ok := strings.Cut(name, ":"); ok {
			name = rest
		}
	}
	return name
}
