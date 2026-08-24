package sftpclient

import (
	"errors"
	"testing"
	"time"
)

func TestTransferControlOnStop(t *testing.T) {
	ctrl := NewTransferControl()
	called := make(chan struct{}, 1)
	ctrl.OnStop(func() { called <- struct{}{} })
	ctrl.Stop()
	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("OnStop hook not invoked")
	}
	if !ctrl.Stopped() {
		t.Fatal("expected stopped")
	}
	err := ctrl.wait()
	if !errors.Is(err, ErrStopped) {
		t.Fatalf("wait: got %v", err)
	}
}
