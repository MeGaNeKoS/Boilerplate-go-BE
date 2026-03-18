package logger

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"time"
)

// callerPC returns the program counter of the caller's caller,
// skipping the specified number of frames.
func callerPC(skip int) uintptr {
	var pcs [1]uintptr
	runtime.Callers(skip, pcs[:])
	return pcs[0]
}

// emit is the single entry point for all log writes. It checks the level,
// builds the record, and hands it to the handler.
func (l *loggerImpl) emit(level slog.Level, msg string) {
	ctx := context.Background()
	if !l.core.handler.Enabled(ctx, level) {
		return
	}
	r := slog.NewRecord(time.Now(), level, msg, callerPC(4))
	r.AddAttrs(
		slog.String("parentID", l.parentID),
		slog.String("childID", l.childID),
	)
	_ = l.core.handler.Handle(ctx, r)
}

// DebugF logs a formatted debug level message.
// The level check runs before fmt.Sprintf to avoid allocation when disabled.
func (l *loggerImpl) DebugF(format string, args ...interface{}) {
	lvl := DEBUG.toSlog()
	if !l.core.handler.Enabled(context.Background(), lvl) {
		return
	}
	l.emit(lvl, fmt.Sprintf(format, args...))
}

// InfoF logs a formatted info level message.
func (l *loggerImpl) InfoF(format string, args ...interface{}) {
	lvl := INFO.toSlog()
	if !l.core.handler.Enabled(context.Background(), lvl) {
		return
	}
	l.emit(lvl, fmt.Sprintf(format, args...))
}

// WarnF logs a formatted warning message.
func (l *loggerImpl) WarnF(format string, args ...interface{}) {
	lvl := WARN.toSlog()
	if !l.core.handler.Enabled(context.Background(), lvl) {
		return
	}
	l.emit(lvl, fmt.Sprintf(format, args...))
}

// ErrorF logs a formatted error message.
func (l *loggerImpl) ErrorF(format string, args ...interface{}) {
	lvl := ERROR.toSlog()
	if !l.core.handler.Enabled(context.Background(), lvl) {
		return
	}
	l.emit(lvl, fmt.Sprintf(format, args...))
}

// FatalF logs a formatted fatal message.
func (l *loggerImpl) FatalF(format string, args ...interface{}) {
	lvl := FATAL.toSlog()
	if !l.core.handler.Enabled(context.Background(), lvl) {
		return
	}
	l.emit(lvl, fmt.Sprintf(format, args...))
}

// Plain message convenience methods.

func (l *loggerImpl) Debug(msg string) { l.emit(DEBUG.toSlog(), msg) }
func (l *loggerImpl) Info(msg string)  { l.emit(INFO.toSlog(), msg) }
func (l *loggerImpl) Warn(msg string)  { l.emit(WARN.toSlog(), msg) }
func (l *loggerImpl) Error(msg string) { l.emit(ERROR.toSlog(), msg) }
func (l *loggerImpl) Fatal(msg string) { l.emit(FATAL.toSlog(), msg) }

// ParentID returns the request parent identifier.
func (l *loggerImpl) ParentID() string { return l.parentID }

// ChildID returns the request child identifier.
func (l *loggerImpl) ChildID() string { return l.childID }
