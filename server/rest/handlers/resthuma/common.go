package resthuma

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/types"
	"net/http"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"golang.org/x/tools/go/packages"

	"github.com/danielgtaylor/huma/v2"

	"project-template/infrastructure/config"
	"project-template/infrastructure/dto/response"
	"project-template/pkg/code"
)

const modulePath = "project-template"

// caches for package and function error inference to avoid repeated traversal
var (
	pkgCache  = map[string]*packages.Package{}
	funcCache = map[string][]*code.Code{}
	cacheMu   sync.Mutex
)

// Empty is an empty input used when a route does not accept any parameters.
type Empty struct{}

// example registry for OpenAPI components
var (
	examples          = map[string]*huma.Example{}
	exampleCounter    = map[string]int{}
	exampleValueIndex = map[string]string{}
)

var registry = huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)

// schema for BaseResponse without a body
var baseResponseSchema = huma.SchemaFromType(registry, reflect.TypeOf(response.BaseResponse{}))

// schema for RFC 7807 problem details
var problemDetailSchema = huma.SchemaFromType(registry, reflect.TypeOf(response.ProblemDetail{}))

// humaError adapts project error codes to huma.StatusError.
type humaError struct {
	status int
	Body   response.ProblemDetail
}

func (e *humaError) Error() string  { return e.Body.Title }
func (e *humaError) GetStatus() int { return e.status }
func (e *humaError) GetHeaders() http.Header {
	h := http.Header{}
	h.Set("Content-Type", "application/problem+json")
	return h
}

// newHumaError converts a Code to humaError. The optional message overrides the
// default message from the Code.
func newHumaError(c *code.Code, msg ...string) *humaError {
	e := *c
	if len(msg) > 0 {
		e.Message = msg[0]
	}
	app := ""
	if config.Cfg != nil {
		app = config.Cfg.AppName
	}
	return &humaError{
		status: e.HTTPCode,
		Body: response.ProblemDetail{
			Type:   fmt.Sprintf("/errors/%d", e.InternalCode),
			Title:  e.Message,
			Status: e.HTTPCode,
			Code:   fmt.Sprintf("%s-ERR-%d", app, e.InternalCode),
		},
	}
}

// NewError converts a Code to an error implementing huma.StatusError.
func NewError(c *code.Code, msg ...string) error {
	return newHumaError(c, msg...)
}

// success wraps a successful response body with our standard GenericResponse.
// If status is 0, http.StatusOK is used.
func success[T any](body T, status int) *response.GenericResponse[T] {
	if status == 0 {
		status = http.StatusOK
	}
	app := ""
	if config.Cfg != nil {
		app = config.Cfg.AppName
	}
	return &response.GenericResponse[T]{
		BaseResponse: response.BaseResponse{
			StatusCode:    fmt.Sprintf("%s-%d", app, status),
			ReturnMessage: "Success",
		},
		Body: body,
	}
}

// SuccessResponse builds a Huma response with the standard success envelope.
func SuccessResponse[T any](status int, body T) *Response[*response.GenericResponse[T]] {
	return NewResponse(status, success(body, status))
}

// NoContentResponse returns a 204 No Content response.
func NoContentResponse() *Response[struct{}] {
	return NewResponse(http.StatusNoContent, struct{}{})
}

// exampleError returns a problem detail example for the given error code.
func exampleError(c *code.Code) *response.ProblemDetail {
	app := ""
	if config.Cfg != nil {
		app = config.Cfg.AppName
	}
	return &response.ProblemDetail{
		Type:   fmt.Sprintf("/errors/%d", c.InternalCode),
		Title:  c.Message,
		Status: c.HTTPCode,
		Code:   fmt.Sprintf("%s-ERR-%d", app, c.InternalCode),
	}
}

// registerExample stores the value under the given name for later component registration
func registerExample(name string, value any) *huma.Example {
	b, err := json.Marshal(value)
	if err == nil {
		if exist, ok := exampleValueIndex[string(b)]; ok {
			return &huma.Example{Ref: "#/components/examples/" + exist}
		}
	}

	if _, ok := examples[name]; ok {
		exampleCounter[name]++
		name = fmt.Sprintf("%s-%d", name, exampleCounter[name])
	}
	examples[name] = &huma.Example{Value: value}
	if err == nil {
		exampleValueIndex[string(b)] = name
	}
	return &huma.Example{Ref: "#/components/examples/" + name}
}

// Success represents a successful response example and metadata.
type Success[T any] struct {
	Status      int
	Description string
	Body        T
}

// NewSuccess creates a Success value with the given status, description and body.
func NewSuccess[T any](status int, desc string, body T) Success[T] {
	return Success[T]{Status: status, Description: desc, Body: body}
}

// RegisterExamples adds all stored examples to the API components.
func RegisterExamples(api huma.API) {
	if api.OpenAPI().Components == nil {
		api.OpenAPI().Components = &huma.Components{}
	}
	if api.OpenAPI().Components.Examples == nil {
		api.OpenAPI().Components.Examples = map[string]*huma.Example{}
	}
	for n, ex := range examples {
		api.OpenAPI().Components.Examples[n] = ex
	}
}

// RegisterSchemas adds all generated schemas to the API components.
func RegisterSchemas(api huma.API) {
	if api.OpenAPI().Components == nil {
		api.OpenAPI().Components = &huma.Components{}
	}
	if api.OpenAPI().Components.Schemas == nil {
		api.OpenAPI().Components.Schemas = huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
	}
	for n, s := range registry.Map() {
		api.OpenAPI().Components.Schemas.Map()[n] = s
	}
}

// ResponseMap builds a map of Huma responses with one or more success
// examples plus any number of error codes, using example names derived from
// the provided operation name and body type.
func ResponseMap[T any](op string, successes []Success[T], errs []*code.Code) map[string]*huma.Response {
	responses := map[string]*huma.Response{}
	t := reflect.TypeOf((*T)(nil)).Elem()
	typed := !(t.Kind() == reflect.Interface && t.NumMethod() == 0) && !(t.Kind() == reflect.Struct && t.NumField() == 0)
	var typedSchema *huma.Schema
	if typed {
		typedSchema = huma.SchemaFromType(registry, reflect.TypeOf(response.GenericResponse[T]{}))
	}
	for _, s := range successes {
		if s.Status == 0 {
			s.Status = http.StatusOK
		}
		if s.Status == http.StatusNoContent {
			responses[fmt.Sprintf("%d", s.Status)] = &huma.Response{Description: s.Description}
			continue
		}
		schema := typedSchema
		var example any
		if typed {
			example = success[T](s.Body, s.Status)
		} else {
			// treat as a BaseResponse without a body
			schema = baseResponseSchema
			ex := success[any](nil, s.Status)
			example = &response.BaseResponse{
				StatusCode:    ex.StatusCode,
				ReturnMessage: ex.ReturnMessage,
			}
		}
		typeName := strings.TrimPrefix(t.String(), "[]")
		if typeName == "interface {}" || typeName == "struct {}" {
			typeName = "any"
		}
		base := fmt.Sprintf("%s-%s-%d", op, strings.TrimPrefix(typeName, "*"), s.Status)
		responses[fmt.Sprintf("%d", s.Status)] = &huma.Response{
			Description: s.Description,
			Content: map[string]*huma.MediaType{
				"application/json": {
					Schema: schema,
					Examples: map[string]*huma.Example{
						"success": registerExample(
							base,
							example,
						),
					},
				},
			},
		}
	}
	for _, e := range errs {
		name := fmt.Sprintf("error-%d", e.InternalCode)
		responses[fmt.Sprintf("%d", e.HTTPCode)] = &huma.Response{
			Description: e.Message,
			Content: map[string]*huma.MediaType{
				"application/problem+json": {
					Schema: problemDetailSchema,
					Examples: map[string]*huma.Example{
						name: registerExample(name, exampleError(e)),
					},
				},
			},
		}
	}
	return responses
}

// InferErrorCodes parses the source of the given handler function and returns
// any error codes used directly or by functions it invokes within this module.
// If precomputed data exists for the handler, it is returned without further
// analysis to avoid runtime overhead.
func InferErrorCodes(handler any) []*code.Code {
	start := time.Now()
	v := reflect.ValueOf(handler)
	if v.Kind() != reflect.Func {
		return nil
	}
	fn := runtime.FuncForPC(v.Pointer())
	if fn == nil {
		return nil
	}
	name := fn.Name()
	last := strings.LastIndex(name, ".")
	if last == -1 {
		return nil
	}
	pkgPath := name[:last]
	fnName := name[last+1:]

	// Check for precomputed errors generated at build time.
	if names, ok := precomputedErrors[fnName]; ok {
		out := make([]*code.Code, 0, len(names))
		for _, n := range names {
			if c, ok := code.Lookup(n); ok {
				out = append(out, c)
			}
		}
		return out
	}

	cacheMu.Lock()
	if cached, ok := funcCache[pkgPath+"."+fnName]; ok {
		cacheMu.Unlock()
		fmt.Printf("inferErrorCodes %s.%s cache hit (%v)\n", pkgPath, fnName, time.Since(start))
		return cached
	}
	cacheMu.Unlock()

	codes := map[string]*code.Code{}
	visited := map[string]bool{}
	inferFrom(pkgPath, fnName, visited, codes)

	out := make([]*code.Code, 0, len(codes))
	for _, c := range codes {
		out = append(out, c)
	}

	cacheMu.Lock()
	funcCache[pkgPath+"."+fnName] = out
	cacheMu.Unlock()

	fmt.Printf("inferErrorCodes %s.%s took %v\n", pkgPath, fnName, time.Since(start))

	return out
}

// inferFrom loads the given package and function then walks its AST looking for
// error codes and recursively inspects any project functions it calls.
func inferFrom(pkgPath, fnName string, visited map[string]bool, codes map[string]*code.Code) {
	key := pkgPath + "." + fnName
	if visited[key] {
		return
	}
	visited[key] = true

	pkg := loadPackage(pkgPath)
	if pkg == nil {
		return
	}
	for i, f := range pkg.Syntax {
		for _, d := range f.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Name.Name != fnName {
				continue
			}
			inspectFunc(pkgPath, fd, pkg.TypesInfo, visited, codes)
			return
		}
		_ = i
	}
}

// loadPackage returns the package for the given path, caching results so the
// same package isn't repeatedly loaded during error inference.
func loadPackage(pkgPath string) *packages.Package {
	start := time.Now()
	cacheMu.Lock()
	if pkg, ok := pkgCache[pkgPath]; ok {
		cacheMu.Unlock()
		fmt.Printf("loadPackage %s cache hit (%v)\n", pkgPath, time.Since(start))
		return pkg
	}
	cacheMu.Unlock()

	cfg := &packages.Config{Mode: packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo | packages.NeedFiles}
	pkgs, err := packages.Load(cfg, pkgPath)
	if err != nil || len(pkgs) == 0 {
		cacheMu.Lock()
		pkgCache[pkgPath] = nil
		cacheMu.Unlock()
		fmt.Printf("loadPackage %s failed (%v)\n", pkgPath, time.Since(start))
		return nil
	}
	pkg := pkgs[0]
	cacheMu.Lock()
	pkgCache[pkgPath] = pkg
	cacheMu.Unlock()
	fmt.Printf("loadPackage %s took %v\n", pkgPath, time.Since(start))
	return pkg
}

// inspectFunc walks the given function body collecting error codes and
// recursively following calls to other project functions.
func inspectFunc(pkgPath string, decl *ast.FuncDecl, info *types.Info, visited map[string]bool, codes map[string]*code.Code) {
	if decl == nil || decl.Body == nil {
		return
	}
	ast.Inspect(decl.Body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CallExpr:
			// newHumaError(code.Err...)
			if id, ok := n.Fun.(*ast.Ident); ok && id.Name == "newHumaError" {
				if len(n.Args) > 0 {
					if sel, ok := n.Args[0].(*ast.SelectorExpr); ok {
						if pkgIdent, ok := sel.X.(*ast.Ident); ok && pkgIdent.Name == "code" {
							if c, ok := code.Lookup(sel.Sel.Name); ok {
								codes[sel.Sel.Name] = c
							}
						}
					}
				}
				return true
			}

			// other function/method calls
			var obj *types.Func
			switch fun := n.Fun.(type) {
			case *ast.SelectorExpr:
				if sel := info.Selections[fun]; sel != nil {
					obj = sel.Obj().(*types.Func)
				}
			case *ast.Ident:
				if o := info.Uses[fun]; o != nil {
					if f, ok := o.(*types.Func); ok {
						obj = f
					}
				}
			}
			if obj != nil {
				if p := obj.Pkg(); p != nil {
					ppath := p.Path()
					if strings.HasPrefix(ppath, modulePath) {
						inferFrom(ppath, obj.Name(), visited, codes)
					}
				}
			}

		case *ast.SelectorExpr:
			if id, ok := n.X.(*ast.Ident); ok && id.Name == "code" {
				if c, ok := code.Lookup(n.Sel.Name); ok {
					codes[n.Sel.Name] = c
				}
			}
		}
		return true
	})
}

// ResponseMapFromHandler builds a response map using error codes inferred from
// the handler's source. Extra error codes can be supplied for cases where
// runtime errors may occur that are not explicitly returned in the handler.
func ResponseMapFromHandler[T any](op string, successes []Success[T], handler any, extra ...*code.Code) map[string]*huma.Response {
	errs := InferErrorCodes(handler)
	if len(extra) > 0 {
		errs = append(errs, extra...)
	}
	return ResponseMap[T](op, successes, errs)
}
