package utils

import (
	"bytes"
	"io"
	"testing"
)

type errorReader struct{}

func (errorReader) Read(p []byte) (int, error) { return 0, io.ErrUnexpectedEOF }

func TestUniqueIdByTime(t *testing.T) {
	old := randReader
	defer func() { randReader = old }()
	randReader = bytes.NewBuffer(bytes.Repeat([]byte{0xAB}, 16))
	id, err := UniqueIdByTime(0)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if id != "abababababababababababababababab" {
		t.Fatalf("unexpected id %s", id)
	}
}

func TestUniqueIdByTimeError(t *testing.T) {
	old := randReader
	defer func() { randReader = old }()
	randReader = errorReader{}
	if _, err := UniqueIdByTime(0); err == nil {
		t.Fatalf("expected error")
	}
}
