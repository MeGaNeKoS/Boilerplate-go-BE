package logger

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"project-template/infrastructure/config"

	"github.com/bouk/monkey"
)

func TestParseLogLevel(t *testing.T) {
	cases := map[string]Level{
		"DEBUG":   DEBUG,
		"INFO":    INFO,
		"WARN":    WARN,
		"ERROR":   ERROR,
		"FATAL":   FATAL,
		"unknown": INFO,
	}
	for input, want := range cases {
		if got := ParseLogLevel(input); got != want {
			t.Errorf("ParseLogLevel(%q)=%v, want %v", input, got, want)
		}
	}
}

func TestLogger(t *testing.T) {
	sharedCore = nil
	dir := t.TempDir()
	cfg := config.LogConfig{Path: dir, FileName: "app.log", PerLevelFiles: map[string]string{"ERROR": "err.log"}, VerboseLevel: "DEBUG"}
	lgrIface, err := NewLogger(cfg, "p", "c")
	if err != nil {
		t.Fatalf("NewLogger error: %v", err)
	}
	lgr := lgrIface.(*loggerImpl)
	if lgr.ParentID() != "p" || lgr.ChildID() != "c" {
		t.Fatalf("id mismatch")
	}
	lgr.DebugF("d")
	lgr.InfoF("hello")
	lgr.WarnF("w")
	lgr.ErrorF("e")
	lgr.FatalF("f")
	data, err := os.ReadFile(filepath.Join(dir, "app.log"))
	if err != nil || !strings.Contains(string(data), "INFO") || !strings.Contains(string(data), "DEBUG") {
		t.Fatalf("log not written")
	}
	CloseLogFile()
	if _, err := lgr.core.logFile.Write([]byte("x")); err == nil {
		t.Fatalf("expected write error after close")
	}
}

func TestInitLoggerCoreErrors(t *testing.T) {
	t.Run("mkdir failure", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "file")
		if err := os.WriteFile(file, []byte("x"), 0600); err != nil {
			t.Fatal(err)
		}
		_, err := initLoggerCore(config.LogConfig{Path: file, FileName: "app.log"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("open default file failure", func(t *testing.T) {
		dir := t.TempDir()
		_, err := initLoggerCore(config.LogConfig{Path: dir, FileName: ""})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("per level file failure", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, "err.log"), 0755); err != nil {
			t.Fatal(err)
		}
		cfg := config.LogConfig{Path: dir, FileName: "app.log", PerLevelFiles: map[string]string{"ERROR": "err.log"}}
		_, err := initLoggerCore(cfg)
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("per level skip", func(t *testing.T) {
		dir := t.TempDir()
		cfg := config.LogConfig{Path: dir, FileName: "app.log", PerLevelFiles: map[string]string{"DEBUG": ""}}
		core, err := initLoggerCore(cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := core.levelFile[DEBUG]; ok {
			t.Fatalf("expected debug level file to be skipped")
		}
		core.logFile.Close()
		for _, w := range core.watchers {
			w.Close()
		}
	})
}

func TestLogBranches(t *testing.T) {
	var b bytes.Buffer
	core := &loggerCore{
		level:       INFO,
		debugLogger: log.New(&b, "", 0),
		infoLogger:  log.New(&b, "", 0),
		warnLogger:  log.New(&b, "", 0),
		errorLogger: log.New(&b, "", 0),
		fatalLogger: log.New(&b, "", 0),
	}
	l := &loggerImpl{core: core, parentID: "p", childID: "c"}

	l.log(DEBUG, "DEBUG", "skip")
	if b.Len() != 0 {
		t.Fatalf("expected no log, got %s", b.String())
	}

	core.level = DEBUG
	core.toStdout = true
	l.depth = 999999
	b.Reset()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	l.log(Level(99), "OTHER", "msg")
	w.Close()
	os.Stdout = oldStdout
	out, _ := io.ReadAll(r)
	r.Close()

	logOut := b.String()
	if !strings.Contains(logOut, "unknown") || !strings.Contains(string(out), "unknown") {
		t.Fatalf("expected unknown path, got buffer=%q stdout=%q", logOut, string(out))
	}
	if !strings.Contains(logOut, "OTHER") || !strings.Contains(logOut, "msg") {
		t.Fatalf("log missing parts: %q", logOut)
	}
}

func TestNewLoggerError(t *testing.T) {
	sharedCore = nil
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := NewLogger(config.LogConfig{Path: file, FileName: "app.log"}, "p", "c")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDepthIsolation(t *testing.T) {
	var b bytes.Buffer
	core := &loggerCore{level: DEBUG, debugLogger: log.New(&b, "", 0), infoLogger: log.New(&b, "", 0), warnLogger: log.New(&b, "", 0), errorLogger: log.New(&b, "", 0), fatalLogger: log.New(&b, "", 0)}
	l := loggerImpl{core: core}
	l.InfoF("one")
	if l.depth != 0 {
		t.Fatalf("depth mutated: %d", l.depth)
	}
	if !strings.Contains(b.String(), "one") {
		t.Fatalf("log missing")
	}
}

func TestCloseFunctions(t *testing.T) {
	// should handle nil shared core
	sharedCore = nil
	CloseLogFile()

	dir := t.TempDir()
	cfg := config.LogConfig{Path: dir, FileName: "app.log"}
	lgr, err := NewLogger(cfg, "p", "c")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	CloseLogFile()
	if sharedCore != nil {
		t.Fatalf("sharedCore not cleared")
	}
	impl := lgr.(*loggerImpl)
	if _, err := impl.core.logFile.Write([]byte("x")); err == nil {
		t.Fatalf("expected error after close")
	}

}

func TestInitLoggerCoreFailureClosesFiles(t *testing.T) {
	sharedCore = nil
	dir := t.TempDir()

	var defaultF, levelF *os.File
	cnt := 0
	origOpen := os.OpenFile
	patch := monkey.Patch(os.OpenFile, func(name string, flag int, perm os.FileMode) (*os.File, error) {
		cnt++
		if cnt == 1 {
			var err error
			defaultF, err = origOpen(name, flag, perm)
			return defaultF, err
		}
		if cnt == 2 {
			var err error
			levelF, err = origOpen(name, flag, perm)
			return levelF, err
		}
		return nil, fmt.Errorf("bad open")
	})
	defer patch.Unpatch()

	cfg := config.LogConfig{Path: dir, FileName: "app.log", PerLevelFiles: map[string]string{"INFO": "info.log", "ERROR": "err.log"}}
	_, err := initLoggerCore(cfg)
	if err == nil {
		t.Fatalf("expected error")
	}
	if _, err := defaultF.Write([]byte("x")); err == nil {
		t.Fatalf("default file not closed")
	}
	if _, err := levelF.Write([]byte("x")); err == nil {
		t.Fatalf("level file not closed")
	}
}

func TestInitLoggerCoreSecondLevelFailure(t *testing.T) {
	sharedCore = nil
	dir := t.TempDir()

	var defaultF, levelF *os.File
	appPath := filepath.Join(dir, "app.log")
	infoPath := filepath.Join(dir, "info.log")
	errPath := filepath.Join(dir, "err.log")
	var guard *monkey.PatchGuard
	guard = monkey.Patch(os.OpenFile, func(name string, flag int, perm os.FileMode) (*os.File, error) {
		if name == errPath {
			return nil, fmt.Errorf("bad open")
		}
		guard.Unpatch()
		defer guard.Restore()
		f, err := os.OpenFile(name, flag, perm)
		if name == appPath {
			defaultF = f
		} else if name == infoPath {
			levelF = f
		}
		return f, err
	})
	defer guard.Unpatch()

	cfg := config.LogConfig{Path: dir, FileName: "app.log", PerLevelFiles: map[string]string{"INFO": "info.log", "ERROR": "err.log"}}
	_, err := initLoggerCore(cfg)
	if err == nil {
		t.Fatalf("expected error")
	}
	if _, err := defaultF.Write([]byte("x")); err == nil {
		t.Fatalf("default file not closed")
	}
	if _, err := levelF.Write([]byte("x")); err == nil {
		t.Fatalf("level file not closed")
	}
}

// TestInitLoggerCoreClosesMultipleFiles explicitly verifies that all opened
// log files are closed when a later file fails to open. This covers the branch
// that iterates over levelFiles and closes each handle before returning an
// error.
func TestInitLoggerCoreClosesMultipleFiles(t *testing.T) {
	sharedCore = nil
	dir := t.TempDir()

	var defaultF, levelF *os.File
	cnt := 0
	origOpen := os.OpenFile
	patch := monkey.Patch(os.OpenFile, func(name string, flag int, perm os.FileMode) (*os.File, error) {
		cnt++
		if cnt == 1 {
			var err error
			defaultF, err = origOpen(name, flag, perm)
			return defaultF, err
		}
		if cnt == 2 {
			var err error
			levelF, err = origOpen(name, flag, perm)
			return levelF, err
		}
		return nil, fmt.Errorf("bad open")
	})
	defer patch.Unpatch()

	cfg := config.LogConfig{Path: dir, FileName: "app.log", PerLevelFiles: map[string]string{"INFO": "info.log", "ERROR": "err.log"}}
	_, err := initLoggerCore(cfg)
	if err == nil {
		t.Fatalf("expected error")
	}
	if _, err := defaultF.Write([]byte("x")); err == nil {
		t.Fatalf("default file not closed")
	}
	if _, err := levelF.Write([]byte("x")); err == nil {
		t.Fatalf("level file not closed")
	}
}

func TestLogFileReopen(t *testing.T) {
	sharedCore = nil
	dir := t.TempDir()
	cfg := config.LogConfig{Path: dir, FileName: "app.log", PerLevelFiles: map[string]string{"ERROR": "err.log"}, VerboseLevel: "DEBUG"}
	lgrIface, err := NewLogger(cfg, "p", "c")
	defer CloseLogFile()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	lgr := lgrIface.(*loggerImpl)
	lgr.InfoF("one")
	lgr.ErrorF("bad1")

	os.Remove(filepath.Join(dir, "app.log"))
	os.Remove(filepath.Join(dir, "err.log"))
	time.Sleep(200 * time.Millisecond)

	lgr.InfoF("two")
	lgr.ErrorF("bad2")
	time.Sleep(200 * time.Millisecond)

	data, err := os.ReadFile(filepath.Join(dir, "app.log"))
	if err != nil || !strings.Contains(string(data), "two") {
		t.Fatalf("log file not recreated")
	}
	edata, err := os.ReadFile(filepath.Join(dir, "err.log"))
	if err != nil || !strings.Contains(string(edata), "bad2") {
		t.Fatalf("level log file not recreated")
	}
}
func TestInitLoggerCoreClosesAllLevelFiles(t *testing.T) {
	sharedCore = nil
	dir := t.TempDir()

	var defaultF, firstLF, secondLF *os.File
	cnt := 0
	origOpen := os.OpenFile
	patch := monkey.Patch(os.OpenFile, func(name string, flag int, perm os.FileMode) (*os.File, error) {
		cnt++
		if cnt == 1 {
			var err error
			defaultF, err = origOpen(name, flag, perm)
			return defaultF, err
		}
		if cnt == 2 {
			var err error
			firstLF, err = origOpen(name, flag, perm)
			return firstLF, err
		}
		if cnt == 3 {
			var err error
			secondLF, err = origOpen(name, flag, perm)
			return secondLF, err
		}
		return nil, fmt.Errorf("bad open")
	})
	defer patch.Unpatch()

	cfg := config.LogConfig{
		Path:          dir,
		FileName:      "app.log",
		PerLevelFiles: map[string]string{"INFO": "info.log", "WARN": "warn.log", "ERROR": "err.log"},
	}
	_, err := initLoggerCore(cfg)
	if err == nil {
		t.Fatalf("expected error")
	}
	if _, err := defaultF.Write([]byte("x")); err == nil {
		t.Fatalf("default file not closed")
	}
	if _, err := firstLF.Write([]byte("x")); err == nil {
		t.Fatalf("first level file not closed")
	}
	if _, err := secondLF.Write([]byte("x")); err == nil {
		t.Fatalf("second level file not closed")
	}
}
