//go:build systemd

package lifecycle

import (
	"bytes"
	"errors"
	"log"
	"os"
	"strings"
	"testing"

	"project-template/infrastructure/supervisor"

	"github.com/bouk/monkey"
	"github.com/coreos/go-systemd/v22/daemon"
)

func TestNotifyReadySystemd(t *testing.T) {
	called := false
	patchNotify := monkey.Patch(daemon.SdNotify, func(un bool, msg string) (bool, error) {
		if msg != daemon.SdNotifyReady {
			t.Fatalf("unexpected notify message: %s", msg)
		}
		called = true
		return true, nil
	})
	defer patchNotify.Unpatch()

	startupCalled := false
	patchStartup := monkey.Patch(supervisor.OnStartUp, func() error { startupCalled = true; return nil })
	defer patchStartup.Unpatch()

	os.Setenv("NOTIFY_SOCKET", "/tmp/socket")
	defer os.Unsetenv("NOTIFY_SOCKET")

	NotifyReady()
	if !called {
		t.Fatalf("SdNotify not called")
	}
	if !startupCalled {
		t.Fatalf("OnStartUp not called")
	}
}

func TestNotifyReadySystemdStartupError(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	notifyCalled := false
	patchNotify := monkey.Patch(daemon.SdNotify, func(un bool, msg string) (bool, error) {
		if msg != daemon.SdNotifyReady {
			t.Fatalf("unexpected notify message: %s", msg)
		}
		notifyCalled = true
		return true, nil
	})
	defer patchNotify.Unpatch()

	startupCalled := false
	patchStartup := monkey.Patch(supervisor.OnStartUp, func() error {
		startupCalled = true
		return errors.New("boom")
	})
	defer patchStartup.Unpatch()

	os.Setenv("NOTIFY_SOCKET", "/tmp/socket")
	defer os.Unsetenv("NOTIFY_SOCKET")

	NotifyReady()

	if !notifyCalled {
		t.Fatalf("SdNotify not called")
	}
	if !startupCalled {
		t.Fatalf("OnStartUp not called")
	}
	out := buf.String()
	if !strings.Contains(out, "startup:") || !strings.Contains(out, "boom") {
		t.Fatalf("startup error log not found: %s", out)
	}
}
