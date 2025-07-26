package utils

import "strings"

// ReplacePlaceholders substitutes ${key} tokens in the string using the provided
// map. Escaped tokens ($${key}) are left intact.
func ReplacePlaceholders(s string, values map[string]string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		// Look for next "${"
		start := strings.Index(s[i:], "${")
		if start == -1 {
			b.WriteString(s[i:])
			break
		}
		start += i

		// Check for escape: is there a $ immediately before "${"?
		escaped := false
		if start > 0 && s[start-1] == '$' {
			escaped = true
		}

		// Write up to (but not including) the found "${"
		b.WriteString(s[i : start-(1*boolToInt(escaped))])

		// Find the closing "}"
		end := strings.Index(s[start:], "}")
		if end == -1 {
			// No closing "}", write rest and break
			b.WriteString(s[start:])
			break
		}
		end = start + end

		key := s[start+2 : end]
		if escaped {
			// Output as literal ${key}
			b.WriteString("${" + key + "}")
		} else if val, ok := values[key]; ok {
			b.WriteString(val)
		} else {
			b.WriteString("${" + key + "}")
		}

		// Move past this match
		i = end + 1
	}
	return b.String()
}

// Helper to convert bool to int (true=1, false=0)
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
