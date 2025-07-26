package utils

import "strings"

// NormalizeBasePath ensures the base path starts and ends with a slash.
// The root path is returned as "/".
func NormalizeBasePath(base string) string {
	if base == "" {
		base = "/"
	}
	if !strings.HasPrefix(base, "/") {
		base = "/" + base
	}
	if !strings.HasSuffix(base, "/") {
		base = base + "/"
	}
	if base == "//" {
		return "/"
	}
	return base
}
