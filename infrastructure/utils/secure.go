package utils

import (
	"fmt"
	"strings"
)

// SecureString is a string alias that hides most of the value when printed. It
// implements fmt.Stringer so logging tokens does not leak them.
type SecureString string

// SecureBytes is a byte slice alias that behaves the same as SecureString when
// formatted.
type SecureBytes []byte

const (
	securePrefix = 4
	secureSuffix = 4
	secureMaxLen = 16
)

func (s SecureString) String() string {
	v := string(s)

	l := len(v)
	switch {
	case l == 0:
		return "<empty>"
	case l < secureMaxLen:
		return strings.Repeat("*", l)
	default:
		return fmt.Sprintf("%s...(len:%d)...%s", v[:securePrefix], l, v[l-secureSuffix:])
	}
}

// GoString masks the value for %#v formatting as well.
func (s SecureString) GoString() string { return s.String() }

func (s SecureBytes) String() string {
	b := []byte(s)

	l := len(b)
	switch {
	case l == 0:
		return "<empty>"
	case l < secureMaxLen:
		return strings.Repeat("*", l)
	default:
		return fmt.Sprintf("%s...(len:%d)...%s", string(b[:securePrefix]), l, string(b[l-secureSuffix:]))
	}
}

// GoString masks the value for %#v formatting as well.
func (s SecureBytes) GoString() string { return s.String() }
