package connection

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/bouk/monkey"
)

func TestListenNoSocket(t *testing.T) {
	if err := Listen(""); err == nil {
		t.Fatal("expected error")
	}
}

func TestListenRemoveError(t *testing.T) {
	// Use an invalid path containing a NUL byte to reliably trigger a
	// removal failure on all operating systems. The NUL character causes
	// os.RemoveAll to return an error before Listen attempts to create the
	// socket.
	err := Listen("\x00")
	if err == nil || !strings.Contains(err.Error(), "failed to remove old socket") {
		t.Fatalf("expected remove error, got %v", err)
	}
}

func TestListenListenError(t *testing.T) {
	// directory does not exist so net.Listen will fail
	sock := filepath.Join(t.TempDir(), "nosuch", "sock")
	if err := Listen(sock); err == nil || !strings.Contains(err.Error(), "listen error") {
		t.Fatalf("expected listen error, got %v", err)
	}
}

func TestHandleRunsCommand(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "systemctl")
	argsFile := filepath.Join(dir, "args")
	var scriptContent string
	if runtime.GOOS == "windows" {
		script += ".bat"
		scriptContent = "@echo %* > \"" + argsFile + "\"\n"
	} else {
		scriptContent = "#!/bin/sh\necho \"$@\" >" + argsFile + "\n"
	}
	if err := os.WriteFile(script, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	oldPath := os.Getenv("PATH")
	err := os.Setenv("PATH", dir+string(os.PathListSeparator)+oldPath)
	if err != nil {
		return
	}
	defer func(key, value string) {
		err := os.Setenv(key, value)
		if err != nil {
			return
		}
	}("PATH", oldPath)

	c1, c2 := net.Pipe()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		handle(c1)
	}()
	if _, err := c2.Write([]byte("restart svc.service\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	_ = c2.Close()
	wg.Wait()
	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("args: %v", err)
	}
	got := strings.TrimSpace(string(data))
	if got != "restart svc.service" {
		t.Fatalf("unexpected args %s", got)
	}
}

func TestHandleStopCommand(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "systemctl")
	argsFile := filepath.Join(dir, "args")
	var scriptContent string
	if runtime.GOOS == "windows" {
		script += ".bat"
		scriptContent = "@echo %* > \"" + argsFile + "\"\n"
	} else {
		scriptContent = "#!/bin/sh\necho \"$@\" >" + argsFile + "\n"
	}
	if err := os.WriteFile(script, []byte(scriptContent), 0755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	oldPath := os.Getenv("PATH")
	_ = os.Setenv("PATH", dir+string(os.PathListSeparator)+oldPath)
	defer func() { _ = os.Setenv("PATH", oldPath) }()

	c1, c2 := net.Pipe()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); handle(c1) }()
	if _, err := c2.Write([]byte("stop svc.service\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	_ = c2.Close()
	wg.Wait()
	data, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("args: %v", err)
	}
	got := strings.TrimSpace(string(data))
	if got != "stop svc.service" {
		t.Fatalf("unexpected args %s", got)
	}
}

func TestHandleReadError(t *testing.T) {
	c1, c2 := net.Pipe()
	_ = c2.Close() // trigger read error in handle
	handle(c1)
}

func TestHandleEmptyService(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "systemctl")
	argsFile := filepath.Join(dir, "args")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho \"$@\" >"+argsFile+"\n"), 0755); err != nil {
		t.Fatalf("write script: %v", err)
	}
	oldPath := os.Getenv("PATH")
	_ = os.Setenv("PATH", dir+":"+oldPath)
	defer func() { _ = os.Setenv("PATH", oldPath) }()

	c1, c2 := net.Pipe()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); handle(c1) }()
	if _, err := c2.Write([]byte("\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	_ = c2.Close()
	wg.Wait()
	if _, err := os.Stat(argsFile); !os.IsNotExist(err) {
		t.Fatalf("expected no command execution, got file err=%v", err)
	}
}

type stubConn struct {
	io.Reader
	closeErr error
}

func (s stubConn) Read(b []byte) (int, error)     { return s.Reader.Read(b) }
func (stubConn) Write([]byte) (int, error)        { return 0, nil }
func (s stubConn) Close() error                   { return s.closeErr }
func (stubConn) LocalAddr() net.Addr              { return nil }
func (stubConn) RemoteAddr() net.Addr             { return nil }
func (stubConn) SetDeadline(time.Time) error      { return nil }
func (stubConn) SetReadDeadline(time.Time) error  { return nil }
func (stubConn) SetWriteDeadline(time.Time) error { return nil }

func TestHandleCloseError(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	c := stubConn{Reader: strings.NewReader("\n"), closeErr: errors.New("close")}
	handle(c)
	if !strings.Contains(buf.String(), "Failed to close connection") {
		t.Fatalf("close error log not found: %s", buf.String())
	}
}

func TestHandleRunError(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)
	patchRun := monkey.PatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "Run", func(*exec.Cmd) error {
		return errors.New("run")
	})
	defer patchRun.Unpatch()

	c := stubConn{Reader: strings.NewReader("restart svc.service\n")}
	handle(c)
	if !strings.Contains(buf.String(), "systemctl restart svc.service failed") {
		t.Fatalf("run error log not found: %s", buf.String())
	}
}

type stubListener struct {
	accept func() (net.Conn, error)
	close  func() error
}

func (s *stubListener) Accept() (net.Conn, error) { return s.accept() }
func (s *stubListener) Close() error              { return s.close() }
func (*stubListener) Addr() net.Addr              { return &net.UnixAddr{Net: "unix", Name: ""} }

func TestListenLoopErrors(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)
	monkey.Patch(os.RemoveAll, func(string) error { return nil })
	monkey.Patch(os.Chmod, func(string, os.FileMode) error { return errors.New("chmod") })
	calledClose := false
	l := &stubListener{}
	count := 0
	l.accept = func() (net.Conn, error) {
		if count == 0 {
			count++
			return nil, errors.New("accept")
		}
		panic("stop")
	}
	l.close = func() error { calledClose = true; return errors.New("close") }
	monkey.Patch(net.Listen, func(string, string) (net.Listener, error) { return l, nil })
	defer monkey.UnpatchAll()

	defer func() {
		if r := recover(); r != "stop" {
			t.Fatalf("unexpected panic: %v", r)
		}
		out := buf.String()
		if !strings.Contains(out, "Failed to chmod socket") ||
			!strings.Contains(out, "accept error") ||
			!strings.Contains(out, "Failed to close listener") ||
			!calledClose {
			t.Fatalf("expected logs not found: %s", out)
		}
	}()

	_ = Listen("sock")
}
func TestListenCallsHandle(t *testing.T) {
	monkey.Patch(os.RemoveAll, func(string) error { return nil })
	monkey.Patch(os.Chmod, func(string, os.FileMode) error { return nil })
	done := make(chan struct{})
	monkey.Patch(handle, func(c net.Conn) {
		_ = c.Close()
		close(done)
	})
	l := &stubListener{}
	first := true
	l.accept = func() (net.Conn, error) {
		if first {
			first = false
			c1, c2 := net.Pipe()
			_ = c2.Close()
			return c1, nil
		}
		panic("stop")
	}
	l.close = func() error { return nil }
	monkey.Patch(net.Listen, func(string, string) (net.Listener, error) { return l, nil })
	defer monkey.UnpatchAll()

	defer func() {
		if r := recover(); r != "stop" {
			t.Fatalf("unexpected panic: %v", r)
		}
	}()

	_ = Listen("sock")

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("handle not called")
	}
}

func TestHandleRestartWithTempService(t *testing.T) {
	var calls []string
	monkey.Patch(exec.Command, func(name string, args ...string) *exec.Cmd {
		calls = append(calls, name+" "+strings.Join(args, " "))
		return &exec.Cmd{}
	})
	monkey.PatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "Run", func(*exec.Cmd) error { return nil })
	monkey.PatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "Output", func(*exec.Cmd) ([]byte, error) { return []byte("active\n"), nil })
	defer monkey.UnpatchAll()

	c1, c2 := net.Pipe()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); handle(c1) }()
	if _, err := c2.Write([]byte("restart main.service,tmp.service\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	_ = c2.Close()
	wg.Wait()

	want := []string{
		"systemctl start tmp.service",
		"systemctl is-active tmp.service",
		"systemctl restart main.service",
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls %#v, want %#v", calls, want)
	}
}

func TestHandleRestartTempServiceStartError(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	monkey.Patch(exec.Command, func(name string, args ...string) *exec.Cmd {
		return &exec.Cmd{Path: name, Args: append([]string{name}, args...)}
	})
	monkey.PatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "Run", func(cmd *exec.Cmd) error {
		if len(cmd.Args) > 1 && cmd.Args[1] == "start" {
			return errors.New("boom")
		}
		return nil
	})
	monkey.PatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "Output", func(*exec.Cmd) ([]byte, error) { return []byte("active\n"), nil })
	defer monkey.UnpatchAll()

	c := stubConn{Reader: strings.NewReader("restart main.service,tmp.service\n")}
	handle(c)

	if !strings.Contains(buf.String(), "systemctl start tmp.service failed") {
		t.Fatalf("start error log not found: %s", buf.String())
	}
}

func TestHandleRestartTempServiceTimeout(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	monkey.Patch(exec.Command, func(name string, args ...string) *exec.Cmd {
		return &exec.Cmd{Path: name, Args: append([]string{name}, args...)}
	})
	monkey.PatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "Run", func(*exec.Cmd) error { return nil })
	monkey.PatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "Output", func(*exec.Cmd) ([]byte, error) {
		return []byte("inactive\n"), errors.New("state")
	})
	monkey.Patch(time.Sleep, func(time.Duration) {})
	now := time.Unix(0, 0)
	monkey.Patch(time.Now, func() time.Time {
		now = now.Add(time.Second)
		return now
	})
	defer monkey.UnpatchAll()

	c := stubConn{Reader: strings.NewReader("restart main.service,tmp.service 1\n")}
	handle(c)

	if !strings.Contains(buf.String(), "temporary service not active") {
		t.Fatalf("timeout log not found: %s", buf.String())
	}
}

func TestHandleStopError(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	monkey.Patch(exec.Command, func(name string, args ...string) *exec.Cmd {
		return &exec.Cmd{Path: name, Args: append([]string{name}, args...)}
	})
	monkey.PatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "Run", func(cmd *exec.Cmd) error {
		if len(cmd.Args) > 1 && cmd.Args[1] == "stop" {
			return errors.New("fail")
		}
		return nil
	})
	defer monkey.UnpatchAll()

	c := stubConn{Reader: strings.NewReader("stop svc.service\n")}
	handle(c)

	if !strings.Contains(buf.String(), "systemctl stop svc.service failed") {
		t.Fatalf("stop error log not found: %s", buf.String())
	}
}

func TestHandleRestartNoArgs(t *testing.T) {
	var called bool
	monkey.Patch(exec.Command, func(name string, args ...string) *exec.Cmd {
		called = true
		return &exec.Cmd{}
	})
	defer monkey.UnpatchAll()

	c := stubConn{Reader: strings.NewReader("restart   \n")}
	handle(c)

	if called {
		t.Fatal("command executed for empty restart")
	}
}

func TestHandleRestartEmptyMain(t *testing.T) {
	var called bool
	monkey.Patch(exec.Command, func(name string, args ...string) *exec.Cmd {
		called = true
		return &exec.Cmd{}
	})
	defer monkey.UnpatchAll()

	c := stubConn{Reader: strings.NewReader("restart , ,\n")}
	handle(c)

	if called {
		t.Fatal("command executed for empty main service")
	}
}

func TestHandleEnvTimeoutParseError(t *testing.T) {
	var (
		buf      bytes.Buffer
		outCount int
		now      = time.Unix(0, 0)
	)
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)
	_ = os.Setenv("TEMP_SERVICE_ACTIVE_TIMEOUT", "bad")
	defer func() { _ = os.Unsetenv("TEMP_SERVICE_ACTIVE_TIMEOUT") }()

	monkey.Patch(exec.Command, func(name string, args ...string) *exec.Cmd {
		if len(args) > 0 && args[0] == "is-active" {
			outCount++
		}
		return &exec.Cmd{Path: name, Args: append([]string{name}, args...)}
	})
	monkey.PatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "Run", func(*exec.Cmd) error { return nil })
	monkey.PatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "Output", func(*exec.Cmd) ([]byte, error) {
		return []byte("inactive\n"), errors.New("state")
	})
	monkey.Patch(time.Sleep, func(time.Duration) {})
	monkey.Patch(time.Now, func() time.Time {
		now = now.Add(time.Second)
		return now
	})
	defer monkey.UnpatchAll()

	c := stubConn{Reader: strings.NewReader("restart main.service,tmp.service\n")}
	handle(c)

	if !strings.Contains(buf.String(), "temporary service not active") {
		t.Fatalf("timeout log not found: %s", buf.String())
	}
	if outCount != 31 {
		t.Fatalf("unexpected is-active calls %d", outCount)
	}
}

func TestHandleEnvTimeoutValue(t *testing.T) {
	var (
		buf      bytes.Buffer
		outCount int
		now      = time.Unix(0, 0)
	)
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)
	_ = os.Setenv("TEMP_SERVICE_ACTIVE_TIMEOUT", "2")
	defer func() { _ = os.Unsetenv("TEMP_SERVICE_ACTIVE_TIMEOUT") }()

	monkey.Patch(exec.Command, func(name string, args ...string) *exec.Cmd {
		if len(args) > 0 && args[0] == "is-active" {
			outCount++
		}
		return &exec.Cmd{Path: name, Args: append([]string{name}, args...)}
	})
	monkey.PatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "Run", func(*exec.Cmd) error { return nil })
	monkey.PatchInstanceMethod(reflect.TypeOf(&exec.Cmd{}), "Output", func(*exec.Cmd) ([]byte, error) {
		return []byte("inactive\n"), errors.New("state")
	})
	monkey.Patch(time.Sleep, func(time.Duration) {})
	monkey.Patch(time.Now, func() time.Time {
		now = now.Add(time.Second)
		return now
	})
	defer monkey.UnpatchAll()

	c := stubConn{Reader: strings.NewReader("restart main.service,tmp.service\n")}
	handle(c)

	if !strings.Contains(buf.String(), "temporary service not active") {
		t.Fatalf("timeout log not found: %s", buf.String())
	}
	if outCount != 3 {
		t.Fatalf("unexpected is-active calls %d", outCount)
	}
}

func TestHandleStopNoArgs(t *testing.T) {
	var called bool
	monkey.Patch(exec.Command, func(name string, args ...string) *exec.Cmd {
		called = true
		return &exec.Cmd{}
	})
	defer monkey.UnpatchAll()

	c := stubConn{Reader: strings.NewReader("stop   \n")}
	handle(c)

	if called {
		t.Fatal("command executed for empty stop")
	}
}

func TestHandleOnlySpaces(t *testing.T) {
	var called bool
	monkey.Patch(exec.Command, func(name string, args ...string) *exec.Cmd {
		called = true
		return &exec.Cmd{}
	})
	defer monkey.UnpatchAll()

	c := stubConn{Reader: strings.NewReader("\t\n")}
	handle(c)

	if called {
		t.Fatal("command executed for blank message")
	}
}
