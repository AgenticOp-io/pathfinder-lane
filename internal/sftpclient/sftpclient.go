// Package sftpclient is a thin SFTP helper over an existing ssh.Client.
//
// Throughput notes:
//   - pkg/sftp File implements WriterTo/ReaderFrom with concurrent requests.
//     Wrapping the remote file in a plain Reader/Writer disables that path —
//     pause/stop/progress hang off the *local* side (or a thin counter) so
//     concurrent reads/writes stay enabled.
//   - io.CopyBuffer uses a 1 MiB buffer so even the fallback sequential path
//     issues fewer round-trips than the default 32 KiB Copy buffer.
package sftpclient

import (
	"fmt"
	"io"
	"os"
	"path"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

const (
	opTimeout = 25 * time.Second
	copyBufSize = 1024 * 1024
	// Keep concurrency modest: SFTP shares the SSH mux with the live terminal.
	// 64 in-flight requests starved the session channel and froze the UI.
	maxConcurrentPerFile = 8
	progressEvery        = 250 * time.Millisecond
)

// Entry is one remote directory row.
type Entry struct {
	Name    string
	Path    string
	IsDir   bool
	Size    int64
	ModTime time.Time
}

// Client wraps pkg/sftp. The mutex only guards the session pointer.
type Client struct {
	c  *sftp.Client
	mu sync.Mutex
}

func Open(sshClient *ssh.Client) (*Client, error) {
	type result struct {
		cli *Client
		err error
	}
	ch := make(chan result, 1)
	go func() {
		// Concurrent reads are on by default; concurrent writes must be opted in.
		// Cap per-file concurrency: this client shares the SSH connection with
		// the interactive terminal — too many in-flight requests starve it and
		// the whole app looks frozen.
		c, err := sftp.NewClient(sshClient,
			sftp.UseConcurrentWrites(true),
			sftp.MaxConcurrentRequestsPerFile(maxConcurrentPerFile),
		)
		if err != nil {
			ch <- result{nil, err}
			return
		}
		ch <- result{&Client{c: c}, nil}
	}()
	select {
	case r := <-ch:
		return r.cli, r.err
	case <-time.After(opTimeout):
		go func() {
			if r := <-ch; r.cli != nil {
				_ = r.cli.Close()
			}
		}()
		return nil, fmt.Errorf("SFTP open timed out after %s", opTimeout)
	}
}

func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.c == nil {
		return nil
	}
	err := c.c.Close()
	c.c = nil
	return err
}

func (c *Client) session() (*sftp.Client, error) {
	if c == nil {
		return nil, fmt.Errorf("SFTP client is closed")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.c == nil {
		return nil, fmt.Errorf("SFTP client is closed")
	}
	return c.c, nil
}

func (c *Client) do(fn func(*sftp.Client) error) error {
	cli, err := c.session()
	if err != nil {
		return err
	}
	done := make(chan error, 1)
	go func() { done <- fn(cli) }()
	select {
	case err := <-done:
		return err
	case <-time.After(opTimeout):
		// Do NOT Close the shared client on timeout. Closing aborts in-flight
		// uploads/downloads on the same session and can wedge the SSH mux
		// (terminal + SFTP), which presents as a full UI freeze. The orphaned
		// goroutine may still finish later; the caller just sees a timeout.
		return fmt.Errorf("SFTP operation timed out after %s", opTimeout)
	}
}

func (c *Client) doTransfer(fn func(*sftp.Client) error) error {
	cli, err := c.session()
	if err != nil {
		return err
	}
	return fn(cli)
}

// ProgressFunc reports bytes copied so far and the known total (0 if unknown).
type ProgressFunc func(done, total int64)

// progressWriter counts bytes, throttles UI callbacks, and honors pause/stop
// on Write so the remote sftp.File can keep its WriterTo/ReadFrom fast paths.
type progressWriter struct {
	w     io.Writer
	total int64
	done  int64
	fn    ProgressFunc
	ctrl  *TransferControl
	mu    sync.Mutex
	last  time.Time
}

func (p *progressWriter) Write(b []byte) (int, error) {
	if err := p.ctrl.wait(); err != nil {
		return 0, err
	}
	n, err := p.w.Write(b)
	p.mu.Lock()
	p.done += int64(n)
	p.maybeReportLocked(err != nil)
	p.mu.Unlock()
	return n, err
}

func (p *progressWriter) maybeReportLocked(force bool) {
	if p.fn == nil {
		return
	}
	now := time.Now()
	if !force && !(p.total > 0 && p.done >= p.total) && now.Sub(p.last) < progressEvery {
		return
	}
	p.last = now
	done, total := p.done, p.total
	fn := p.fn
	// Callback outside the lock so a slow UI hop cannot stall the copy.
	go fn(done, total)
}

// progressReader counts bytes read and honors pause/stop — used as the source
// for sftp.File.ReadFrom so concurrent uploads stay enabled.
type progressReader struct {
	r     io.Reader
	total int64
	done  int64
	fn    ProgressFunc
	ctrl  *TransferControl
	mu    sync.Mutex
	last  time.Time
}

func (p *progressReader) Read(b []byte) (int, error) {
	if err := p.ctrl.wait(); err != nil {
		return 0, err
	}
	n, err := p.r.Read(b)
	p.mu.Lock()
	p.done += int64(n)
	if p.fn != nil {
		now := time.Now()
		if err != nil || (p.total > 0 && p.done >= p.total) || now.Sub(p.last) >= progressEvery {
			p.last = now
			done, total := p.done, p.total
			fn := p.fn
			p.mu.Unlock()
			go fn(done, total)
			return n, err
		}
	}
	p.mu.Unlock()
	return n, err
}

func (c *Client) List(dir string) ([]Entry, error) {
	if dir == "" {
		dir = "."
	}
	var out []Entry
	err := c.do(func(cli *sftp.Client) error {
		fis, err := cli.ReadDir(dir)
		if err != nil {
			return err
		}
		out = make([]Entry, 0, len(fis))
		for _, fi := range fis {
			out = append(out, Entry{
				Name:    fi.Name(),
				Path:    path.Join(dir, fi.Name()),
				IsDir:   fi.IsDir(),
				Size:    fi.Size(),
				ModTime: fi.ModTime(),
			})
		}
		return nil
	})
	return out, err
}

func (c *Client) Download(remote, local string) error {
	return c.DownloadProgress(remote, local, nil, nil)
}

// DownloadProgress copies remote → local with progress and optional pause/stop.
func (c *Client) DownloadProgress(remote, local string, progress ProgressFunc, ctrl *TransferControl) error {
	return c.doTransfer(func(cli *sftp.Client) error {
		var total int64
		if st, err := cli.Stat(remote); err == nil {
			total = st.Size()
		}
		rf, err := cli.Open(remote)
		if err != nil {
			return err
		}
		defer rf.Close()
		lf, err := os.Create(local)
		if err != nil {
			return err
		}
		defer lf.Close()

		pw := &progressWriter{w: lf, total: total, fn: progress, ctrl: ctrl}
		// Prefer WriterTo so pkg/sftp can pipeline concurrent reads.
		if wt, ok := any(rf).(io.WriterTo); ok {
			_, err = wt.WriteTo(pw)
		} else {
			buf := make([]byte, copyBufSize)
			_, err = io.CopyBuffer(pw, rf, buf)
		}
		if progress != nil {
			progress(pw.done, total)
		}
		return err
	})
}

func (c *Client) Upload(local, remote string) error {
	return c.UploadProgress(local, remote, nil, nil)
}

// UploadProgress copies local → remote with progress and optional pause/stop.
func (c *Client) UploadProgress(local, remote string, progress ProgressFunc, ctrl *TransferControl) error {
	return c.doTransfer(func(cli *sftp.Client) error {
		var total int64
		if st, err := os.Stat(local); err == nil {
			total = st.Size()
		}
		lf, err := os.Open(local)
		if err != nil {
			return err
		}
		defer lf.Close()
		rf, err := cli.Create(remote)
		if err != nil {
			return err
		}
		defer rf.Close()

		pr := &progressReader{r: lf, total: total, fn: progress, ctrl: ctrl}
		// Prefer ReaderFrom so pkg/sftp can pipeline concurrent writes.
		if rt, ok := any(rf).(io.ReaderFrom); ok {
			_, err = rt.ReadFrom(pr)
		} else {
			buf := make([]byte, copyBufSize)
			_, err = io.CopyBuffer(rf, pr, buf)
		}
		if progress != nil {
			progress(pr.done, total)
		}
		return err
	})
}

func (c *Client) Mkdir(remote string) error {
	return c.do(func(cli *sftp.Client) error {
		return cli.Mkdir(remote)
	})
}

func (c *Client) Remove(remote string) error {
	return c.do(func(cli *sftp.Client) error {
		return cli.Remove(remote)
	})
}

func (c *Client) Rename(oldpath, newpath string) error {
	return c.do(func(cli *sftp.Client) error {
		return cli.Rename(oldpath, newpath)
	})
}

func (c *Client) Getwd() (string, error) {
	var wd string
	err := c.do(func(cli *sftp.Client) error {
		var e error
		wd, e = cli.Getwd()
		return e
	})
	return wd, err
}

func (c *Client) RealPath(p string) (string, error) {
	var out string
	err := c.do(func(cli *sftp.Client) error {
		var e error
		out, e = cli.RealPath(p)
		return e
	})
	return out, err
}

func Join(elem ...string) string {
	return path.Join(elem...)
}

// EnsureDir creates remote and any missing parents.
func EnsureDir(c *Client, remote string) error {
	if c == nil || remote == "" || remote == "." || remote == "/" {
		return nil
	}
	return c.do(func(cli *sftp.Client) error {
		var stack []string
		for p := remote; p != "" && p != "." && p != "/"; p = path.Dir(p) {
			stack = append(stack, p)
			next := path.Dir(p)
			if next == p {
				break
			}
		}
		for i := len(stack) - 1; i >= 0; i-- {
			p := stack[i]
			if _, err := cli.Stat(p); err == nil {
				continue
			}
			if err := cli.Mkdir(p); err != nil {
				if _, statErr := cli.Stat(p); statErr == nil {
					continue
				}
				return fmt.Errorf("mkdir %s: %w", p, err)
			}
		}
		return nil
	})
}
