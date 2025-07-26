package logger

import "strings"

// ParseLogLevel converts a string representation of a level into the Level
// type. Unknown values default to INFO.
func ParseLogLevel(level string) Level {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN":
		return WARN
	case "ERROR":
		return ERROR
	case "FATAL":
		return FATAL
	default:
		return INFO
	}
}
