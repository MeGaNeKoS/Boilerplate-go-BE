package logger

import (
	"log"
	"os"
	"sync"

	"project-template/infrastructure/config"

	"github.com/fsnotify/fsnotify"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
	FATAL
)

type Logger interface {
	DebugF(format string, args ...interface{})
	InfoF(format string, args ...interface{})
	WarnF(format string, args ...interface{})
	ErrorF(format string, args ...interface{})
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

	debugLogger *log.Logger
	infoLogger  *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger
	fatalLogger *log.Logger

	parentID string // For request id tracing through entire request lifetime across all service
	childID  string // For request id tracing through entire request lifetime in specific endpoint only
}

type loggerImpl struct {
	core     *loggerCore
	parentID string
	childID  string
	depth    int
}
