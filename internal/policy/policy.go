package policy

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// Policy is the MSP send/change gate applied to typing and fan-out.
type Policy struct {
	// ReadOnly blocks all interactive and scripted sends (anti-idle still allowed via SendRaw path if host chooses).
	ReadOnly bool
	// ChangeWindowStart / End are local-time HH:MM (24h). Empty = no window limit.
	// When both set, sends are allowed only inside [start, end). Overnight windows
	// (e.g. 22:00–06:00) are supported.
	ChangeWindowStart string
	ChangeWindowEnd   string
}

// Allow reports whether a send is permitted right now.
func (p Policy) Allow(now time.Time) (ok bool, reason string) {
	if p.ReadOnly {
		return false, "read-only mode is on — disable it in Settings → Ops"
	}
	start := strings.TrimSpace(p.ChangeWindowStart)
	end := strings.TrimSpace(p.ChangeWindowEnd)
	if start == "" && end == "" {
		return true, ""
	}
	if start == "" || end == "" {
		return false, "change window needs both start and end (HH:MM)"
	}
	sm, err1 := parseHHMM(start)
	em, err2 := parseHHMM(end)
	if err1 != nil || err2 != nil {
		return false, "change window times must be HH:MM"
	}
	cur := now.Hour()*60 + now.Minute()
	if sm == em {
		return true, "" // full day
	}
	inWindow := false
	if sm < em {
		inWindow = cur >= sm && cur < em
	} else {
		// overnight
		inWindow = cur >= sm || cur < em
	}
	if !inWindow {
		return false, fmt.Sprintf("outside change window %s–%s", start, end)
	}
	return true, ""
}

func parseHHMM(s string) (minutes int, err error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("bad time")
	}
	h, err := strconv.Atoi(parts[0])
	if err != nil || h < 0 || h > 23 {
		return 0, fmt.Errorf("bad hour")
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil || m < 0 || m > 59 {
		return 0, fmt.Errorf("bad minute")
	}
	return h*60 + m, nil
}
