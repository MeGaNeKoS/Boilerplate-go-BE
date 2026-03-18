package logger

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"project-template/infrastructure/config"
)

var (
	coreMu     sync.Mutex
	sharedCore *loggerCore
)

// NewLogger creates a new logger instance using the provided configuration and
// request identifiers.
func NewLogger(logConfig config.LogConfig, parentID, childID string) (Logger, error) {
	coreMu.Lock()
	defer coreMu.Unlock()
	if sharedCore == nil {
		var err error
		sharedCore, err = initLoggerCore(logConfig)
		if err != nil {
			return nil, err
		}
	}
	return &loggerImpl{core: sharedCore, parentID: parentID, childID: childID}, nil
}

func initLoggerCore(logConfig config.LogConfig) (*loggerCore, error) {
	level := ParseLogLevel(logConfig.VerboseLevel)

	if err := os.MkdirAll(logConfig.Path, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	defaultLogPath := filepath.Join(logConfig.Path, logConfig.FileName)
	defaultFile, err := os.OpenFile(defaultLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open default log file: %w", err)
	}

	levelFiles := make(map[Level]*os.File)
	levelPaths := make(map[Level]string)
	writers := make(map[slog.Level]io.Writer)

	for strLevel, fileName := range logConfig.PerLevelFiles {
		lvl := ParseLogLevel(strLevel)
		if fileName == "" {
			continue
		}
		fullPath := filepath.Join(logConfig.Path, fileName)
		f, err := os.OpenFile(fullPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			_ = defaultFile.Close()
			for _, lf := range levelFiles {
				_ = lf.Close()
			}
			return nil, fmt.Errorf("failed to open %s log file: %w", strLevel, err)
		}
		levelFiles[lvl] = f
		levelPaths[lvl] = fullPath
		writers[lvl.toSlog()] = io.MultiWriter(defaultFile, f)
	}

	mu := &sync.Mutex{}
	handler := &pipeHandler{
		mu:            mu,
		level:         level.toSlog(),
		toStdout:      logConfig.AlsoLogToStdout,
		writers:       writers,
		defaultWriter: defaultFile,
	}

	core := &loggerCore{
		cfg:           logConfig,
		level:         level,
		toStdout:      logConfig.AlsoLogToStdout,
		logFile:       defaultFile,
		logFilePath:   defaultLogPath,
		levelFile:     levelFiles,
		levelFilePath: levelPaths,
		handler:       handler,
	}
	core.startWatchers()
	return core, nil
}

// CloseLogFile releases any resources associated with the shared logger core.
// It is primarily intended for use in tests so temporary log files can be
// removed on Windows where open files cannot be deleted.
func CloseLogFile() {
	coreMu.Lock()
	if sharedCore == nil {
		coreMu.Unlock()
		return
	}

	oldSharedCore := sharedCore
	sharedCore = nil
	coreMu.Unlock()
	if oldSharedCore.logFile != nil {
		_ = oldSharedCore.logFile.Close()
	}
	for _, w := range oldSharedCore.watchers {
		_ = w.Close()
	}

	seen := map[*os.File]bool{}
	for _, f := range oldSharedCore.levelFile {
		if f != nil && !seen[f] && f != oldSharedCore.logFile {
			_ = f.Close()
			seen[f] = true
		}
	}
}
