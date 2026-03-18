package logger

import (
	"fmt"
	"log"
	"runtime"
	"time"
)

// callerDepth is the number of stack frames to skip so runtime.Caller
// reports the call site, not the logger internals.
// Caller (1) -> public method (2) -> log, so skip 2.
const callerDepth = 2

func (l *loggerImpl) log(level Level, label, msg string) {
	if level < l.core.level {
		return
	}

	_, file, line, ok := runtime.Caller(callerDepth)
	if !ok {
		file = "unknown"
		line = 0
	}

	timestamp := time.Now().Format(time.DateTime)
	logLine := fmt.Sprintf("%s|%s|%s|%s|%s|%d|%s",
		timestamp, l.parentID, l.childID, label, file, line, msg)

	var logger *log.Logger
	switch level {
	case DEBUG:
		logger = l.core.debugLogger
	case INFO:
		logger = l.core.infoLogger
	case WARN:
		logger = l.core.warnLogger
	case ERROR:
		logger = l.core.errorLogger
	case FATAL:
		logger = l.core.fatalLogger
	default:
		logger = l.core.infoLogger
	}

	logger.Println(logLine)

	if l.core.toStdout {
		fmt.Println(logLine)
	}
}

// DebugF logs a formatted debug level message.
func (l *loggerImpl) DebugF(format string, args ...interface{}) {
	l.log(DEBUG, "DEBUG", fmt.Sprintf(format, args...))
}

// InfoF logs a formatted info level message.
func (l *loggerImpl) InfoF(format string, args ...interface{}) {
	l.log(INFO, "INFO", fmt.Sprintf(format, args...))
}

// WarnF logs a formatted warning message.
func (l *loggerImpl) WarnF(format string, args ...interface{}) {
	l.log(WARN, "WARN", fmt.Sprintf(format, args...))
}

// ErrorF logs a formatted error message.
func (l *loggerImpl) ErrorF(format string, args ...interface{}) {
	l.log(ERROR, "ERROR", fmt.Sprintf(format, args...))
}

// FatalF logs a formatted fatal message.
func (l *loggerImpl) FatalF(format string, args ...interface{}) {
	l.log(FATAL, "FATAL", fmt.Sprintf(format, args...))
}

// Plain message convenience methods.

func (l *loggerImpl) Debug(msg string) { l.log(DEBUG, "DEBUG", msg) }
func (l *loggerImpl) Info(msg string)  { l.log(INFO, "INFO", msg) }
func (l *loggerImpl) Warn(msg string)  { l.log(WARN, "WARN", msg) }
func (l *loggerImpl) Error(msg string) { l.log(ERROR, "ERROR", msg) }
func (l *loggerImpl) Fatal(msg string) { l.log(FATAL, "FATAL", msg) }

// ParentID returns the request parent identifier.
func (l *loggerImpl) ParentID() string {
	return l.parentID
}

// ChildID returns the request child identifier.
func (l *loggerImpl) ChildID() string {
	return l.childID
}
