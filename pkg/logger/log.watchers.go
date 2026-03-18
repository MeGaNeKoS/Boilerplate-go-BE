package logger

import (
	"io"
	stdlog "log"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

const (
	maxReopenRetries  = 50
	initialRetryDelay = 100 * time.Millisecond
	maxRetryDelay     = 5 * time.Second
)

func (c *loggerCore) startWatchers() {
	paths := []string{c.logFilePath}
	for _, p := range c.levelFilePath {
		paths = append(paths, p)
	}
	for _, p := range paths {
		w, err := fsnotify.NewWatcher()
		if err != nil {
			stdlog.Printf("logger: failed to create watcher for %s: %v", p, err)
			continue
		}
		if err := w.Add(filepath.Dir(p)); err != nil {
			stdlog.Printf("logger: failed to watch directory for %s: %v", p, err)
			if cerr := w.Close(); cerr != nil {
				stdlog.Printf("logger: failed to close watcher: %v", cerr)
			}
			continue
		}
		c.watchers = append(c.watchers, w)
		go c.watchLoop(w, p)
	}
}

func (c *loggerCore) watchLoop(w *fsnotify.Watcher, path string) {
	for {
		select {
		case ev, ok := <-w.Events:
			if !ok {
				return
			}
			if ev.Op&(fsnotify.Remove|fsnotify.Rename) != 0 && filepath.Clean(ev.Name) == filepath.Clean(path) {
				c.reopenFile(path)
			}
		case _, ok := <-w.Errors:
			if !ok {
				return
			}
		}
	}
}

func (c *loggerCore) reopenFile(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delay := initialRetryDelay
	for attempt := 0; attempt < maxReopenRetries; attempt++ {
		f, err := openLogFile(path)
		if err != nil {
			if attempt < maxReopenRetries-1 {
				time.Sleep(delay)
				delay *= 2
				if delay > maxRetryDelay {
					delay = maxRetryDelay
				}
			}
			continue
		}
		cleanPath := filepath.Clean(path)
		if cleanPath == filepath.Clean(c.logFilePath) {
			if c.logFile != nil {
				if cerr := c.logFile.Close(); cerr != nil {
					stdlog.Printf("logger: failed to close old log file: %v", cerr)
				}
			}
			c.logFile = f
		} else {
			for lvl, p := range c.levelFilePath {
				if filepath.Clean(p) == cleanPath {
					if lf := c.levelFile[lvl]; lf != nil && lf != c.logFile {
						if cerr := lf.Close(); cerr != nil {
							stdlog.Printf("logger: failed to close old level file: %v", cerr)
						}
					}
					c.levelFile[lvl] = f
					break
				}
			}
		}
		c.rebuildLoggers()
		return
	}
	stdlog.Printf("logger: failed to reopen %s after %d retries", path, maxReopenRetries)
}

func openLogFile(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
}

func (c *loggerCore) rebuildLoggers() {
	levelWriters := make(map[Level]io.Writer)
	for lvl, lf := range c.levelFile {
		if lf != nil {
			levelWriters[lvl] = io.MultiWriter(c.logFile, lf)
		}
	}
	fallback := func(lvl Level) io.Writer {
		if w, ok := levelWriters[lvl]; ok {
			return w
		}
		return c.logFile
	}
	c.debugLogger.SetOutput(fallback(DEBUG))
	c.infoLogger.SetOutput(fallback(INFO))
	c.warnLogger.SetOutput(fallback(WARN))
	c.errorLogger.SetOutput(fallback(ERROR))
	c.fatalLogger.SetOutput(fallback(FATAL))
}
