package utils

import (
	"bytes"
	"strings"
	"testing"
)

func TestSecureString(t *testing.T) {
	s := SecureString("secret-token")
	if got := s.String(); got != strings.Repeat("*", len("secret-token")) {
		t.Fatalf("unexpected mask: %s", got)
	}
}

func TestSecureBytes(t *testing.T) {
	b := SecureBytes([]byte("byte-secret"))
	if got := b.String(); got != strings.Repeat("*", len("byte-secret")) {
		t.Fatalf("unexpected mask: %s", got)
	}
}

func TestSecureLongString(t *testing.T) {
	s := SecureString(strings.Repeat("a", 20))
	if got := s.String(); got != "aaaa...(len:20)...aaaa" {
		t.Fatalf("unexpected mask: %s", got)
	}
}

func TestSecureLongBytes(t *testing.T) {
	b := SecureBytes(bytes.Repeat([]byte("b"), 20))
	if got := b.String(); got != "bbbb...(len:20)...bbbb" {
		t.Fatalf("unexpected mask: %s", got)
	}
}

func TestSecureEmptyAndGoString(t *testing.T) {
	if got := SecureString("").String(); got != "<empty>" {
		t.Fatalf("empty string mask: %s", got)
	}
	if got := SecureBytes(nil).String(); got != "<empty>" {
		t.Fatalf("empty bytes mask: %s", got)
	}
	s := SecureString("val")
	if s.GoString() != s.String() {
		t.Fatalf("GoString mismatch")
	}
	b := SecureBytes([]byte("val"))
	if b.GoString() != b.String() {
		t.Fatalf("GoString bytes mismatch")
	}
}
