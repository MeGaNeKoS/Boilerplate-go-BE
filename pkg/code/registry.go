package code

import (
	"bufio"
	"os"
	"runtime"
	"strings"
)

var registry = map[string]*Code{}

// register stores the code using the variable name of the caller and returns
// the same pointer so callers don't receive a copy.
func register(c *Code) *Code {
	if _, file, line, ok := runtime.Caller(1); ok {
		if name := varName(file, line); name != "" {
			registry[name] = c
		}
	}
	return c
}

// varName retrieves the identifier on the given line of code.
func varName(file string, line int) string {
	f, err := os.Open(file)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()
	scanner := bufio.NewScanner(f)
	for i := 1; scanner.Scan(); i++ {
		if i == line {
			l := scanner.Text()
			if idx := strings.Index(l, "="); idx != -1 {
				return strings.TrimSpace(l[:idx])
			}
			break
		}
	}
	return ""
}

// Lookup returns the Code pointer for the given name.
func Lookup(name string) (*Code, bool) {
	c, ok := registry[name]
	return c, ok
}

// All returns a copy of the registered codes map.
func All() map[string]*Code {
	out := make(map[string]*Code, len(registry))
	for k, v := range registry {
		out[k] = v
	}
	return out
}
