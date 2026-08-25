package crtbridge

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"testing"
	"time"
)

func TestHandleVPNFailureStillDialsOriginal(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		c, err := ln.Accept()
		if err != nil {
			return
		}
		defer c.Close()
		_, _ = io.Copy(c, c)
	}()
	_, portStr, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatal(err)
	}

	orig := ensureVPN
	ensureVPN = func(context.Context, Settings, string) error {
		return fmt.Errorf("no FortiClient CLI")
	}
	t.Cleanup(func() { ensureVPN = orig })

	a := &Agent{
		cfg:  Settings{Mode: AutoFortiClient},
		Log:  log.New(io.Discard, "", 0),
		Opts: Options{AppHome: t.TempDir()},
	}
	client, server := net.Pipe()
	defer client.Close()
	go a.handle(server, Session{
		VPNTunnel:    "Aspire",
		Customer:     "Aspire",
		OriginalHost: "127.0.0.1",
		OriginalPort: port,
	})

	_ = client.SetDeadline(time.Now().Add(5 * time.Second))
	if _, err := client.Write([]byte("ping")); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 4)
	if _, err := io.ReadFull(client, buf); err != nil {
		t.Fatalf("VPN failure must still splice to original host: %v", err)
	}
	if string(buf) != "ping" {
		t.Fatalf("got %q", buf)
	}
}
