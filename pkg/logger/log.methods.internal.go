// Receiver Policy: POINTER RECEIVER ONLY
// Purpose: Internal methods may mutate loggerImpl (e.g., depth).
// Do not use value receivers in this file.

package logger

import (
	"fmt"
	"log"
	"runtime"
	"time"
)

func (l *loggerImpl) log(level Level, label, msg string) {
	if level < l.core.level {
		return
	}

	_, file, line, ok := runtime.Caller(l.depth + 1)
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
