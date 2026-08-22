package portfwd

import (
	"fmt"
	"io"
	"net"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
)

// Kind of SSH port forward.
type Kind string

const (
	Local   Kind = "local"   // listen locally, dial remote via SSH
	Remote  Kind = "remote"  // listen on remote, dial local
	Dynamic Kind = "dynamic" // local SOCKS5 proxy via SSH
)

// Spec describes one forward to start.
type Spec struct {
	Kind       Kind
	ListenAddr string // e.g. 127.0.0.1:1080
	TargetAddr string // host:port for Local/Remote; unused for Dynamic
}

// Handle is a running forward; call Close to stop.
type Handle struct {
	Spec Spec
	ln   net.Listener
	done chan struct{}
	wg   sync.WaitGroup
}

// Close stops accepting and waits for active pipes to finish.
func (h *Handle) Close() error {
	if h == nil {
		return nil
	}
	select {
	case <-h.done:
	default:
		close(h.done)
	}
	var err error
	if h.ln != nil {
		err = h.ln.Close()
	}
	h.wg.Wait()
	return err
}

// Start begins a forward over client. The SSH client must stay open.
func Start(client *ssh.Client, spec Spec) (*Handle, error) {
	if client == nil {
		return nil, fmt.Errorf("ssh client is nil")
	}
	spec.ListenAddr = strings.TrimSpace(spec.ListenAddr)
	spec.TargetAddr = strings.TrimSpace(spec.TargetAddr)
	if spec.ListenAddr == "" {
		return nil, fmt.Errorf("listen address required")
	}
	switch spec.Kind {
	case Local:
		if spec.TargetAddr == "" {
			return nil, fmt.Errorf("target address required for local forward")
		}
		return startLocal(client, spec)
	case Remote:
		if spec.TargetAddr == "" {
			return nil, fmt.Errorf("target address required for remote forward")
		}
		return startRemote(client, spec)
	case Dynamic:
		return startDynamic(client, spec)
	default:
		return nil, fmt.Errorf("unknown forward kind %q", spec.Kind)
	}
}

func startLocal(client *ssh.Client, spec Spec) (*Handle, error) {
	ln, err := net.Listen("tcp", spec.ListenAddr)
	if err != nil {
		return nil, err
	}
	h := &Handle{Spec: spec, ln: ln, done: make(chan struct{})}
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		for {
			local, err := ln.Accept()
			if err != nil {
				return
			}
			h.wg.Add(1)
			go func(c net.Conn) {
				defer h.wg.Done()
				defer c.Close()
				remote, err := client.Dial("tcp", spec.TargetAddr)
				if err != nil {
					return
				}
				defer remote.Close()
				pipe(c, remote)
			}(local)
		}
	}()
	return h, nil
}

func startRemote(client *ssh.Client, spec Spec) (*Handle, error) {
	ln, err := client.Listen("tcp", spec.ListenAddr)
	if err != nil {
		return nil, err
	}
	h := &Handle{Spec: spec, ln: ln, done: make(chan struct{})}
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		for {
			remote, err := ln.Accept()
			if err != nil {
				return
			}
			h.wg.Add(1)
			go func(c net.Conn) {
				defer h.wg.Done()
				defer c.Close()
				local, err := net.Dial("tcp", spec.TargetAddr)
				if err != nil {
					return
				}
				defer local.Close()
				pipe(c, local)
			}(remote)
		}
	}()
	return h, nil
}

func startDynamic(client *ssh.Client, spec Spec) (*Handle, error) {
	ln, err := net.Listen("tcp", spec.ListenAddr)
	if err != nil {
		return nil, err
	}
	h := &Handle{Spec: spec, ln: ln, done: make(chan struct{})}
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			h.wg.Add(1)
			go func(c net.Conn) {
				defer h.wg.Done()
				defer c.Close()
				serveSOCKS5(c, client)
			}(conn)
		}
	}()
	return h, nil
}

func pipe(a, b net.Conn) {
	done := make(chan struct{}, 2)
	cp := func(dst, src net.Conn) {
		_, _ = io.Copy(dst, src)
		done <- struct{}{}
	}
	go cp(a, b)
	go cp(b, a)
	<-done
}
