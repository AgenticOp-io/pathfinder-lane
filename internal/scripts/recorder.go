package scripts

import (
	"regexp"
	"strings"
	"sync"
	"time"
)

// Recorder captures Send steps for later Save into scripts.yaml.
type Recorder struct {
	mu          sync.Mutex
	name        string
	steps       []Step
	lastAt      time.Time
	minGap      time.Duration
	active      bool
	pendingWait bool // last Send ended with Enter; awaiting prompt via NoteOutput
	outBuf      string // recent output while pendingWait
}

// NewRecorder starts an empty recording named name.
func NewRecorder(name string) *Recorder {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Recorded"
	}
	return &Recorder{
		name:   name,
		minGap: 200 * time.Millisecond,
		active: true,
		lastAt: time.Now(),
	}
}

// Active reports whether the recorder is accepting input.
func (r *Recorder) Active() bool {
	if r == nil {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.active
}

// Name is the script name being recorded.
func (r *Recorder) Name() string {
	if r == nil {
		return ""
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.name
}

// Stop ends capture; further Note calls are ignored.
func (r *Recorder) Stop() {
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.active = false
	r.pendingWait = false
	r.outBuf = ""
}

// Note appends a Send step, inserting DelayMs when idle between keystrokes.
func (r *Recorder) Note(send string) {
	if r == nil || send == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.active {
		return
	}
	now := time.Now()
	if !r.lastAt.IsZero() && len(r.steps) > 0 {
		gap := now.Sub(r.lastAt)
		if gap >= r.minGap {
			ms := int(gap / time.Millisecond)
			if ms > 60000 {
				ms = 60000
			}
			// Attach delay to previous step when it has no delay yet.
			prev := &r.steps[len(r.steps)-1]
			if prev.DelayMs == 0 {
				prev.DelayMs = ms
			} else {
				r.steps = append(r.steps, Step{DelayMs: ms})
			}
		}
	}
	r.steps = append(r.steps, Step{Send: send})
	r.lastAt = now
	// A send that ends with Enter is a command; wait for the next prompt.
	if strings.ContainsAny(send, "\r\n") {
		r.pendingWait = true
		r.outBuf = ""
	}
}

// promptTail matches common CLI prompts at end of output (Cisco/Juniper/Linux).
var promptTail = regexp.MustCompile(`(?m)[\w.@{}\[\]/~:-]+[#>\$]\s*$`)

// InferPrompt returns a wait_for string from recent terminal output, or "".
func InferPrompt(output string) string {
	s := strings.TrimRight(output, " \t")
	if s == "" {
		return ""
	}
	lines := strings.Split(s, "\n")
	var last string
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimRight(lines[i], "\r \t")
		if line != "" {
			last = line
			break
		}
	}
	if last == "" || !promptTail.MatchString(last) {
		return ""
	}
	runes := []rune(last)
	if len(runes) <= 24 {
		return last
	}
	ch := runes[len(runes)-1]
	if ch == '#' || ch == '>' || ch == '$' {
		return string(ch)
	}
	return ""
}

// NoteOutput inspects session output and, when a prompt appears after a
// recorded command, attaches wait_for to that step (auto-detection).
func (r *Recorder) NoteOutput(chunk string) {
	if r == nil || chunk == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.active || !r.pendingWait || len(r.steps) == 0 {
		return
	}
	r.outBuf += chunk
	if len(r.outBuf) > 8192 {
		r.outBuf = r.outBuf[len(r.outBuf)-8192:]
	}
	prompt := InferPrompt(r.outBuf)
	if prompt == "" {
		return
	}
	prev := &r.steps[len(r.steps)-1]
	if prev.Send == "" || prev.WaitFor != "" {
		r.pendingWait = false
		r.outBuf = ""
		return
	}
	prev.WaitFor = prompt
	if prev.TimeoutMs == 0 {
		prev.TimeoutMs = 15000
	}
	r.pendingWait = false
	r.outBuf = ""
}

// Script returns a copy of the recorded script.
func (r *Recorder) Script() Script {
	if r == nil {
		return Script{}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	out := Script{Name: r.name, Scope: "active", Steps: make([]Step, len(r.steps))}
	copy(out.Steps, r.steps)
	return out
}

// Upsert replaces or appends script into f and returns the updated file.
func Upsert(f File, s Script) File {
	s.Name = strings.TrimSpace(s.Name)
	if s.Name == "" {
		s.Name = "Recorded"
	}
	if s.Scope == "" {
		s.Scope = "active"
	}
	for i := range f.Scripts {
		if strings.EqualFold(f.Scripts[i].Name, s.Name) {
			f.Scripts[i] = s
			return f
		}
	}
	f.Scripts = append(f.Scripts, s)
	if f.Version == 0 {
		f.Version = 1
	}
	return f
}
