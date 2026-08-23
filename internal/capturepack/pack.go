package capturepack

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/scottpeterman/pathfinderssh/internal/capture"
	"github.com/scottpeterman/pathfinderssh/internal/evidence"
	"github.com/scottpeterman/pathfinderssh/internal/ui"
)

// Options configures a post-change capture pack.
type Options struct {
	IncidentID     string
	Customer       string
	AppHome        string
	StorePath      string
	LinkedHosts    []string
	Scrollbacks    []evidence.File
	IncludeMap     bool
	IncludeConfigs bool
}

// Collect appends map JSON and running-config artifacts to scrollback files.
func Collect(opts Options) ([]evidence.File, error) {
	out := append([]evidence.File{}, opts.Scrollbacks...)
	if opts.IncludeMap && strings.TrimSpace(opts.Customer) != "" {
		if f, err := latestMapFile(opts.AppHome, opts.Customer); err == nil && f != nil {
			out = append(out, *f)
		}
	}
	if opts.IncludeConfigs && strings.TrimSpace(opts.StorePath) != "" {
		cfg, err := configFiles(opts.StorePath, opts.LinkedHosts)
		if err != nil {
			return out, err
		}
		out = append(out, cfg...)
	}
	return out, nil
}

func latestMapFile(appHome, customer string) (*evidence.File, error) {
	dir := ui.CustomerMapsDir(appHome, customer)
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var jsonFiles []string
	for _, e := range ents {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
			jsonFiles = append(jsonFiles, e.Name())
		}
	}
	if len(jsonFiles) == 0 {
		return nil, os.ErrNotExist
	}
	sort.Strings(jsonFiles)
	name := jsonFiles[len(jsonFiles)-1]
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return nil, err
	}
	return &evidence.File{
		Name:    "maps/" + customer + "-" + name,
		Content: data,
	}, nil
}

func configFiles(storeRoot string, hosts []string) ([]evidence.File, error) {
	st, err := capture.OpenFileStore(storeRoot)
	if err != nil {
		return nil, err
	}
	devs, err := st.Devices()
	if err != nil {
		return nil, err
	}
	want := hostSet(hosts)
	var out []evidence.File
	for _, d := range devs {
		if !deviceMatches(d, want) {
			continue
		}
		types, err := st.Types(d.Canonical)
		if err != nil {
			continue
		}
		for _, ti := range types {
			if !strings.Contains(strings.ToLower(ti.Type), "running") {
				continue
			}
			if ti.File == "" {
				continue
			}
			data, err := st.Read(d.Canonical, ti.Type, ti.File)
			if err != nil || len(data) == 0 {
				continue
			}
			out = append(out, evidence.File{
				Name:    "configs/" + sanitize(d.Canonical) + "-" + ti.Type + ".txt",
				Content: data,
			})
		}
	}
	return out, nil
}

func hostSet(hosts []string) map[string]bool {
	m := make(map[string]bool)
	for _, h := range hosts {
		h = strings.TrimSpace(strings.ToLower(h))
		if h != "" {
			m[h] = true
		}
	}
	return m
}

func deviceMatches(d capture.DeviceInfo, want map[string]bool) bool {
	if len(want) == 0 {
		return true
	}
	check := func(s string) bool {
		s = strings.TrimSpace(strings.ToLower(s))
		return s != "" && want[s]
	}
	if check(d.Canonical) {
		return true
	}
	for _, a := range d.Aliases {
		if check(a) {
			return true
		}
	}
	for w := range want {
		if strings.Contains(strings.ToLower(d.Canonical), w) {
			return true
		}
	}
	return false
}

func sanitize(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, s)
}

// PackName builds a local zip filename for an incident capture pack.
func PackName(incidentID string) string {
	id := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, incidentID)
	return "pathfinder-capturepack-" + id + "-" + time.Now().Format("20060102-150405") + ".zip"
}
