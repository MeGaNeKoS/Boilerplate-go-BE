//go:build systemd

package supervisor

import (
	"bytes"
	"errors"
	"net"
	"os"
	"testing"
	"time"

	"github.com/bouk/monkey"
)

type recordConn struct {
	bytes.Buffer
	closed bool
}

func (r *recordConn) Read(b []byte) (int, error)      { return 0, nil }
func (r *recordConn) Write(b []byte) (int, error)     { return r.Buffer.Write(b) }
func (r *recordConn) Close() error                    { r.closed = true; return nil }
func (recordConn) LocalAddr() net.Addr                { return nil }
func (recordConn) RemoteAddr() net.Addr               { return nil }
func (recordConn) SetDeadline(t time.Time) error      { return nil }
func (recordConn) SetReadDeadline(t time.Time) error  { return nil }
func (recordConn) SetWriteDeadline(t time.Time) error { return nil }

func TestRequestRestartSuccess(t *testing.T) {
	var rc recordConn
	patch := monkey.Patch(net.Dial, func(network, addr string) (net.Conn, error) {
		if network != "unix" || addr != "/tmp/sock" {
			t.Fatalf("unexpected dial %s %s", network, addr)
		}
		return &rc, nil
	})
	defer patch.Unpatch()

	os.Setenv("SUPERVISOR_SOCKET", "/tmp/sock")
	os.Setenv("SELF_SERVICE_UNIT", "main.service")
	os.Unsetenv("TEMP_SERVICE")
	os.Unsetenv("TEMP_SERVICE_UNIT")
	defer os.Unsetenv("SUPERVISOR_SOCKET")
	defer os.Unsetenv("SELF_SERVICE_UNIT")

	if err := RequestRestart(); err != nil {
		t.Fatalf("RequestRestart error: %v", err)
	}
	if rc.String() != "restart main.service\n" {
		t.Fatalf("unexpected message: %q", rc.String())
	}
	if !rc.closed {
		t.Fatal("connection not closed")
	}
}

func TestRequestRestartMissingSocket(t *testing.T) {
	os.Unsetenv("SUPERVISOR_SOCKET")
	os.Setenv("SELF_SERVICE_UNIT", "svc.service")
	defer os.Unsetenv("SELF_SERVICE_UNIT")
	if err := RequestRestart(); err == nil {
		t.Fatal("expected error")
	}
}

func TestRequestRestartWithTempUnit(t *testing.T) {
	var rc recordConn
	patch := monkey.Patch(net.Dial, func(network, addr string) (net.Conn, error) {
		if network != "unix" || addr != "/tmp/sock" {
			t.Fatalf("unexpected dial %s %s", network, addr)
		}
		return &rc, nil
	})
	defer patch.Unpatch()

	os.Setenv("SUPERVISOR_SOCKET", "/tmp/sock")
	os.Setenv("SELF_SERVICE_UNIT", "main.service")
	os.Setenv("TEMP_SERVICE_UNIT", "tmp.service")
	os.Unsetenv("TEMP_SERVICE")
	defer os.Unsetenv("SUPERVISOR_SOCKET")
	defer os.Unsetenv("SELF_SERVICE_UNIT")
	defer os.Unsetenv("TEMP_SERVICE_UNIT")

	if err := RequestRestart(); err != nil {
		t.Fatalf("RequestRestart error: %v", err)
	}
	if rc.String() != "restart main.service,tmp.service\n" {
		t.Fatalf("unexpected message: %q", rc.String())
	}
	if !rc.closed {
		t.Fatal("connection not closed")
	}
}

func TestRequestRestartTempServiceSkip(t *testing.T) {
	var called bool
	patch := monkey.Patch(net.Dial, func(network, addr string) (net.Conn, error) {
		called = true
		return nil, nil
	})
	defer patch.Unpatch()

	os.Setenv("TEMP_SERVICE", "true")
	os.Setenv("SUPERVISOR_SOCKET", "/tmp/sock")
	os.Setenv("SELF_SERVICE_UNIT", "main.service")
	os.Unsetenv("TEMP_SERVICE_UNIT")
	defer os.Unsetenv("TEMP_SERVICE")
	defer os.Unsetenv("SUPERVISOR_SOCKET")
	defer os.Unsetenv("SELF_SERVICE_UNIT")

	if err := RequestRestart(); err != nil {
		t.Fatalf("RequestRestart error: %v", err)
	}
	if called {
		t.Fatal("net.Dial should not be called")
	}
}

func TestOnStartUpSuccess(t *testing.T) {
	var rc recordConn
	patch := monkey.Patch(net.Dial, func(network, addr string) (net.Conn, error) {
		if network != "unix" || addr != "/tmp/sock" {
			t.Fatalf("unexpected dial %s %s", network, addr)
		}
		return &rc, nil
	})
	defer patch.Unpatch()

	os.Setenv("SUPERVISOR_SOCKET", "/tmp/sock")
	os.Setenv("TEMP_SERVICE_UNIT", "tmp.service")
	os.Unsetenv("TEMP_SERVICE")
	defer os.Unsetenv("SUPERVISOR_SOCKET")
	defer os.Unsetenv("TEMP_SERVICE_UNIT")

	if err := OnStartUp(); err != nil {
		t.Fatalf("OnStartUp error: %v", err)
	}
	if rc.String() != "stop tmp.service\n" {
		t.Fatalf("unexpected message: %q", rc.String())
	}
	if !rc.closed {
		t.Fatal("connection not closed")
	}
}

func TestOnStartUpTempServiceSkip(t *testing.T) {
	var called bool
	patch := monkey.Patch(net.Dial, func(network, addr string) (net.Conn, error) {
		called = true
		return nil, nil
	})
	defer patch.Unpatch()

	os.Setenv("TEMP_SERVICE", "true")
	os.Setenv("TEMP_SERVICE_UNIT", "tmp.service")
	defer os.Unsetenv("TEMP_SERVICE")
	defer os.Unsetenv("TEMP_SERVICE_UNIT")

	if err := OnStartUp(); err != nil {
		t.Fatalf("OnStartUp error: %v", err)
	}
	if called {
		t.Fatal("net.Dial should not be called")
	}
}

func TestRequestRestartDialError(t *testing.T) {
	patch := monkey.Patch(net.Dial, func(string, string) (net.Conn, error) {
		return nil, errors.New("dial")
	})
	defer patch.Unpatch()

	os.Setenv("SUPERVISOR_SOCKET", "/tmp/sock")
	os.Setenv("SELF_SERVICE_UNIT", "main.service")
	os.Unsetenv("TEMP_SERVICE")
	defer os.Unsetenv("SUPERVISOR_SOCKET")
	defer os.Unsetenv("SELF_SERVICE_UNIT")

	if err := RequestRestart(); err == nil {
		t.Fatalf("expected error")
	}
}

func TestOnStartUpMissingSocket(t *testing.T) {
	os.Unsetenv("SUPERVISOR_SOCKET")
	os.Setenv("TEMP_SERVICE_UNIT", "tmp.service")
	os.Unsetenv("TEMP_SERVICE")
	defer os.Unsetenv("TEMP_SERVICE_UNIT")

	if err := OnStartUp(); err == nil {
		t.Fatal("expected error")
	}
}

func TestOnStartUpDialError(t *testing.T) {
	patch := monkey.Patch(net.Dial, func(string, string) (net.Conn, error) {
		return nil, errors.New("dial")
	})
	defer patch.Unpatch()

	os.Setenv("SUPERVISOR_SOCKET", "/tmp/sock")
	os.Setenv("TEMP_SERVICE_UNIT", "tmp.service")
	os.Unsetenv("TEMP_SERVICE")
	defer os.Unsetenv("SUPERVISOR_SOCKET")
	defer os.Unsetenv("TEMP_SERVICE_UNIT")

	if err := OnStartUp(); err == nil {
		t.Fatal("expected error")
	}
}

type writeErrConn struct {
	recordConn
	err error
}

func (w *writeErrConn) Write(b []byte) (int, error) { return 0, w.err }

func TestOnStartUpWriteError(t *testing.T) {
	wc := &writeErrConn{err: errors.New("write")}
	patch := monkey.Patch(net.Dial, func(string, string) (net.Conn, error) { return wc, nil })
	defer patch.Unpatch()

	os.Setenv("SUPERVISOR_SOCKET", "/tmp/sock")
	os.Setenv("TEMP_SERVICE_UNIT", "tmp.service")
	os.Unsetenv("TEMP_SERVICE")
	defer os.Unsetenv("SUPERVISOR_SOCKET")
	defer os.Unsetenv("TEMP_SERVICE_UNIT")

	if err := OnStartUp(); err == nil {
		t.Fatal("expected error")
	}
	if !wc.closed {
		t.Fatal("connection not closed")
	}
}
