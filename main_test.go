package main

import (
	"errors"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"project-template/cmd"
	"project-template/pkg/lifecycle"

	"github.com/bouk/monkey"
)

func TestMainDone(t *testing.T) {
	patchStart := monkey.Patch(cmd.Start, func(ch chan struct{}) { close(ch) })
	patchClose := monkey.Patch(lifecycle.CloseServerListener, func() {})
	patchShutdown := monkey.Patch(lifecycle.ShutdownServer, func() error { return nil })
	defer patchStart.Unpatch()
	defer patchClose.Unpatch()
	defer patchShutdown.Unpatch()

	done := make(chan struct{})
	go func() {
		main()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}

func TestMainSignal(t *testing.T) {
	patchStart := monkey.Patch(cmd.Start, func(ch chan struct{}) {})
	patchClose := monkey.Patch(lifecycle.CloseServerListener, func() {})
	patchShutdown := monkey.Patch(lifecycle.ShutdownServer, func() error { return errors.New("bad") })
	var sigChan chan<- os.Signal
	patchNotify := monkey.Patch(signal.Notify, func(c chan<- os.Signal, sig ...os.Signal) { sigChan = c })
	defer patchStart.Unpatch()
	defer patchClose.Unpatch()
	defer patchShutdown.Unpatch()
	defer patchNotify.Unpatch()

	done := make(chan struct{})
	go func() {
		main()
		close(done)
	}()
	time.Sleep(10 * time.Millisecond)
	sigChan <- syscall.SIGTERM
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout")
	}
}
