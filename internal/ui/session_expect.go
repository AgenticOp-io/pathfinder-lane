package ui

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/scottpeterman/pathfinderssh/internal/scripts"
)

const sessionOutRecentCap = 64 * 1024

type outWatcher struct {
	ch chan string
}

// append recent output and notify watchers (called from readLoop).
func (s *Session) noteOutput(data []byte) {
	if s == nil || len(data) == 0 {
		return
	}
	chunk := string(data)
	s.outMu.Lock()
	s.recentOut.WriteString(chunk)
	if s.recentOut.Len() > sessionOutRecentCap {
		trim := s.recentOut.String()
		s.recentOut.Reset()
		s.recentOut.WriteString(trim[len(trim)-sessionOutRecentCap:])
	}
	watchers := append([]*outWatcher(nil), s.outWatch...)
	s.outMu.Unlock()
	for _, w := range watchers {
		select {
		case w.ch <- chunk:
		default:
			// Slow waiter — drop chunk; they still have the rolling buffer start.
		}
	}
}

// WatchOutput subscribes to raw session output. Call the returned cancel to unsubscribe.
func (s *Session) WatchOutput() (<-chan string, func()) {
	if s == nil {
		ch := make(chan string)
		close(ch)
		return ch, func() {}
	}
	w := &outWatcher{ch: make(chan string, 64)}
	s.outMu.Lock()
	s.outWatch = append(s.outWatch, w)
	s.outMu.Unlock()
	var once sync.Once
	unsub := func() {
		once.Do(func() {
			s.outMu.Lock()
			defer s.outMu.Unlock()
			out := s.outWatch[:0]
			for _, x := range s.outWatch {
				if x != w {
					out = append(out, x)
				}
			}
			s.outWatch = out
			close(w.ch)
		})
	}
	return w.ch, unsub
}

// RecentOutput returns a copy of the trailing output buffer.
func (s *Session) RecentOutput() string {
	if s == nil {
		return ""
	}
	s.outMu.Lock()
	defer s.outMu.Unlock()
	return s.recentOut.String()
}

// WaitForPattern blocks until pattern matches recent+live output.
func (s *Session) WaitForPattern(ctx context.Context, pattern string, useRegex bool, timeout time.Duration) error {
	if s == nil {
		return fmt.Errorf("no session")
	}
	if !s.Connected() {
		return fmt.Errorf("session not connected")
	}
	if timeout <= 0 {
		timeout = time.Duration(scripts.DefaultWaitTimeoutMs) * time.Millisecond
	}
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return nil
	}

	ch, unsub := s.WatchOutput()
	defer unsub()

	var buf strings.Builder
	buf.WriteString(s.RecentOutput())
	if ok, err := scripts.MatchBuffer(buf.String(), pattern, useRegex); err != nil {
		return err
	} else if ok {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				return fmt.Errorf("timed out after %s waiting for %q", timeout, pattern)
			}
			return ctx.Err()
		case chunk, ok := <-ch:
			if !ok {
				return fmt.Errorf("session closed while waiting for %q", pattern)
			}
			buf.WriteString(chunk)
			if buf.Len() > sessionOutRecentCap {
				s := buf.String()
				buf.Reset()
				buf.WriteString(s[len(s)-sessionOutRecentCap:])
			}
			if matched, err := scripts.MatchBuffer(buf.String(), pattern, useRegex); err != nil {
				return err
			} else if matched {
				return nil
			}
		}
	}
}
