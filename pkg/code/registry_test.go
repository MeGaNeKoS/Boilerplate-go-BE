package code

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

var (
	ErrDummy = register(&Code{HTTPCode: http.StatusTeapot, Message: "dummy", InternalCode: 999})
)

func TestRegisterLookup(t *testing.T) {
	c, ok := Lookup("ErrDummy")
	if !ok {
		t.Fatalf("Lookup failed")
	}
	if !errors.Is(c, ErrDummy) {
		t.Fatalf("Lookup returned wrong pointer: %p != %p", c, ErrDummy)
	}
	if c.HTTPCode != http.StatusTeapot || c.Message != "dummy" || c.InternalCode != 999 {
		t.Fatalf("unexpected code: %#v", c)
	}
}

func TestAllReturnsCopy(t *testing.T) {
	all := All()
	if _, ok := all["ErrPayloadError"]; !ok {
		t.Fatalf("expected ErrPayloadError in map")
	}
	delete(all, "ErrPayloadError")
	if _, ok := Lookup("ErrPayloadError"); !ok {
		t.Fatalf("registry mutated after modifying copy")
	}
}

func TestVarName(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "tmp.go")
	content := "package main\n\nvar (\n    Example = 1\n    NoEquals\n)\n"
	if err := os.WriteFile(file, []byte(content), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if name := varName(file, 4); name != "Example" {
		t.Fatalf("varName wrong result: %q", name)
	}
	if name := varName(file, 5); name != "" {
		t.Fatalf("expected empty for line without equals, got %q", name)
	}
	if name := varName(filepath.Join(dir, "missing.go"), 1); name != "" {
		t.Fatalf("expected empty for missing file, got %q", name)
	}
}

func TestRegisterPointerPreserved(t *testing.T) {
	c := &Code{Message: "x"}
	if got := register(c); !errors.Is(got, c) {
		t.Fatalf("register returned different pointer")
	}
}
