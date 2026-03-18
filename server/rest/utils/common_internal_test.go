package utils

import (
	"errors"
	"go/ast"
	"net/http"
	"reflect"
	"testing"

	"github.com/bouk/monkey"
	"github.com/danielgtaylor/huma/v2"
	"golang.org/x/tools/go/packages"

	"project-template/infrastructure/config"
	"project-template/pkg/code"
)

type inspectType struct{}

func (inspectType) Method() {}

// helperFunc is used to exercise the ident->*types.Func path in inspectFunc.
func helperFunc() {}

// testInspectTarget is analyzed via packages.Load in TestInspectFuncBranches.
// It is not called directly but its AST is inspected for code references.
var _ = testInspectTarget

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
	pkgs, err := packages.Load(cfg, modulePath+"/server/rest/utils")
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
	inspectFunc(decl, pkg.TypesInfo, map[string]bool{}, codes)
	if !errors.Is(codes["ErrBadRequest"], code.ErrBadRequest) {
		t.Fatalf("missing direct code reference")
	}
	if !errors.Is(codes["ErrInternalServerError"], code.ErrInternalServerError) {
		t.Fatalf("missing newHumaError reference")
	}
}

func TestInspectFuncNil(t *testing.T) {
	inspectFunc(nil, nil, nil, nil)
}

func TestLoadPackageCache(t *testing.T) {
	p1 := loadPackage(modulePath + "/server/rest/utils")
	if p1 == nil {
		t.Fatalf("package load failed")
	}
	p2 := loadPackage(modulePath + "/server/rest/utils")
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
	inferFrom(modulePath+"/server/rest/utils", "NoSuchFunc", map[string]bool{}, codes)
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
		if errors.Is(c, code.ErrBadRequest) {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected ErrBadRequest")
	}
}

// precomputedTarget is a dummy function used to exercise the precomputedErrors
// branch in InferErrorCodes.
func precomputedTarget() {}

func TestInferErrorCodesPrecomputed(t *testing.T) {
	precomputedErrors["precomputedTarget"] = []string{"ErrBadRequest"}
	t.Cleanup(func() { delete(precomputedErrors, "precomputedTarget") })

	got := InferErrorCodes(precomputedTarget)
	if len(got) != 1 || !errors.Is(got[0], code.ErrBadRequest) {
		t.Fatalf("expected precomputed error code")
	}
}

func TestUpperFirstEmpty(t *testing.T) {
	if got := upperFirst(""); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestTypeLabelPointer(t *testing.T) {
	// reflect.Pointer to a struct should dereference and return the struct name.
	typ := reflect.TypeOf((*code.Code)(nil))
	got := typeLabel(typ)
	if got != "Code" {
		t.Fatalf("expected Code, got %q", got)
	}
}

func TestTypeLabelAnonymousStructEmpty(t *testing.T) {
	// An anonymous struct with no fields should return "Any".
	typ := reflect.TypeOf(struct{}{})
	got := typeLabel(typ)
	if got != "Any" {
		t.Fatalf("expected Any, got %q", got)
	}
}

func TestTypeLabelAnonymousStructWithFields(t *testing.T) {
	// An anonymous struct with fields should return "Struct".
	typ := reflect.TypeOf(struct{ X int }{})
	got := typeLabel(typ)
	if got != "Struct" {
		t.Fatalf("expected Struct, got %q", got)
	}
}

func TestTypeLabelInterface(t *testing.T) {
	// Interface type should return "Any".
	typ := reflect.TypeOf((*error)(nil)).Elem()
	got := typeLabel(typ)
	if got != "Any" {
		t.Fatalf("expected Any, got %q", got)
	}
}

func TestTypeLabelDefaultKind(t *testing.T) {
	// A basic type like int has a name, so upperFirst applies.
	typ := reflect.TypeOf(0)
	got := typeLabel(typ)
	if got != "Int" {
		t.Fatalf("expected Int, got %q", got)
	}
}

func TestNewAuthErrorSetsWWWAuthenticate(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	wwwAuth := `Bearer realm="APP", error="invalid_token"`
	err := NewAuthError(code.ErrInvalidToken, wwwAuth)
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	var se huma.StatusError
	if !errors.As(err, &se) {
		t.Fatal("expected huma.StatusError")
	}
	if se.GetStatus() != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", se.GetStatus())
	}
	type headerer interface {
		GetHeaders() http.Header
	}
	var h headerer
	if !errors.As(err, &h) {
		t.Fatal("expected GetHeaders()")
	}
	got := h.GetHeaders().Get("WWW-Authenticate")
	if got != wwwAuth {
		t.Fatalf("expected %q, got %q", wwwAuth, got)
	}
}

func TestNewRetryableErrorSetsRetryAfter(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	err := NewRetryableError(code.ErrTooManyRequests, 120)
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	var se huma.StatusError
	if !errors.As(err, &se) {
		t.Fatal("expected huma.StatusError")
	}
	if se.GetStatus() != http.StatusTooManyRequests {
		t.Fatalf("expected 429, got %d", se.GetStatus())
	}
	type headerer interface {
		GetHeaders() http.Header
	}
	var h headerer
	if !errors.As(err, &h) {
		t.Fatal("expected GetHeaders()")
	}
	got := h.GetHeaders().Get("Retry-After")
	if got != "120" {
		t.Fatalf("expected '120', got %q", got)
	}
}



