package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/bouk/monkey"
	"github.com/fsnotify/fsnotify"
)

func newCoreWithFiles(t *testing.T, dir string) (*loggerCore, *os.File, *os.File) {
	t.Helper()
	logPath := filepath.Join(dir, "app.log")
	errPath := filepath.Join(dir, "err.log")

	lf, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("open default: %v", err)
	}
	ef, err := os.OpenFile(errPath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("open level: %v", err)
	}

	core := &loggerCore{
		logFile:       lf,
		logFilePath:   logPath,
		levelFile:     map[Level]*os.File{ERROR: ef},
		levelFilePath: map[Level]string{ERROR: errPath},
		debugLogger:   log.New(io.Discard, "", 0),
		infoLogger:    log.New(io.Discard, "", 0),
		warnLogger:    log.New(io.Discard, "", 0),
		errorLogger:   log.New(io.Discard, "", 0),
		fatalLogger:   log.New(io.Discard, "", 0),
	}
	return core, lf, ef
}

func TestReopenFileRetriesDefault(t *testing.T) {
	dir := t.TempDir()
	core, oldF, _ := newCoreWithFiles(t, dir)

	cnt := 0
	var patch *monkey.PatchGuard
	patch = monkey.Patch(os.OpenFile, func(name string, flag int, perm os.FileMode) (*os.File, error) {
		cnt++
		if cnt == 1 {
			return nil, fmt.Errorf("bad open")
		}
		patch.Unpatch()
		defer patch.Restore()
		return os.OpenFile(name, flag, perm)
	})
	defer patch.Unpatch()

	start := time.Now()
	core.reopenFile(core.logFilePath)
	if cnt != 2 {
		t.Fatalf("expected 2 attempts, got %d", cnt)
	}
	if core.logFile == oldF {
		t.Fatalf("log file not replaced")
	}
	if _, err := oldF.Write([]byte("x")); err == nil {
		t.Fatalf("old file not closed")
	}
	if time.Since(start) < 100*time.Millisecond {
		t.Fatalf("reopen did not wait on failure")
	}
	if err := core.logFile.Close(); err != nil {
		t.Fatalf("close new file: %v", err)
	}
	if err := core.levelFile[ERROR].Close(); err != nil {
		t.Fatalf("close level file: %v", err)
	}
}

func TestReopenFileLevel(t *testing.T) {
	dir := t.TempDir()
	core, _, oldLF := newCoreWithFiles(t, dir)

	var patch *monkey.PatchGuard
	patch = monkey.Patch(os.OpenFile, func(name string, flag int, perm os.FileMode) (*os.File, error) {
		patch.Unpatch()
		defer patch.Restore()
		return os.OpenFile(name, flag, perm)
	})
	defer patch.Unpatch()

	core.reopenFile(core.levelFilePath[ERROR])

	nf := core.levelFile[ERROR]
	if nf == oldLF {
		t.Fatalf("level file not replaced")
	}
	if _, err := oldLF.Write([]byte("x")); err == nil {
		t.Fatalf("old level file not closed")
	}
	if err := core.levelFile[ERROR].Close(); err != nil {
		t.Fatalf("close new level file: %v", err)
	}
	if err := core.logFile.Close(); err != nil {
		t.Fatalf("close default log file: %v", err)
	}
}

func TestWatchLoop(t *testing.T) {
	dir := t.TempDir()
	core, oldF, _ := newCoreWithFiles(t, dir)

	w := &fsnotify.Watcher{Events: make(chan fsnotify.Event), Errors: make(chan error)}
	go core.watchLoop(w, core.logFilePath)

	w.Events <- fsnotify.Event{Name: core.logFilePath, Op: fsnotify.Remove}
	time.Sleep(100 * time.Millisecond)

	if core.logFile == oldF {
		t.Fatalf("log file not reopened")
	}
	if _, err := oldF.Write([]byte("x")); err == nil {
		t.Fatalf("old file not closed")
	}

	close(w.Events)
	close(w.Errors)

	core.logFile.Close()
	core.levelFile[ERROR].Close()
}

func TestWatchLoopErrorsClose(t *testing.T) {
	dir := t.TempDir()
	core, _, _ := newCoreWithFiles(t, dir)

	w := &fsnotify.Watcher{Events: make(chan fsnotify.Event), Errors: make(chan error)}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		core.watchLoop(w, core.logFilePath)
	}()

	close(w.Errors)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("watchLoop did not exit on errors channel close")
	}

	close(w.Events)
	core.logFile.Close()
	core.levelFile[ERROR].Close()
}

func TestStartWatchersNewWatcherError(t *testing.T) {
	dir := t.TempDir()
	core, _, _ := newCoreWithFiles(t, dir)

	monkey.Patch(fsnotify.NewBufferedWatcher, func(uint) (*fsnotify.Watcher, error) {
		return nil, fmt.Errorf("new watcher")
	})
	defer monkey.UnpatchAll()

	core.startWatchers()
	if len(core.watchers) != 0 {
		t.Fatalf("expected no watchers, got %d", len(core.watchers))
	}

	core.logFile.Close()
	core.levelFile[ERROR].Close()
}

func TestStartWatchersAddError(t *testing.T) {
	dir := t.TempDir()
	core, _, _ := newCoreWithFiles(t, dir)

	monkey.Patch(fsnotify.NewBufferedWatcher, func(uint) (*fsnotify.Watcher, error) {
		return &fsnotify.Watcher{Events: make(chan fsnotify.Event), Errors: make(chan error)}, nil
	})
	var closeCnt int
	m, _ := reflect.TypeOf(&fsnotify.Watcher{}).MethodByName("AddWith")
	monkey.PatchInstanceMethod(reflect.TypeOf(&fsnotify.Watcher{}), "AddWith", reflect.MakeFunc(m.Type, func(args []reflect.Value) []reflect.Value {
		return []reflect.Value{reflect.ValueOf(fmt.Errorf("add error"))}
	}).Interface())
	monkey.PatchInstanceMethod(reflect.TypeOf(&fsnotify.Watcher{}), "Close", func(*fsnotify.Watcher) error {
		closeCnt++
		return nil
	})
	defer monkey.UnpatchAll()

	core.startWatchers()
	if closeCnt == 0 {
		t.Fatalf("expected watcher to be closed on add failure")
	}
	if len(core.watchers) != 0 {
		t.Fatalf("expected no watchers, got %d", len(core.watchers))
	}

	core.logFile.Close()
	core.levelFile[ERROR].Close()
}
