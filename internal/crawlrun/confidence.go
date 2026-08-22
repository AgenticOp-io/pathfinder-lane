package crawlrun

import "strings"

// Confidence is a 0–100 score for how trustworthy a crawl row is.
//
// Heuristic (MSP ops desk): reached + known platform + neighbors beat
// failed / unknown / not-dialed leaves.
func (d DeviceRow) Confidence() int {
	score := 40
	switch d.State {
	case StateReached:
		score = 70
	case StateRunning, StateQueued:
		score = 50
	case StateFailed:
		score = 15
	case StateNotDialed:
		score = 25
	}
	plat := strings.ToLower(strings.TrimSpace(d.Platform))
	if plat != "" && plat != "unknown" {
		score += 15
	} else if plat == "unknown" || plat == "" {
		score -= 10
	}
	if d.Neighbors > 0 {
		score += 10
	}
	if d.Attempts > 2 {
		score -= 5 * (d.Attempts - 2)
		if score < 0 {
			score = 0
		}
	}
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}
	return score
}
