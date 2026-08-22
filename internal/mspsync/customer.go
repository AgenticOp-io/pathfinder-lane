package mspsync

import (
	"strings"
	"unicode"
)

// NormalizeKey folds a customer or organization name for fuzzy matching
// (PSA folder vs Auvik tenant vs IT Glue org vs manual entry).
func NormalizeKey(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	var b strings.Builder
	lastSpace := false
	for _, r := range strings.ToLower(name) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastSpace = false
			continue
		}
		if !lastSpace && b.Len() > 0 {
			b.WriteByte(' ')
			lastSpace = true
		}
	}
	return strings.TrimSpace(b.String())
}

// ResolveCustomerName maps an external label (PSA company, Auvik tenant, doc org)
// to an existing Customers/<name>/ folder when names differ slightly.
// Returns externalName when no folder matches (caller may create that folder).
func ResolveCustomerName(existing []string, external string) string {
	external = strings.TrimSpace(external)
	if external == "" {
		return ""
	}
	for _, n := range existing {
		if strings.TrimSpace(n) == external {
			return n
		}
	}
	extKey := NormalizeKey(external)
	if extKey == "" {
		return external
	}
	for _, n := range existing {
		if NormalizeKey(n) == extKey {
			return n
		}
	}
	var best string
	var bestScore int
	for _, n := range existing {
		score := MatchScore(n, external)
		if score > bestScore {
			bestScore = score
			best = n
		}
	}
	if bestScore >= 80 {
		return best
	}
	return external
}

// MatchScore returns 0–100 similarity between two customer labels.
func MatchScore(a, b string) int {
	aKey := NormalizeKey(a)
	bKey := NormalizeKey(b)
	if aKey == "" || bKey == "" {
		return 0
	}
	if aKey == bKey {
		return 100
	}
	if strings.Contains(aKey, bKey) || strings.Contains(bKey, aKey) {
		short, long := aKey, bKey
		if len(short) > len(long) {
			short, long = long, short
		}
		if len(long) == 0 {
			return 0
		}
		return 70 + (len(short)*30)/len(long)
	}
	aw := tokenSet(aKey)
	bw := tokenSet(bKey)
	if len(aw) == 0 || len(bw) == 0 {
		return 0
	}
	inter := 0
	for t := range aw {
		if bw[t] {
			inter++
		}
	}
	union := len(aw) + len(bw) - inter
	if union == 0 {
		return 0
	}
	return (inter * 100) / union
}

func tokenSet(key string) map[string]bool {
	out := map[string]bool{}
	for _, t := range strings.Fields(key) {
		if len(t) >= 2 {
			out[t] = true
		}
	}
	return out
}
