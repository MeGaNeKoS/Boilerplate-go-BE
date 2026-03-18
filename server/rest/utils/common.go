package utils

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/types"
	"net/http"
	"reflect"
	"runtime"
	"strconv"
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

// precomputedErrors holds error code names populated by the generator.
// It remains empty in source control so builds succeed even without running
// go generate.
var precomputedErrors = map[string][]string{}

// Empty is an empty input used when a route does not accept any parameters.
type Empty struct{}

// example registry for OpenAPI components
var (
	examples          = map[string]*huma.Example{}
	exampleCounter    = map[string]int{}
	exampleValueIndex = map[string]string{}
)

// ResetExampleRegistry clears the global example registry. Intended for use
// in tests to prevent state leaking between test cases.
func ResetExampleRegistry() {
	examples = map[string]*huma.Example{}
	exampleCounter = map[string]int{}
	exampleValueIndex = map[string]string{}
}

var registry = huma.NewMapRegistry("#/components/schemas/", schemaNamer)

func upperFirst(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func typeLabel(t reflect.Type) string {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Slice:
		if t.Elem().Kind() == reflect.Uint8 {
			return "Binary"
		}
		return typeLabel(t.Elem()) + "List"
	case reflect.Interface:
		return "Any"
	case reflect.Struct:
		if t.Name() != "" {
			return t.Name()
		}
		if t.NumField() == 0 {
			return "Any"
		}
		return "Struct"
	default:
		if t.Name() != "" {
			return upperFirst(t.Name())
		}
		return upperFirst(t.Kind().String())
	}
}

func schemaNamer(t reflect.Type, hint string) string {
	if t.PkgPath() == "project-template/infrastructure/dto/response" && strings.HasPrefix(t.Name(), "GenericResponse[") {
		if f, ok := t.FieldByName("Body"); ok {
			return typeLabel(f.Type) + "Response"
		}
	}
	return huma.DefaultSchemaNamer(t, hint)
}

// schema for BaseResponse without a body
var baseResponseSchema = func() *huma.Schema {
	s := registry.Schema(reflect.TypeOf(response.BaseResponse{}), true, "")
	// Allow composition with a body by omitting additionalProperties.
	s.AdditionalProperties = nil
	return s
}()

// schema for RFC 9457 problem details
var problemDetailSchema = func() *huma.Schema {
	ref := registry.Schema(reflect.TypeOf(response.ProblemDetail{}), true, "")
	if s := registry.SchemaFromRef(ref.Ref); s != nil {
		// Allow extension members as permitted by RFC 7807.
		s.AdditionalProperties = true

		if p, ok := s.Properties["type"]; ok {
			p.Type = huma.TypeString
			p.Format = "uri-reference"
			p.AnyOf = nil
		}
		if p, ok := s.Properties["instance"]; ok {
			p.Format = "uri-reference"
		}
		if p, ok := s.Properties["status"]; ok {
			p.Format = "int32"
		}
	}
	return ref
}()

// humaError adapts project error codes to huma.StatusError.
type humaError struct {
	status          int
	wwwAuthenticate string
	retryAfter      string
	Body            response.ProblemDetail
}

func (e *humaError) Error() string  { return e.Body.Title }
func (e *humaError) GetStatus() int { return e.status }
func (e *humaError) GetHeaders() http.Header {
	h := http.Header{}
	h.Set("Content-Type", "application/problem+json")
	if e.wwwAuthenticate != "" {
		h.Set("WWW-Authenticate", e.wwwAuthenticate)
	}
	if e.retryAfter != "" {
		h.Set("Retry-After", e.retryAfter)
	}
	if link := schemaLinkFromRef(nil, problemDetailSchema.Ref); link != "" {
		h.Set("Link", link)
	}
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

// NewAuthError converts a Code to an error that includes a WWW-Authenticate
// header.
func NewAuthError(c *code.Code, wwwAuth string, msg ...string) error {
	e := newHumaError(c, msg...)
	e.wwwAuthenticate = wwwAuth
	return e
}

// NewRetryableError converts a Code to an error that includes a Retry-After
// header.
func NewRetryableError(c *code.Code, retryAfterSeconds int, msg ...string) error {
	e := newHumaError(c, msg...)
	e.retryAfter = strconv.Itoa(retryAfterSeconds)
	return e
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
	r := NewResponse(status, success(body, status))
	grType := reflect.TypeOf(response.GenericResponse[T]{})
	schema := registry.Schema(grType, true, "")
	orig := r.Body
	r.Body = func(ctx huma.Context) {
		if link := schemaLinkFromRef(ctx, schema.Ref); link != "" {
			ctx.SetHeader("Link", link)
		}
		orig(ctx)
	}
	return r
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

// SuccessMeta stores optional metadata for a success response.
type SuccessMeta struct {
	ContentType string
	Headers     map[string]*huma.Param
}

// Success represents a successful response example and metadata.
type Success[T any] struct {
	Status      int
	Description string
	Body        T
	SuccessMeta
}

// SuccessOption configures optional fields on Success.
type SuccessOption func(*SuccessMeta)

// WithContentType sets the content type for the success response.
func WithContentType(ct string) SuccessOption {
	return func(m *SuccessMeta) { m.ContentType = ct }
}

// WithHeaders sets the headers for the success response.
func WithHeaders(h map[string]*huma.Param) SuccessOption {
	return func(m *SuccessMeta) { m.Headers = h }
}

// NewSuccess creates a Success value with the given status, description and body.
func NewSuccess[T any](status int, desc string, body T, opts ...SuccessOption) Success[T] {
	s := Success[T]{Status: status, Description: desc, Body: body, SuccessMeta: SuccessMeta{ContentType: "application/json"}}
	for _, opt := range opts {
		opt(&s.SuccessMeta)
	}
	return s
}

// Successes returns a slice of Success values, allowing type inference at the call site.
func Successes[T any](s ...Success[T]) []Success[T] { return s }

// NoSuccesses returns a nil slice for operations with no success responses.
func NoSuccesses() []Success[struct{}] { return nil }

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
	dst := api.OpenAPI().Components.Schemas
	for n, s := range registry.Map() {
		ref := "#/components/schemas/" + n
		if t := registry.TypeFromRef(ref); t != nil {
			dst.Schema(t, true, n)
		}
		dst.Map()[n] = s
	}
}

// ResponseMap builds a map of Huma responses with one or more success
// examples plus any number of error codes, using example names derived from
// the provided operation name and body type.
func ResponseMap[T any](op string, successes []Success[T], errs []*code.Code) map[string]*huma.Response {
	responses := map[string]*huma.Response{}
	t := reflect.TypeOf((*T)(nil)).Elem()
	isEmptyInterface := t.Kind() == reflect.Interface && t.NumMethod() == 0
	isEmptyStruct := t.Kind() == reflect.Struct && t.NumField() == 0
	typed := !isEmptyInterface && !isEmptyStruct
	var typedSchema *huma.Schema
	if typed {
		// Register a schema for GenericResponse[T] that composes the
		// BaseResponse metadata with the typed body using `allOf`.
		grType := reflect.TypeOf(response.GenericResponse[T]{})
		typedSchema = registry.Schema(grType, true, "")
		name := strings.TrimPrefix(typedSchema.Ref, "#/components/schemas/")

		base := &huma.Schema{Ref: baseResponseSchema.Ref}
		body := &huma.Schema{
			Type: huma.TypeObject,
			Properties: map[string]*huma.Schema{
				"body": registry.Schema(t, true, ""),
			},
			Required: []string{"body"},
		}

		allOf := []*huma.Schema{base, body}
		registry.Map()[name] = &huma.Schema{
			Type:       huma.TypeObject,
			Properties: map[string]*huma.Schema{},
			AllOf:      allOf,
		}
		// Ensure subsequent uses reference the component definition.
		typedSchema = &huma.Schema{Ref: typedSchema.Ref}
	}
	var bodyExample any
	if typed && len(successes) > 0 {
		bodyExample = successes[0].Body
	}
	for _, s := range successes {
		if s.Status == 0 {
			s.Status = http.StatusOK
		}
		ct := s.ContentType
		if ct == "" {
			ct = "application/json"
		}
		if s.Status == http.StatusNoContent {
			responses[fmt.Sprintf("%d", s.Status)] = &huma.Response{Description: s.Description, Headers: s.Headers}
			continue
		}
		if ct != "application/json" {
			st := reflect.TypeOf(s.Body)
			typeName := typeLabel(st)
			base := fmt.Sprintf("%s-%s-%d", op, typeName, s.Status)
			responses[fmt.Sprintf("%d", s.Status)] = &huma.Response{
				Description: s.Description,
				Headers:     s.Headers,
				Content: map[string]*huma.MediaType{
					ct: {
						Schema: registry.Schema(st, true, ""),
						Examples: map[string]*huma.Example{
							"success": registerExample(base, s.Body),
						},
					},
				},
			}
			continue
		}
		schema := typedSchema
		var example any
		if typed {
			example = success(s.Body, s.Status)
		} else {
			// treat as a BaseResponse without a body
			schema = baseResponseSchema
			ex := success[any](nil, s.Status)
			example = &response.BaseResponse{
				StatusCode:    ex.StatusCode,
				ReturnMessage: ex.ReturnMessage,
			}
		}
		typeName := typeLabel(t)
		base := fmt.Sprintf("%s-%s-%d", op, typeName, s.Status)
		r := &huma.Response{
			Description: s.Description,
			Headers:     s.Headers,
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
		addSchemaHeader(r, schema)
		responses[fmt.Sprintf("%d", s.Status)] = r
	}
	// count error messages to detect duplicates
	msgCounts := map[string]int{}
	for _, e := range errs {
		msgCounts[e.Message]++
	}

	for _, e := range errs {
		key := e.Message
		if msgCounts[e.Message] > 1 {
			key = fmt.Sprintf("%s - %d", e.Message, e.InternalCode)
		}
		comp := fmt.Sprintf("error-%d", e.InternalCode)
		codeKey := fmt.Sprintf("%d", e.HTTPCode)
		r, ok := responses[codeKey]
		if !ok {
			r = &huma.Response{
				Description: e.Message,
				Content: map[string]*huma.MediaType{
					"application/problem+json": {
						Schema:   problemDetailSchema,
						Examples: map[string]*huma.Example{},
					},
				},
			}
			addSchemaHeader(r, problemDetailSchema)
			responses[codeKey] = r
		}
		mt := r.Content["application/problem+json"]
		mt.Examples[key] = registerExample(comp, exampleError(e))
	}
	if typed && bodyExample != nil {
		bs := registry.Schema(t, true, "")
		if bs.Ref != "" {
			name := strings.TrimPrefix(bs.Ref, "#/components/schemas/")
			if schema := registry.Map()[name]; schema != nil && len(schema.Examples) == 0 {
				schema.Examples = []any{bodyExample}
			}
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
			inspectFunc(fd, pkg.TypesInfo, visited, codes)
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
func inspectFunc(decl *ast.FuncDecl, info *types.Info, visited map[string]bool, codes map[string]*code.Code) {
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
					if f, ok := sel.Obj().(*types.Func); ok {
						obj = f
					}
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
