package lifecycle

import "testing"

func TestShutdownAndClose(t *testing.T) {
	calledClose := false
	calledShutdown := false
	RegisterClose(func() { calledClose = true })
	RegisterShutdown(func() error { calledShutdown = true; return nil })

	CloseServerListener()
	if !calledClose {
		t.Fatalf("close not called")
	}

	if err := ShutdownServer(); err != nil || !calledShutdown {
		t.Fatalf("shutdown not executed: %v", err)
	}
}

func TestShutdownAndCloseNil(t *testing.T) {
	// reset global vars
	RegisterClose(nil)
	RegisterShutdown(nil)
	CloseServerListener()
	if err := ShutdownServer(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
