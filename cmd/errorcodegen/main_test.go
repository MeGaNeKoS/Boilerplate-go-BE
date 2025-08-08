package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/bouk/monkey"
	"golang.org/x/tools/go/packages"
)

func TestLoadPackageCaching(t *testing.T) {
	pkgCache = map[string]*packages.Package{}
	var calls int
	monkey.Patch(packages.Load, func(cfg *packages.Config, patterns ...string) ([]*packages.Package, error) {
		calls++
		if strings.Contains(patterns[0], "fail") {
			return nil, errors.New("boom")
		}
		return []*packages.Package{{}}, nil
	})
	defer monkey.Unpatch(packages.Load)

	if p := loadPackage("ok"); p == nil {
		t.Fatalf("expected package")
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
	if p := loadPackage("ok"); p == nil || calls != 1 {
		t.Fatalf("expected cache reuse, calls=%d", calls)
	}
	if p := loadPackage("fail"); p != nil || calls != 2 {
		t.Fatalf("expected failure, calls=%d", calls)
	}
	if p := loadPackage("fail"); p != nil || calls != 2 {
		t.Fatalf("expected cached failure, calls=%d", calls)
	}
}

func TestInferFrom(t *testing.T) {
	pkgCache = map[string]*packages.Package{}
	codes := map[string]struct{}{}
	inferFrom(modulePath+"/server/rest/handlers", "GetItem", map[string]bool{}, codes)
	if _, ok := codes["ErrItemNotFound"]; !ok {
		t.Fatalf("expected ErrItemNotFound")
	}
	if _, ok := codes["ErrInternalServerError"]; !ok {
		t.Fatalf("expected ErrInternalServerError")
	}
}

func TestInferFromNilPackage(t *testing.T) {
	pkgCache = map[string]*packages.Package{}
	monkey.Patch(loadPackage, func(string) *packages.Package { return nil })
	defer monkey.Unpatch(loadPackage)
	codes := map[string]struct{}{}
	inferFrom("none", "Foo", map[string]bool{}, codes)
	if len(codes) != 0 {
		t.Fatalf("expected no codes, got %d", len(codes))
	}
}

func TestMainGeneratesFile(t *testing.T) {
	pkgCache = map[string]*packages.Package{}
	path := "errorcodes_gen.go"
	_ = os.Remove(path)
	main()
	defer os.Remove(path)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read generated: %v", err)
	}
	if !strings.Contains(string(data), "GetItem") {
		t.Fatalf("generated file missing expected handler")
	}
}

func TestMainCreateOutputError(t *testing.T) {
	pkgCache = map[string]*packages.Package{}
	if err := os.Mkdir("errorcodes_gen.go", 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	defer os.RemoveAll("errorcodes_gen.go")

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

func TestInspectFuncNil(t *testing.T) {
	codes := map[string]struct{}{}
	inspectFunc("pkg", nil, nil, nil, codes)
	if len(codes) != 0 {
		t.Fatalf("expected no codes for nil decl")
	}

	decl := &ast.FuncDecl{}
	inspectFunc("pkg", decl, nil, nil, codes)
	if len(codes) != 0 {
		t.Fatalf("expected no codes for decl without body")
	}
}

func TestInspectFuncNewHumaError(t *testing.T) {
	src := `package p
        func f(){ newHumaError(code.ErrInternalServerError) }`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "p.go", src, 0)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	decl := file.Decls[0].(*ast.FuncDecl)
	codes := map[string]struct{}{}
	inspectFunc("p", decl, &types.Info{}, nil, codes)
	if _, ok := codes["ErrInternalServerError"]; !ok {
		t.Fatalf("expected ErrInternalServerError, got %v", codes)
	}
}

func TestMainLoadPackageError(t *testing.T) {
	pkgCache = map[string]*packages.Package{}
	monkey.Patch(packages.Load, func(cfg *packages.Config, patterns ...string) ([]*packages.Package, error) {
		return nil, errors.New("boom")
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
	pkgCache = map[string]*packages.Package{}
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
