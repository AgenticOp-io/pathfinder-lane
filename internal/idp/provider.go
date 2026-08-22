package idp

import "strings"

// Provider names cloud identity backends.
type Provider string

const (
	ProviderLocal  Provider = "local"
	ProviderEntra  Provider = "entra"
	ProviderGoogle Provider = "google"
)

func (p Provider) Normalize() Provider {
	switch Provider(strings.ToLower(strings.TrimSpace(string(p)))) {
	case ProviderEntra:
		return ProviderEntra
	case ProviderGoogle:
		return ProviderGoogle
	default:
		return ProviderLocal
	}
}

func (p Provider) Label() string {
	switch p.Normalize() {
	case ProviderEntra:
		return "Microsoft 365 / Entra ID"
	case ProviderGoogle:
		return "Google Workspace"
	default:
		return "Local (no cloud sign-in)"
	}
}

// ChoiceLabel is the short label for install wizards and setup menus.
func (p Provider) ChoiceLabel() string {
	switch p.Normalize() {
	case ProviderEntra:
		return "Microsoft 365"
	case ProviderGoogle:
		return "Google"
	default:
		return "Just me (solo)"
	}
}

// ProviderFromChoiceLabel maps a setup-menu label back to a provider.
func ProviderFromChoiceLabel(label string) Provider {
	label = strings.TrimSpace(label)
	for _, p := range []Provider{ProviderLocal, ProviderEntra, ProviderGoogle} {
		if p.ChoiceLabel() == label {
			return p
		}
	}
	return ProviderLocal
}

func (p Provider) RequiresCloudLogin() bool {
	return p.Normalize() == ProviderEntra || p.Normalize() == ProviderGoogle
}
