package sftpclient

import (
	"errors"
	"io"
	"sync"
	"sync/atomic"
)

// ErrStopped is returned when TransferControl.Stop ends a copy.
var ErrStopped = errors.New("transfer stopped")

const (
	stateRunning int32 = 0
	statePaused  int32 = 1
	stateStopped int32 = 2
)

// TransferControl pauses, resumes, or stops an in-flight copy.
type TransferControl struct {
	state  atomic.Int32
	mu     sync.Mutex
	cond   *sync.Cond
	onStop []func()
}

// NewTransferControl returns a ready control (running, not paused).
func NewTransferControl() *TransferControl {
	c := &TransferControl{}
	c.cond = sync.NewCond(&c.mu)
	return c
}

// Pause blocks the next Read until Resume or Stop.
func (c *TransferControl) Pause() {
	if c == nil {
		return
	}
	c.state.Store(statePaused)
}

// Resume continues a paused transfer.
func (c *TransferControl) Resume() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.state.Store(stateRunning)
	c.cond.Broadcast()
	c.mu.Unlock()
}

// OnStop registers a hook invoked when Stop runs (e.g. close a blocked file).
func (c *TransferControl) OnStop(fn func()) {
	if c == nil || fn == nil {
		return
	}
	c.mu.Lock()
	if c.state.Load() == stateStopped {
		c.mu.Unlock()
		go fn()
		return
	}
	c.onStop = append(c.onStop, fn)
	c.mu.Unlock()
}

// Stop aborts the transfer; the copy returns ErrStopped.
func (c *TransferControl) Stop() {
	if c == nil {
		return
	}
	c.mu.Lock()
	if c.state.Load() == stateStopped {
		c.mu.Unlock()
		return
	}
	c.state.Store(stateStopped)
	hooks := c.onStop
	c.onStop = nil
	c.cond.Broadcast()
	c.mu.Unlock()
	for _, fn := range hooks {
		if fn != nil {
			go fn()
		}
	}
}

// Stopped reports whether Stop was called.
func (c *TransferControl) Stopped() bool {
	if c == nil {
		return false
	}
	return c.state.Load() == stateStopped
}

// wait is a hot path: when running it is a single atomic load (no mutex).
func (c *TransferControl) wait() error {
	if c == nil {
		return nil
	}
	for {
		switch c.state.Load() {
		case stateRunning:
			return nil
		case stateStopped:
			return ErrStopped
		}
		c.mu.Lock()
		for c.state.Load() == statePaused {
			c.cond.Wait()
		}
		st := c.state.Load()
		c.mu.Unlock()
		if st == stateStopped {
			return ErrStopped
		}
	}
}

// gatedReader checks pause/stop once per Read. Avoid locking on the
// running path — the old version took the mutex twice per chunk.
type gatedReader struct {
	r    io.Reader
	ctrl *TransferControl
}

func (g *gatedReader) Read(p []byte) (int, error) {
	if err := g.ctrl.wait(); err != nil {
		return 0, err
	}
	return g.r.Read(p)
}

func gated(r io.Reader, ctrl *TransferControl) io.Reader {
	if ctrl == nil {
		return r
	}
	return &gatedReader{r: r, ctrl: ctrl}
}
