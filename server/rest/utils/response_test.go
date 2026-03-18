package utils_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"

	"project-template/infrastructure/config"
	dtoitem "project-template/infrastructure/dto/item"
	"project-template/infrastructure/dto/response"
	"project-template/pkg/code"
	"project-template/server/rest/handlers"
	restutils "project-template/server/rest/utils"
)

func withAppName(name string) func() {
	prev := config.Cfg
	config.Cfg = &config.Config{
		AppName: name,
		REST:    config.ListenerConfig{Host: "localhost", Port: "8080"},
		Server: config.ServerConfig{
			Endpoint:      config.EndpointConfig{Based: "/"},
			InternalCIDRs: []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "127.0.0.0/8"},
		},
		OpenAPI: config.OpenAPIConfig{
			Docs:    config.DocsConfig{Public: config.DocConfig{URL: "/docs/public", Schema: "/schema/v1"}},
			Servers: config.ServersConfig{Public: "https://localhost:8080"},
		},
	}
	restutils.SetPrivateCIDRs(config.Cfg.Server.InternalCIDRs)
	return func() { config.Cfg = prev }
}

func TestNewError(t *testing.T) {
	cleanup := withAppName("APP")
	defer cleanup()

	err := restutils.NewError(code.ErrItemNotFound)
	he, ok := err.(interface {
		GetStatus() int
		GetHeaders() http.Header
	})
	if !ok {
		t.Fatalf("returned error does not implement expected interface")
	}
	if he.GetStatus() != code.ErrItemNotFound.HTTPCode {
		t.Fatalf("expected status %d, got %d", code.ErrItemNotFound.HTTPCode, he.GetStatus())
	}
	if ct := he.GetHeaders().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("expected problem+json content type, got %s", ct)
	}
	expLink := "<https://localhost:8080/docs/public/schema/v1/ProblemDetail.json>; rel=\"describedby\"; type=\"application/schema+json\", <https://localhost:8080/docs/public/openapi.json>; rel=\"service-desc\""
	if link := he.GetHeaders().Get("Link"); link != expLink {
		t.Fatalf("schema link header %q", link)
	}
	if err.Error() != code.ErrItemNotFound.Message {
		t.Fatalf("unexpected error message: %s", err.Error())
	}

	err = restutils.NewError(code.ErrItemNotFound, "custom")
	if err.Error() != "custom" {
		t.Fatalf("expected overridden message, got %s", err.Error())
	}
}

func TestResponseHeadersAndBody(t *testing.T) {
	r := restutils.NewResponse(http.StatusCreated, map[string]string{"foo": "bar"})
	r.Header("X-Test", "123")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()
	ctx := humatest.NewContext(&huma.Operation{}, req, resp)

	r.Body(ctx)

	if resp.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, resp.Code)
	}
	if resp.Header().Get("X-Test") != "123" {
		t.Fatalf("header not set on response")
	}
	if resp.Header().Get("Content-Type") != "application/json" {
		t.Fatalf("expected default content type")
	}
	var body map[string]string
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["foo"] != "bar" {
		t.Fatalf("unexpected body: %#v", body)
	}
	if r.GetHeaders().Get("X-Test") != "123" {
		t.Fatalf("header not stored in struct")
	}
}

func TestSuccessAndNoContentResponses(t *testing.T) {
	cleanup := withAppName("APP")
	defer cleanup()

	s := restutils.SuccessResponse(http.StatusOK, map[string]int{"id": 1})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.10:1234"
	resp := httptest.NewRecorder()
	ctx := humatest.NewContext(&huma.Operation{}, req, resp)
	s.Body(ctx)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}
	var gr response.GenericResponse[map[string]int]
	if err := json.Unmarshal(resp.Body.Bytes(), &gr); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if gr.StatusCode != "APP-200" || gr.Body["id"] != 1 {
		t.Fatalf("unexpected envelope: %#v", gr)
	}
	if link := resp.Header().Get("Link"); link == "" || !strings.HasPrefix(link, "<https://localhost:8080/docs/public/schema/v1/") || !strings.Contains(link, "service-desc") {
		t.Fatalf("missing link header on response: %q", link)
	}

	nc := restutils.NoContentResponse()
	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "203.0.113.10:1234"
	resp2 := httptest.NewRecorder()
	ctx2 := humatest.NewContext(&huma.Operation{}, req2, resp2)
	nc.Body(ctx2)
	if resp2.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d", http.StatusNoContent, resp2.Code)
	}
	if strings.TrimSpace(resp2.Body.String()) != "{}" {
		t.Fatalf("unexpected body %q", resp2.Body.String())
	}
	if link := resp2.Header().Get("Link"); link != "" {
		t.Fatalf("unexpected link header: %q", link)
	}
}

func TestResponseMapAndRegister(t *testing.T) {
	cleanup := withAppName("APP")
	defer cleanup()

	successes := restutils.Successes(
		restutils.NewSuccess(http.StatusCreated, "created", dtoitem.Item{ID: 1, Name: "sample"}),
	)
	m := restutils.ResponseMap("op", successes, []*code.Code{code.ErrInternalServerError})
	if _, ok := m["201"]; !ok {
		t.Fatalf("missing success response")
	}
	if _, ok := m["500"]; !ok {
		t.Fatalf("missing error response")
	}
	if s := m["201"].Content["application/json"].Schema; s == nil || s.Ref == "" {
		t.Fatalf("expected schema ref for success response")
	}
	if s := m["500"].Content["application/problem+json"].Schema; s == nil || s.Ref == "" {
		t.Fatalf("expected schema ref for error response")
	}
	ref := m["201"].Content["application/json"].Schema.Ref
	name := strings.TrimPrefix(ref, "#/components/schemas/")
	if h := m["201"].Headers["Link"]; h == nil || h.Examples == nil {
		t.Fatalf("missing link header example: %#v", h)
	} else {
		v, _ := h.Examples["schema"].Value.(string)
		if !strings.Contains(v, "/docs/public/schema/v1/"+name) || !strings.Contains(v, "service-desc") {
			t.Fatalf("missing link header example: %q", v)
		}
	}
	ref = m["500"].Content["application/problem+json"].Schema.Ref
	name = strings.TrimPrefix(ref, "#/components/schemas/")
	if h := m["500"].Headers["Link"]; h == nil || h.Examples == nil {
		t.Fatalf("missing error link header example: %#v", h)
	} else {
		v, _ := h.Examples["schema"].Value.(string)
		if !strings.Contains(v, "/docs/public/schema/v1/"+name) || !strings.Contains(v, "service-desc") {
			t.Fatalf("missing error link header example: %q", v)
		}
	}

	_, api := humatest.New(t)
	restutils.RegisterExamples(api)
	restutils.RegisterSchemas(api)
	if len(api.OpenAPI().Components.Examples) == 0 {
		t.Fatalf("expected examples to be registered")
	}
	if api.OpenAPI().Components.Schemas == nil || len(api.OpenAPI().Components.Schemas.Map()) == 0 {
		t.Fatalf("expected schemas to be registered")
	}
}

func TestResponseSchemaInlineBase(t *testing.T) {
	cleanup := withAppName("APP")
	defer cleanup()

	succ := restutils.Successes(restutils.NewSuccess(http.StatusOK, "ok", dtoitem.Item{}))
	m := restutils.ResponseMap("op", succ, nil)
	_, api := humatest.New(t)
	restutils.RegisterSchemas(api)

	ref := m["200"].Content["application/json"].Schema.Ref
	name := strings.TrimPrefix(ref, "#/components/schemas/")
	s := api.OpenAPI().Components.Schemas.Map()[name]
	if s == nil {
		t.Fatalf("schema not registered")
	}
	if len(s.AllOf) != 2 {
		t.Fatalf("expected allOf with body and base, got %#v", s.AllOf)
	}
	meta := s.AllOf[0]
	if meta == nil || meta.Ref != "#/components/schemas/BaseResponse" {
		t.Fatalf("base response ref missing: %#v", meta)
	}
	body := s.AllOf[1]
	if body == nil {
		t.Fatalf("missing body schema")
	}
	bprop := body.Properties["body"]
	if bprop == nil || bprop.Ref != "#/components/schemas/Item" {
		t.Fatalf("body ref incorrect: %#v", bprop)
	}
}

func TestSchemaExampleAndURL(t *testing.T) {
	cleanup := withAppName("APP")
	defer cleanup()
	config.Cfg.OpenAPI.Docs = config.DocsConfig{
		Public:   config.DocConfig{URL: "/docs", Schema: "/schemas"},
		Internal: config.DocConfig{URL: "/docs", Schema: "/schemas/internal"},
	}

	succ := restutils.Successes(restutils.NewSuccess(http.StatusOK, "ok", dtoitem.Item{ID: 1, Name: "sample"}))
	restutils.ResponseMap("op", succ, nil)
	_, api := humatest.New(t, huma.Config{SchemasPath: "/schemas", OpenAPI: &huma.OpenAPI{}})
	restutils.RegisterSchemas(api)

	s := api.OpenAPI().Components.Schemas.Map()["Item"]
	if s == nil || len(s.Examples) == 0 {
		t.Fatalf("expected item schema example")
	}
	if p := s.Properties["$schema"]; p != nil {
		t.Fatalf("unexpected $schema property: %#v", p)
	}
}

func TestResponseMapFromHandler(t *testing.T) {
	cleanup := withAppName("APP")
	defer cleanup()

	succ := restutils.Successes(restutils.NewSuccess(http.StatusOK, "ok", dtoitem.Item{}))
	m := restutils.ResponseMapFromHandler("GetItem", succ, handlers.GetItem)
	if _, ok := m["500"]; !ok {
		t.Fatalf("expected inferred error response")
	}
	if _, ok := m["404"]; !ok {
		t.Fatalf("expected not found error response")
	}
}

func TestResponseMapBinarySuccess(t *testing.T) {
	cleanup := withAppName("APP")
	defer cleanup()

	succ := restutils.Successes(
		restutils.NewSuccess(http.StatusOK, "file", []byte("data"),
			restutils.WithContentType("application/octet-stream"),
			restutils.WithHeaders(map[string]*huma.Param{
				"Content-Disposition": {Schema: &huma.Schema{Type: "string"}},
			}),
		),
	)
	m := restutils.ResponseMapFromHandler("downloadFile", succ, handlers.DownloadFile)
	r, ok := m["200"]
	if !ok {
		t.Fatalf("missing success response")
	}
	if r.Content["application/octet-stream"] == nil {
		t.Fatalf("expected octet-stream content")
	}
	if _, ok := r.Headers["Content-Disposition"]; !ok {
		t.Fatalf("expected content-disposition header")
	}
}

func TestResponseExampleNaming(t *testing.T) {
	succ := restutils.Successes(restutils.NewSuccess(http.StatusCreated, "created", dtoitem.Item{}))
	m := restutils.ResponseMap("createItem", succ, nil)
	r, ok := m["201"]
	if !ok {
		t.Fatalf("missing success response")
	}
	ex := r.Content["application/json"].Examples["success"].Ref
	if !strings.Contains(ex, "createItem-Item-201") {
		t.Fatalf("unexpected example ref: %s", ex)
	}
	if ref := r.Content["application/json"].Schema.Ref; ref != "#/components/schemas/ItemResponse" {
		t.Fatalf("unexpected schema ref: %s", ref)
	}
}

func TestResponseExampleNamingListAndError(t *testing.T) {
	restutils.ResetExampleRegistry()
	succ := restutils.Successes(restutils.NewSuccess(http.StatusOK, "ok", []dtoitem.Item{}))
	m := restutils.ResponseMap("listItems", succ, []*code.Code{code.ErrNotFound})
	r, ok := m["200"]
	if !ok {
		t.Fatalf("missing success response")
	}
	ex := r.Content["application/json"].Examples["success"].Ref
	if !strings.Contains(ex, "listItems-ItemList-200") {
		t.Fatalf("unexpected example ref: %s", ex)
	}
	if ref := r.Content["application/json"].Schema.Ref; ref != "#/components/schemas/ItemListResponse" {
		t.Fatalf("unexpected schema ref: %s", ref)
	}
	er, ok := m["404"]
	if !ok {
		t.Fatalf("missing error response")
	}
	ref := er.Content["application/problem+json"].Examples["Resource not found"].Ref
	if !strings.Contains(ref, "error-15") {
		t.Fatalf("unexpected error example ref: %s", ref)
	}
}

func TestResponseErrorExampleDuplicateMessage(t *testing.T) {
	errs := []*code.Code{
		code.ErrPayloadError,
		code.ErrBadRequest.ReplaceMessage("Payload error"),
	}
	m := restutils.ResponseMap("dup", restutils.NoSuccesses(), errs)
	r, ok := m["400"]
	if !ok {
		t.Fatalf("missing error response")
	}
	ex := r.Content["application/problem+json"].Examples
	if ex["Payload error - 1"].Ref == "" {
		t.Fatalf("missing first payload error example")
	}
	if ex["Payload error - 3"].Ref == "" {
		t.Fatalf("missing second payload error example")
	}
	if !strings.Contains(ex["Payload error - 1"].Ref, "error-1") {
		t.Fatalf("unexpected ref for first: %s", ex["Payload error - 1"].Ref)
	}
	if !strings.Contains(ex["Payload error - 3"].Ref, "error-3") {
		t.Fatalf("unexpected ref for second: %s", ex["Payload error - 3"].Ref)
	}
}
func TestResponseMultiHeaderAndBytes(t *testing.T) {
	r := restutils.NewResponse(http.StatusOK, []byte("raw"))
	r.Header("X-Test", "a").Header("X-Test", "b")
	r.Header("Content-Type", "text/plain")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()
	ctx := humatest.NewContext(&huma.Operation{}, req, resp)

	r.Body(ctx)

	values := resp.Header()["X-Test"]
	if len(values) != 2 || values[0] != "a" || values[1] != "b" {
		t.Fatalf("expected appended headers, got %v", values)
	}
	if ct := resp.Header().Get("Content-Type"); ct != "text/plain" {
		t.Fatalf("expected content type preserved, got %s", ct)
	}
	if body := resp.Body.String(); body != "raw" {
		t.Fatalf("expected raw body, got %q", body)
	}
}

func TestResponseStringBody(t *testing.T) {
	r := restutils.NewResponse(http.StatusOK, "hello world")
	r.Header("Content-Type", "text/plain")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()
	ctx := humatest.NewContext(&huma.Operation{}, req, resp)

	r.Body(ctx)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	if ct := resp.Header().Get("Content-Type"); ct != "text/plain" {
		t.Fatalf("expected text/plain, got %s", ct)
	}
	if body := resp.Body.String(); body != "hello world" {
		t.Fatalf("expected raw string, got %q", body)
	}
}

func TestResponseReaderBody(t *testing.T) {
	r := restutils.NewResponse[io.Reader](http.StatusOK, strings.NewReader("streamed"))
	r.Header("Content-Type", "application/octet-stream")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()
	ctx := humatest.NewContext(&huma.Operation{}, req, resp)

	r.Body(ctx)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	if body := resp.Body.String(); body != "streamed" {
		t.Fatalf("expected streamed body, got %q", body)
	}
}

func TestResponseReaderCloserBody(t *testing.T) {
	closed := false
	rc := io.NopCloser(strings.NewReader("data"))
	orig := rc
	// Wrap to detect Close call
	r := restutils.NewResponse[io.Reader](http.StatusOK, &trackCloser{Reader: orig, closed: &closed})
	r.Header("Content-Type", "application/octet-stream")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()
	ctx := humatest.NewContext(&huma.Operation{}, req, resp)

	r.Body(ctx)

	if body := resp.Body.String(); body != "data" {
		t.Fatalf("expected body 'data', got %q", body)
	}
	if !closed {
		t.Fatal("expected reader to be closed after body write")
	}
}

type trackCloser struct {
	io.Reader
	closed *bool
}

func (tc *trackCloser) Close() error {
	*tc.closed = true
	return nil
}

func TestRewriteSchemaExamplesRemovesSchema(t *testing.T) {
	reg := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
	base := &huma.Schema{
		Type: huma.TypeObject,
		Properties: map[string]*huma.Schema{
			"statusCode":    {Type: huma.TypeString},
			"returnMessage": {Type: huma.TypeString},
		},
		Required: []string{"statusCode", "returnMessage"},
	}
	reg.Map()["Base"] = base
	reg.Map()["Resp"] = &huma.Schema{
		Type: huma.TypeObject,
		Properties: map[string]*huma.Schema{
			"$schema": {Type: huma.TypeString},
		},
		AllOf: []*huma.Schema{
			{Ref: "#/components/schemas/Base"},
			{Type: huma.TypeObject, Properties: map[string]*huma.Schema{
				"$schema": {Type: huma.TypeString},
				"body":    {Type: huma.TypeString},
			}, Required: []string{"body"}},
		},
	}
	spec := &huma.OpenAPI{Components: &huma.Components{Schemas: reg}}
	restutils.RewriteSchemaExamples(spec, "/schema")
	s := spec.Components.Schemas.Map()["Resp"]
	if s.Properties != nil {
		if _, ok := s.Properties["$schema"]; ok {
			t.Fatalf("root $schema property retained: %#v", s.Properties)
		}
	}
	if len(s.AllOf) != 2 {
		t.Fatalf("expected two allOf members: %#v", s.AllOf)
	}
	for i, sub := range s.AllOf {
		if sub == nil {
			t.Fatalf("nil allOf schema at %d", i)
		}
		if _, ok := sub.Properties["$schema"]; ok {
			t.Fatalf("unexpected $schema in allOf[%d]: %#v", i, sub.Properties)
		}
	}
}

func TestRewriteSchemaLinks(t *testing.T) {
	cleanup := withAppName("APP")
	defer cleanup()

	config.Cfg.OpenAPI.Docs.Internal = config.DocConfig{URL: "/docs/internal", Schema: "/schema"}
	config.Cfg.OpenAPI.Servers.Internal = "https://localhost:8080"

	succ := restutils.Successes(restutils.NewSuccess(http.StatusOK, "ok", dtoitem.Item{}))
	m := restutils.ResponseMap("op", succ, nil)
	spec := &huma.OpenAPI{Paths: map[string]*huma.PathItem{"/": {Get: &huma.Operation{Responses: m}}}}
	restutils.RewriteSchemaLinks(spec, true)
	ex := spec.Paths["/"].Get.Responses["200"].Headers["Link"].Examples["schema"].Value.(string)
	if !strings.Contains(ex, "/docs/internal/schema/") {
		t.Fatalf("expected internal link, got %q", ex)
	}
}

func TestInternalSchemaLink(t *testing.T) {
	cleanup := withAppName("APP")
	defer cleanup()

	config.Cfg.OpenAPI.Docs.Internal = config.DocConfig{URL: "/docs/internal", Schema: "/schema"}
	config.Cfg.OpenAPI.Servers.Internal = "https://localhost:8080"

	s := restutils.SuccessResponse(http.StatusOK, "ok")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	resp := httptest.NewRecorder()
	ctx := humatest.NewContext(&huma.Operation{}, req, resp)
	s.Body(ctx)
	if link := resp.Header().Get("Link"); !strings.Contains(link, "/docs/internal/schema/") {
		t.Fatalf("expected internal link, got %q", link)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/", nil)
	req2.RemoteAddr = "10.0.0.1:5678"
	resp2 := httptest.NewRecorder()
	ctx2 := humatest.NewContext(&huma.Operation{}, req2, resp2)
	_, _ = restutils.SchemaLinkTransformer(ctx2, strconv.Itoa(http.StatusBadRequest), &response.ProblemDetail{})
	if link := resp2.Header().Get("Link"); !strings.Contains(link, "/docs/internal/schema/") {
		t.Fatalf("expected internal error link, got %q", link)
	}
}

func TestInternalSchemaLinkForwardedFor(t *testing.T) {
	cleanup := withAppName("APP")
	defer cleanup()

	config.Cfg.OpenAPI.Docs.Internal = config.DocConfig{URL: "/docs/internal", Schema: "/schema"}
	config.Cfg.OpenAPI.Servers.Internal = "https://localhost:8080"

	s := restutils.SuccessResponse(http.StatusOK, "ok")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.1:1234"
	req.Header.Set("X-Forwarded-For", "10.0.0.5")
	resp := httptest.NewRecorder()
	ctx := humatest.NewContext(&huma.Operation{}, req, resp)
	s.Body(ctx)
	if link := resp.Header().Get("Link"); !strings.Contains(link, "/docs/internal/schema/") {
		t.Fatalf("expected internal link, got %q", link)
	}
}

func TestRewriteExampleNames(t *testing.T) {
	spec := &huma.OpenAPI{
		Components: &huma.Components{Examples: map[string]*huma.Example{
			"error-1": {Value: &response.ProblemDetail{Title: "Payload error", Type: "/errors/1", Status: 400}},
			"error-3": {Value: &response.ProblemDetail{Title: "Payload error", Type: "/errors/3", Status: 400}},
		}},
		Paths: map[string]*huma.PathItem{
			"/": {Get: &huma.Operation{Responses: map[string]*huma.Response{
				"400": {Content: map[string]*huma.MediaType{
					"application/problem+json": {
						Examples: map[string]*huma.Example{
							"payload-error-1": {Ref: "#/components/examples/error-1"},
							"payload-error-3": {Ref: "#/components/examples/error-3"},
						},
					},
				}},
			}}},
		},
	}
	restutils.RewriteExampleNames(spec)
	ex := spec.Paths["/"].Get.Responses["400"].Content["application/problem+json"].Examples
	if _, ok := ex["Payload error - 1"]; !ok {
		t.Fatalf("missing renamed example: %#v", ex)
	}
	if _, ok := ex["Payload error - 3"]; !ok {
		t.Fatalf("missing second renamed example: %#v", ex)
	}
	if _, ok := ex["payload-error-1"]; ok {
		t.Fatalf("old slug key present: %#v", ex)
	}
}
