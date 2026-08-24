package auvik

import (
	"net"
	"testing"
)

func TestPortListeningWithoutDial(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	if !portListening(port) {
		t.Fatalf("expected portListening(%d) true while bound", port)
	}

	_ = ln.Close()
	if portListening(port) {
		t.Fatalf("expected portListening(%d) false after close", port)
	}
}

func TestPortListeningInvalid(t *testing.T) {
	if portListening(0) || portListening(-1) {
		t.Fatal("invalid ports must be false")
	}
}
