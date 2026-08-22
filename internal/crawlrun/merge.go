package crawlrun

import (
	"strings"
)

// MergeSuggestion proposes combining two crawl rows (duplicate IP or low confidence).
type MergeSuggestion struct {
	IdentityA string
	IdentityB string
	NameA     string
	NameB     string
	Reason    string
}

// MergeSuggestions returns heuristic duplicate-device hints for the crawl table.
func MergeSuggestions(rows []DeviceRow) []MergeSuggestion {
	var out []MergeSuggestion
	byIdentity := map[string][]DeviceRow{}
	for _, r := range rows {
		id := strings.TrimSpace(r.Identity)
		if id == "" {
			continue
		}
		byIdentity[id] = append(byIdentity[id], r)
	}
	for id, group := range byIdentity {
		if len(group) < 2 {
			continue
		}
		a, b := group[0], group[1]
		out = append(out, MergeSuggestion{
			IdentityA: id,
			IdentityB: id,
			NameA:     a.Display(),
			NameB:     b.Display(),
			Reason:    "same address, different names — review for duplicate device",
		})
	}
	for i := 0; i < len(rows); i++ {
		for j := i + 1; j < len(rows); j++ {
			a, b := rows[i], rows[j]
			if strings.TrimSpace(a.Identity) == "" || a.Identity != b.Identity {
				continue
			}
			if a.Display() == b.Display() {
				continue
			}
			if a.Confidence() >= 50 && b.Confidence() >= 50 {
				continue
			}
			out = append(out, MergeSuggestion{
				IdentityA: a.Identity,
				IdentityB: b.Identity,
				NameA:     a.Display(),
				NameB:     b.Display(),
				Reason:    "low confidence duplicate IP — consider merging sessions",
			})
		}
	}
	return out
}
