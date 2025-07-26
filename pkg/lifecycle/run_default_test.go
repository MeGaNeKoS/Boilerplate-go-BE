package lifecycle

import (
	"bytes"
	"errors"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/bouk/monkey"
)

func TestRunDone(t *testing.T) {
	patchNotify := monkey.Patch(signal.Notify, func(c chan<- os.Signal, sig ...os.Signal) {})
	patchClose := monkey.Patch(CloseServerListener, func() {})
	patchShutdown := monkey.Patch(ShutdownServer, func() error { return nil })
	defer patchNotify.Unpatch()
	defer patchClose.Unpatch()
	defer patchShutdown.Unpatch()

	Run(func(done chan struct{}) { close(done) })
}

func TestRunSignal(t *testing.T) {
	var sigChan chan<- os.Signal
	patchNotify := monkey.Patch(signal.Notify, func(c chan<- os.Signal, sig ...os.Signal) { sigChan = c })
	calledClose := false
	calledShutdown := false
	patchClose := monkey.Patch(CloseServerListener, func() { calledClose = true })
	patchShutdown := monkey.Patch(ShutdownServer, func() error { calledShutdown = true; return nil })
	defer patchNotify.Unpatch()
	defer patchClose.Unpatch()
	defer patchShutdown.Unpatch()

	go func() {
		time.Sleep(10 * time.Millisecond)
		sigChan <- syscall.SIGTERM
	}()

	Run(func(done chan struct{}) {})

	if !calledClose || !calledShutdown {
		t.Fatalf("shutdown not called")
	}
}

func TestRunSignalShutdownError(t *testing.T) {
	var sigChan chan<- os.Signal
	patchNotify := monkey.Patch(signal.Notify, func(c chan<- os.Signal, sig ...os.Signal) { sigChan = c })
	calledClose := false
	patchClose := monkey.Patch(CloseServerListener, func() { calledClose = true })
	patchShutdown := monkey.Patch(ShutdownServer, func() error { return errors.New("bad") })
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer patchNotify.Unpatch()
	defer patchClose.Unpatch()
	defer patchShutdown.Unpatch()
	defer log.SetOutput(os.Stderr)

	go func() {
		time.Sleep(10 * time.Millisecond)
		sigChan <- syscall.SIGTERM
	}()

	Run(func(done chan struct{}) {})

	if !calledClose {
		t.Fatalf("close not called")
	}
	logOut := buf.String()
	if !strings.Contains(logOut, "Error shutting down server:") || !strings.Contains(logOut, "bad") {
		t.Fatalf("shutdown error log missing: %q", logOut)
	}
}
