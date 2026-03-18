package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"sync"
)

const timestampFormat = "2006-01-02 15:04:05.000"

// pipeHandler is a custom slog.Handler that writes pipe-delimited log lines
// and routes output to per-level file writers.
type pipeHandler struct {
	mu       *sync.Mutex
	level    slog.Level
	toStdout bool

	// writers maps slog levels to their io.Writer destination.
	writers map[slog.Level]io.Writer

	// defaultWriter is used when no level-specific writer is configured.
	defaultWriter io.Writer
}

func (h *pipeHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *pipeHandler) Handle(_ context.Context, r slog.Record) error {
	label := labelFromSlog(r.Level)

	var parentID, childID string
	r.Attrs(func(a slog.Attr) bool {
		switch a.Key {
		case "parentID":
			parentID = a.Value.String()
		case "childID":
			childID = a.Value.String()
		}
		return true
	})

	file := "unknown"
	line := 0
	if r.PC != 0 {
		frames := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := frames.Next()
		if f.File != "" {
			file = f.File
			line = f.Line
		}
	}

	logLine := fmt.Sprintf("%s|%s|%s|%s|%s|%d|%s",
		r.Time.Format(timestampFormat), parentID, childID, label, file, line, r.Message)

	h.mu.Lock()
	_, _ = fmt.Fprintln(h.writerForLevel(r.Level), logLine)
	if h.toStdout {
		_, _ = fmt.Fprintln(os.Stdout, logLine)
	}
	h.mu.Unlock()

	return nil
}

func (h *pipeHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }

func (h *pipeHandler) WithGroup(_ string) slog.Handler { return h }

func (h *pipeHandler) writerForLevel(level slog.Level) io.Writer {
	if w, ok := h.writers[level]; ok {
		return w
	}
	return h.defaultWriter
}
