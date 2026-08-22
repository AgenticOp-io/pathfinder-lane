package scripts

import (
	"sort"
	"strings"
)

// RankNames orders script names by relevance to customer / platform / notes.
// Exact substring hits in the script name score highest; unmatched names keep
// their relative order after ranked ones.
func RankNames(names []string, hints ...string) []string {
	if len(names) == 0 {
		return nil
	}
	type scored struct {
		name  string
		score int
		idx   int
	}
	tokens := hintTokens(hints...)
	out := make([]scored, len(names))
	for i, n := range names {
		out[i] = scored{name: n, score: scoreName(n, tokens), idx: i}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].score != out[j].score {
			return out[i].score > out[j].score
		}
		return out[i].idx < out[j].idx
	})
	namesOut := make([]string, len(out))
	for i := range out {
		namesOut[i] = out[i].name
	}
	return namesOut
}

func hintTokens(hints ...string) []string {
	seen := map[string]struct{}{}
	var toks []string
	for _, h := range hints {
		for _, p := range strings.FieldsFunc(strings.ToLower(h), func(r rune) bool {
			return r == ' ' || r == ',' || r == '/' || r == '-' || r == '_' || r == '.' || r == '\t' || r == '\n'
		}) {
			p = strings.TrimSpace(p)
			if len(p) < 3 {
				continue
			}
			if _, ok := seen[p]; ok {
				continue
			}
			seen[p] = struct{}{}
			toks = append(toks, p)
		}
	}
	return toks
}

func scoreName(name string, tokens []string) int {
	if len(tokens) == 0 {
		return 0
	}
	lower := strings.ToLower(name)
	score := 0
	for _, t := range tokens {
		if strings.Contains(lower, t) {
			score += 10 + len(t)
		}
	}
	return score
}
