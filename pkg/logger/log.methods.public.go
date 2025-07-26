// Receiver Policy: VALUE RECEIVER ONLY
// Purpose: All public methods use value receivers to isolate goroutine state.
// Do not use pointer receivers in this file.

package logger

import (
	"fmt"
)

// DebugF logs a formatted debug level message.
func (l loggerImpl) DebugF(format string, args ...interface{}) {
	l.depth++
	l.log(DEBUG, "DEBUG", fmt.Sprintf(format, args...))
}

// InfoF logs a formatted info level message.
func (l loggerImpl) InfoF(format string, args ...interface{}) {
	l.depth++
	l.log(INFO, "INFO", fmt.Sprintf(format, args...))
}

// WarnF logs a formatted warning message.
func (l loggerImpl) WarnF(format string, args ...interface{}) {
	l.depth++
	l.log(WARN, "WARN", fmt.Sprintf(format, args...))
}

// ErrorF logs a formatted error message.
func (l loggerImpl) ErrorF(format string, args ...interface{}) {
	l.depth++
	l.log(ERROR, "ERROR", fmt.Sprintf(format, args...))
}

// FatalF logs a formatted fatal message.
func (l loggerImpl) FatalF(format string, args ...interface{}) {
	l.depth++
	l.log(FATAL, "FATAL", fmt.Sprintf(format, args...))
}

// ParentID returns the request parent identifier.
func (l loggerImpl) ParentID() string {
	return l.parentID
}

// ChildID returns the request child identifier.
func (l loggerImpl) ChildID() string {
	return l.childID
}
