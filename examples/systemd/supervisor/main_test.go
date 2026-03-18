package main

import (
	"errors"
	"log"
	"os"
	"testing"

	"github.com/bouk/monkey"

	"project-template/examples/systemd/supervisor/connection"
)

type stubListen struct {
	called bool
	arg    string
	err    error
}

func (s *stubListen) call(socket string) error {
	s.called = true
	s.arg = socket
	return s.err
}

func TestRunSuccess(t *testing.T) {
	st := &stubListen{}
	patchListen := monkey.Patch(connection.Listen, st.call)
	defer patchListen.Unpatch()
	_ = os.Setenv("SUPERVISOR_SOCKET", "sock")
	defer func() { _ = os.Unsetenv("SUPERVISOR_SOCKET") }()
	if err := run(); err != nil {
		t.Fatalf("run error: %v", err)
	}
	if !st.called || st.arg != "sock" {
		t.Fatalf("listen not called correctly: %#v", st)
	}
}

func TestRunError(t *testing.T) {
	st := &stubListen{err: errors.New("fail")}
	patchListen := monkey.Patch(connection.Listen, st.call)
	defer patchListen.Unpatch()
	_ = os.Setenv("SUPERVISOR_SOCKET", "bad")
	defer func() { _ = os.Unsetenv("SUPERVISOR_SOCKET") }()
	if err := run(); err == nil {
		t.Fatalf("expected error")
	}
}

func TestMainError(t *testing.T) {
	patchRun := monkey.Patch(run, func() error { return errors.New("boom") })
	patchFatal := monkey.Patch(log.Fatal, func(v ...interface{}) { panic("fatal") })
	defer patchRun.Unpatch()
	defer patchFatal.Unpatch()
	defer func() {
		if r := recover(); r != "fatal" {
			t.Fatalf("expected panic 'fatal', got %v", r)
		}
	}()
	main()
}
