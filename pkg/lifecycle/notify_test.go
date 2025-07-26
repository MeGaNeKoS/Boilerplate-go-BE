//go:build !systemd

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
)

func TestNotifyReady(t *testing.T) {
	called := false
	patch := monkey.Patch(supervisor.OnStartUp, func() error { called = true; return nil })
	defer patch.Unpatch()

	NotifyReady()
	if !called {
		t.Fatalf("OnStartUp not called")
	}
}

func TestNotifyReadyStartUpError(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	called := false
	patch := monkey.Patch(supervisor.OnStartUp, func() error {
		called = true
		return errors.New("boom")
	})
	defer patch.Unpatch()

	NotifyReady()

	if !called {
		t.Fatalf("OnStartUp not called")
	}
	out := buf.String()
	if !strings.Contains(out, "startup:") || !strings.Contains(out, "boom") {
		t.Fatalf("startup error log not found: %s", out)
	}
}
