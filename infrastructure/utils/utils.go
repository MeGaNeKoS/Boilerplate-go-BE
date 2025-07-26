package utils

import (
	"crypto/rand"
	"encoding/hex"
	"io"
)

var randReader io.Reader = rand.Reader

// UniqueIdByTime generates a pseudo-random hexadecimal string. The counter
// argument is unused but kept for backward compatibility with older helpers.
// It returns the generated ID or an error if the random source fails.
func UniqueIdByTime(_ uint64) (string, error) {
	b := make([]byte, 16)
	if _, err := io.ReadFull(randReader, b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
