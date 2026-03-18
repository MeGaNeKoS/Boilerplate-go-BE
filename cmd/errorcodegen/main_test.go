package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/bouk/monkey"
	"golang.org/x/tools/go/packages"
)

func TestMainGeneratesFile(t *testing.T) {
	path := "errorcodes_gen.go"
	_ = os.Remove(path)
	main()
	defer func() { _ = os.Remove(path) }()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated: %v", err)
	}
	if !strings.Contains(string(data), "GetItem") {
		t.Fatalf("generated file missing expected handler")
	}
}

func TestMainCreateOutputError(t *testing.T) {
	orig := createOutput
	createOutput = func(string) (io.WriteCloser, error) { return nil, fmt.Errorf("create fail") }
	defer func() { createOutput = orig }()

	var got string
	monkey.Patch(log.Fatalf, func(format string, args ...any) {
		got = fmt.Sprintf(format, args...)
		panic("fatal")
	})
	defer monkey.Unpatch(log.Fatalf)

	defer func() {
		if r := recover(); r == nil || !strings.Contains(got, "create output") {
			t.Fatalf("expected fatal, got %v %q", r, got)
		}
	}()
	main()
}

func TestMainLoadPackageError(t *testing.T) {
	monkey.Patch(packages.Load, func(cfg *packages.Config, patterns ...string) ([]*packages.Package, error) {
		return nil, fmt.Errorf("boom")
	})
	defer monkey.Unpatch(packages.Load)

	var got string
	monkey.Patch(log.Fatalf, func(format string, args ...any) {
		got = fmt.Sprintf(format, args...)
		panic("fatal")
	})
	defer monkey.Unpatch(log.Fatalf)

	defer func() {
		if r := recover(); r == nil || !strings.Contains(got, "failed to load package") {
			t.Fatalf("expected fatal, got %v %q", r, got)
		}
	}()
	main()
}

func TestMainLoadPackageEmpty(t *testing.T) {
	monkey.Patch(packages.Load, func(cfg *packages.Config, patterns ...string) ([]*packages.Package, error) {
		return []*packages.Package{}, nil
	})
	defer monkey.Unpatch(packages.Load)

	var got string
	monkey.Patch(log.Fatalf, func(format string, args ...any) {
		got = fmt.Sprintf(format, args...)
		panic("fatal")
	})
	defer monkey.Unpatch(log.Fatalf)

	defer func() {
		if r := recover(); r == nil || !strings.Contains(got, "failed to load package") {
			t.Fatalf("expected fatal, got %v %q", r, got)
		}
	}()
	main()
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, fmt.Errorf("write error") }
func (failWriter) Close() error              { return nil }

func TestMainWriteOutputError(t *testing.T) {
	orig := createOutput
	createOutput = func(string) (io.WriteCloser, error) { return failWriter{}, nil }
	defer func() { createOutput = orig }()

	var got string
	monkey.Patch(log.Fatalf, func(format string, args ...any) {
		got = fmt.Sprintf(format, args...)
		panic("fatal")
	})
	defer monkey.Unpatch(log.Fatalf)

	defer func() {
		if r := recover(); r == nil || !strings.Contains(got, "write output") {
			t.Fatalf("expected write output fatal, got %v %q", r, got)
		}
	}()
	main()
}

type failCloser struct{ buf strings.Builder }

func (f *failCloser) Write(p []byte) (int, error) { return f.buf.Write(p) }
func (f *failCloser) Close() error                { return fmt.Errorf("close error") }

func TestMainCloseOutputError(t *testing.T) {
	orig := createOutput
	createOutput = func(string) (io.WriteCloser, error) { return &failCloser{}, nil }
	defer func() { createOutput = orig }()

	var got string
	monkey.Patch(log.Fatalf, func(format string, args ...any) {
		got = fmt.Sprintf(format, args...)
		panic("fatal")
	})
	defer monkey.Unpatch(log.Fatalf)

	defer func() {
		if r := recover(); r == nil || !strings.Contains(got, "close output") {
			t.Fatalf("expected close output fatal, got %v %q", r, got)
		}
	}()
	main()
}
