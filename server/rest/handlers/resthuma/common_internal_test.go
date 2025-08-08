package resthuma

import (
	"errors"
	"go/ast"
	"testing"

	"github.com/bouk/monkey"
	"golang.org/x/tools/go/packages"

	"project-template/pkg/code"
)

type inspectType struct{}

func (inspectType) Method() {}

// helperFunc is used to exercise the ident->*types.Func path in inspectFunc.
func helperFunc() {}

func testInspectTarget() {
	_ = code.ErrBadRequest
	_ = newHumaError(code.ErrInternalServerError)
	var fn func()
	fn()
	var it inspectType
	it.Method()
	helperFunc()
}

func TestInspectFuncBranches(t *testing.T) {
	cfg := &packages.Config{Mode: packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedFiles, Tests: true}
	pkgs, err := packages.Load(cfg, modulePath+"/server/rest/handlers/resthuma")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	var pkg *packages.Package
	var decl *ast.FuncDecl
	for _, p := range pkgs {
		for _, f := range p.Syntax {
			for _, d := range f.Decls {
				if fd, ok := d.(*ast.FuncDecl); ok && fd.Name.Name == "testInspectTarget" {
					pkg = p
					decl = fd
					break
				}
			}
		}
	}
	if decl == nil {
		t.Fatalf("test function not found")
	}
	codes := map[string]*code.Code{}
	inspectFunc(modulePath+"/server/rest/handlers/resthuma", decl, pkg.TypesInfo, map[string]bool{}, codes)
	if codes["ErrBadRequest"] != code.ErrBadRequest {
		t.Fatalf("missing direct code reference")
	}
	if codes["ErrInternalServerError"] != code.ErrInternalServerError {
		t.Fatalf("missing newHumaError reference")
	}
}

func TestInspectFuncNil(t *testing.T) {
	inspectFunc("", nil, nil, nil, nil)
}

func TestLoadPackageCache(t *testing.T) {
	p1 := loadPackage(modulePath + "/server/rest/handlers/resthuma")
	if p1 == nil {
		t.Fatalf("package load failed")
	}
	p2 := loadPackage(modulePath + "/server/rest/handlers/resthuma")
	if p2 != p1 {
		t.Fatalf("expected cached package")
	}
}

func TestLoadPackageMissing(t *testing.T) {
	patch := monkey.Patch(packages.Load, func(_ *packages.Config, _ ...string) ([]*packages.Package, error) {
		return nil, errors.New("load error")
	})
	defer patch.Unpatch()
	if p := loadPackage("error/path"); p != nil {
		t.Fatalf("expected nil for load failure")
	}
	// verify cache hit for failed load
	if p := loadPackage("error/path"); p != nil {
		t.Fatalf("expected cached nil package")
	}
}

func TestInferFromBranches(t *testing.T) {
	inferFrom("x", "y", map[string]bool{"x.y": true}, map[string]*code.Code{})
	codes := map[string]*code.Code{}
	inferFrom("missing/package", "fn", map[string]bool{}, codes)
	inferFrom(modulePath+"/server/rest/handlers/resthuma", "NoSuchFunc", map[string]bool{}, codes)
	if len(codes) != 0 {
		t.Fatalf("expected no codes for missing functions")
	}
}

func TestInferFromNilPackage(t *testing.T) {
	patch := monkey.Patch(packages.Load, func(_ *packages.Config, _ ...string) ([]*packages.Package, error) {
		return nil, errors.New("oops")
	})
	defer patch.Unpatch()
	inferFrom("bad/pkg", "fn", map[string]bool{}, map[string]*code.Code{})
}

func TestInferErrorCodesLoop(t *testing.T) {
	patch := monkey.Patch(inferFrom, func(pkgPath, fnName string, visited map[string]bool, codes map[string]*code.Code) {
		codes["ErrBadRequest"] = code.ErrBadRequest
	})
	defer patch.Unpatch()

	got := InferErrorCodes(newHumaError)
	var found bool
	for _, c := range got {
		if c == code.ErrBadRequest {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected ErrBadRequest")
	}
}
