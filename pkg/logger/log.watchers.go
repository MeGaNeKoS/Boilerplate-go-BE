package logger

import (
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

func (c *loggerCore) startWatchers() {
	paths := []string{c.logFilePath}
	for _, p := range c.levelFilePath {
		paths = append(paths, p)
	}
	for _, p := range paths {
		w, err := fsnotify.NewWatcher()
		if err != nil {
			continue
		}
		if err := w.Add(filepath.Dir(p)); err != nil {
			_ = w.Close()
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

	for {
		f, err := openLogFile(path)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if filepath.Clean(path) == filepath.Clean(c.logFilePath) {
			if c.logFile != nil {
				_ = c.logFile.Close()
			}
			c.logFile = f
		} else {
			for lvl, p := range c.levelFilePath {
				if filepath.Clean(p) == filepath.Clean(path) {
					if lf := c.levelFile[lvl]; lf != nil && lf != c.logFile {
						_ = lf.Close()
					}
					c.levelFile[lvl] = f
					break
				}
			}
		}
		c.rebuildLoggers()
		return
	}
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
