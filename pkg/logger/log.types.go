package logger

import (
	"log/slog"
	"os"
	"sync"

	"project-template/infrastructure/config"

	"github.com/fsnotify/fsnotify"
)

// Level mirrors slog.Level but keeps the existing API.
type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
	FATAL
)

// toSlog converts our Level to slog.Level.
func (l Level) toSlog() slog.Level {
	switch l {
	case DEBUG:
		return slog.LevelDebug
	case INFO:
		return slog.LevelInfo
	case WARN:
		return slog.LevelWarn
	case ERROR:
		return slog.LevelError
	case FATAL:
		return slog.LevelError + 4 // custom level above Error
	default:
		return slog.LevelInfo
	}
}

// labelFromSlog returns the pipe-delimited label for the log line.
func labelFromSlog(l slog.Level) string {
	switch {
	case l >= slog.LevelError+4:
		return "FATAL"
	case l >= slog.LevelError:
		return "ERROR"
	case l >= slog.LevelWarn:
		return "WARN"
	case l >= slog.LevelInfo:
		return "INFO"
	default:
		return "DEBUG"
	}
}

// Logger is the public interface used throughout the application.
type Logger interface {
	Debug(msg string)
	DebugF(format string, args ...interface{})
	Info(msg string)
	InfoF(format string, args ...interface{})
	Warn(msg string)
	WarnF(format string, args ...interface{})
	Error(msg string)
	ErrorF(format string, args ...interface{})
	Fatal(msg string)
	FatalF(format string, args ...interface{})
	ParentID() string
	ChildID() string
}

type loggerCore struct {
	cfg           config.LogConfig
	mu            sync.Mutex
	level         Level
	toStdout      bool
	logFile       *os.File
	logFilePath   string
	levelFile     map[Level]*os.File
	levelFilePath map[Level]string
	watchers      []*fsnotify.Watcher
	handler       slog.Handler
}

type loggerImpl struct {
	core     *loggerCore
	parentID string
	childID  string
}
