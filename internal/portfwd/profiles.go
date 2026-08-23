package portfwd

import (
	"encoding/json"
	"os"
	"path/filepath"
)

const profilesFile = "port-forward-profiles.json"

// Profile is a saved forward preset (local / remote / dynamic).
type Profile struct {
	Name       string `json:"name"`
	Kind       string `json:"kind"` // local, remote, dynamic
	ListenAddr string `json:"listen_addr"`
	TargetAddr string `json:"target_addr,omitempty"`
}

// ProfilesPath returns the catalog file under app home.
func ProfilesPath(appHome string) string {
	return filepath.Join(appHome, profilesFile)
}

// LoadProfiles reads saved profiles.
func LoadProfiles(appHome string) ([]Profile, error) {
	path := ProfilesPath(appHome)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Profile
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// SaveProfiles writes profiles to app home.
func SaveProfiles(appHome string, profiles []Profile) error {
	raw, err := json.MarshalIndent(profiles, "", "  ")
	if err != nil {
		return err
	}
	path := ProfilesPath(appHome)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o600)
}

// UpsertProfile adds or replaces a profile by name.
func UpsertProfile(profiles []Profile, p Profile) []Profile {
	for i, existing := range profiles {
		if existing.Name == p.Name {
			profiles[i] = p
			return profiles
		}
	}
	return append(profiles, p)
}

// SpecFromProfile converts a profile to a runtime spec.
func SpecFromProfile(p Profile) Spec {
	spec := Spec{ListenAddr: p.ListenAddr, TargetAddr: p.TargetAddr}
	switch p.Kind {
	case "remote":
		spec.Kind = Remote
	case "dynamic":
		spec.Kind = Dynamic
	default:
		spec.Kind = Local
	}
	return spec
}
