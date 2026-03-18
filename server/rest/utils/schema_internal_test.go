package utils

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"

	"project-template/infrastructure/config"
	"project-template/infrastructure/dto/response"
)

// --- RewriteSchemaExamples ---

func TestRewriteSchemaExamplesNil(t *testing.T) {
	// nil OpenAPI
	RewriteSchemaExamples(nil, "")
	// nil Components
	RewriteSchemaExamples(&huma.OpenAPI{}, "")
	// nil Schemas
	RewriteSchemaExamples(&huma.OpenAPI{Components: &huma.Components{}}, "")
}

func TestRewriteSchemaExamplesNilSchema(t *testing.T) {
	reg := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
	reg.Map()["Nil"] = nil
	spec := &huma.OpenAPI{Components: &huma.Components{Schemas: reg}}
	RewriteSchemaExamples(spec, "")
}

// --- RewriteSchemaLinks ---

func TestRewriteSchemaLinksNil(t *testing.T) {
	// nil OpenAPI
	RewriteSchemaLinks(nil, false)
	// nil Paths
	RewriteSchemaLinks(&huma.OpenAPI{}, false)
}

func TestRewriteSchemaLinksNilOperation(t *testing.T) {
	spec := &huma.OpenAPI{Paths: map[string]*huma.PathItem{
		"/": {Get: nil, Post: nil},
	}}
	RewriteSchemaLinks(spec, false)
}

func TestRewriteSchemaLinksNilResponse(t *testing.T) {
	spec := &huma.OpenAPI{Paths: map[string]*huma.PathItem{
		"/": {Get: &huma.Operation{Responses: map[string]*huma.Response{
			"200": nil,
		}}},
	}}
	RewriteSchemaLinks(spec, false)
}

func TestRewriteSchemaLinksNoHeaders(t *testing.T) {
	spec := &huma.OpenAPI{Paths: map[string]*huma.PathItem{
		"/": {Get: &huma.Operation{Responses: map[string]*huma.Response{
			"200": {Description: "ok"},
		}}},
	}}
	RewriteSchemaLinks(spec, false)
}

func TestRewriteSchemaLinksNoLinkHeader(t *testing.T) {
	spec := &huma.OpenAPI{Paths: map[string]*huma.PathItem{
		"/": {Get: &huma.Operation{Responses: map[string]*huma.Response{
			"200": {Headers: map[string]*huma.Param{"Other": {}}},
		}}},
	}}
	RewriteSchemaLinks(spec, false)
}

func TestRewriteSchemaLinksEmptyRef(t *testing.T) {
	spec := &huma.OpenAPI{Paths: map[string]*huma.PathItem{
		"/": {Get: &huma.Operation{Responses: map[string]*huma.Response{
			"200": {
				Headers: map[string]*huma.Param{
					"Link": {Examples: map[string]*huma.Example{"schema": {Value: "old"}}},
				},
				Content: map[string]*huma.MediaType{
					"application/json": {Schema: &huma.Schema{Ref: ""}},
				},
			},
		}}},
	}}
	RewriteSchemaLinks(spec, false)
}

func TestRewriteSchemaLinksNilContentSchema(t *testing.T) {
	spec := &huma.OpenAPI{Paths: map[string]*huma.PathItem{
		"/": {Get: &huma.Operation{Responses: map[string]*huma.Response{
			"200": {
				Headers: map[string]*huma.Param{
					"Link": {Examples: map[string]*huma.Example{"schema": {Value: "old"}}},
				},
				Content: map[string]*huma.MediaType{
					"application/json": {Schema: nil},
				},
			},
		}}},
	}}
	RewriteSchemaLinks(spec, false)
}

// --- RewriteExampleNames ---

func TestRewriteExampleNamesNil(t *testing.T) {
	// nil OpenAPI
	RewriteExampleNames(nil)
	// nil Paths
	RewriteExampleNames(&huma.OpenAPI{Components: &huma.Components{Examples: map[string]*huma.Example{}}})
	// nil Components
	RewriteExampleNames(&huma.OpenAPI{Paths: map[string]*huma.PathItem{}})
	// nil Examples
	RewriteExampleNames(&huma.OpenAPI{
		Components: &huma.Components{},
		Paths:      map[string]*huma.PathItem{},
	})
}

func TestRewriteExampleNamesNilExampleValue(t *testing.T) {
	spec := &huma.OpenAPI{
		Components: &huma.Components{Examples: map[string]*huma.Example{
			"e1": nil,
			"e2": {Value: nil},
		}},
		Paths: map[string]*huma.PathItem{
			"/": {Get: &huma.Operation{Responses: map[string]*huma.Response{
				"400": {Content: map[string]*huma.MediaType{
					"application/problem+json": {Examples: map[string]*huma.Example{
						"x": {Ref: "#/components/examples/e1"},
					}},
				}},
			}}},
		},
	}
	RewriteExampleNames(spec)
}

func TestRewriteExampleNamesNilOperationAndContent(t *testing.T) {
	spec := &huma.OpenAPI{
		Components: &huma.Components{Examples: map[string]*huma.Example{
			"e1": {Value: &response.ProblemDetail{Title: "T", Type: "/errors/1"}},
		}},
		Paths: map[string]*huma.PathItem{
			"/a": {Get: nil},
			"/b": {Get: &huma.Operation{Responses: map[string]*huma.Response{
				"400": nil,
			}}},
			"/c": {Get: &huma.Operation{Responses: map[string]*huma.Response{
				"400": {Content: nil},
			}}},
			"/d": {Get: &huma.Operation{Responses: map[string]*huma.Response{
				"400": {Content: map[string]*huma.MediaType{
					"text/plain": nil,
				}},
			}}},
			"/e": {Get: &huma.Operation{Responses: map[string]*huma.Response{
				"400": {Content: map[string]*huma.MediaType{
					"text/plain": {Examples: map[string]*huma.Example{}},
				}},
			}}},
		},
	}
	RewriteExampleNames(spec)
}

func TestRewriteExampleNamesNonProblemDetail(t *testing.T) {
	// An example whose value is not a *ProblemDetail should keep its original key.
	spec := &huma.OpenAPI{
		Components: &huma.Components{Examples: map[string]*huma.Example{
			"success": {Value: map[string]string{"ok": "yes"}},
		}},
		Paths: map[string]*huma.PathItem{
			"/": {Get: &huma.Operation{Responses: map[string]*huma.Response{
				"200": {Content: map[string]*huma.MediaType{
					"application/json": {Examples: map[string]*huma.Example{
						"success": {Ref: "#/components/examples/success"},
					}},
				}},
			}}},
		},
	}
	RewriteExampleNames(spec)
	ex := spec.Paths["/"].Get.Responses["200"].Content["application/json"].Examples
	if _, ok := ex["success"]; !ok {
		t.Fatalf("expected original key preserved for non-ProblemDetail: %#v", ex)
	}
}

// --- isPrivate ---

func TestIsPrivateNilCtx(t *testing.T) {
	if isPrivate(nil) {
		t.Fatal("expected false for nil context")
	}
}

func TestIsPrivateXRealIP(t *testing.T) {
	prev := config.Cfg
	config.Cfg = &config.Config{
		Server: config.ServerConfig{InternalCIDRs: []string{"10.0.0.0/8"}},
	}
	SetPrivateCIDRs(config.Cfg.Server.InternalCIDRs)
	defer func() { config.Cfg = prev }()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.10:1234"
	req.Header.Set("X-Real-IP", "10.0.0.5")
	resp := httptest.NewRecorder()
	ctx := humatest.NewContext(&huma.Operation{}, req, resp)
	if !isPrivate(ctx) {
		t.Fatal("expected private from X-Real-IP")
	}
}

func TestIsPrivateRemoteAddr(t *testing.T) {
	prev := config.Cfg
	config.Cfg = &config.Config{
		Server: config.ServerConfig{InternalCIDRs: []string{"10.0.0.0/8"}},
	}
	SetPrivateCIDRs(config.Cfg.Server.InternalCIDRs)
	defer func() { config.Cfg = prev }()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.1:5678"
	resp := httptest.NewRecorder()
	ctx := humatest.NewContext(&huma.Operation{}, req, resp)
	if !isPrivate(ctx) {
		t.Fatal("expected private from RemoteAddr")
	}
}

func TestIsPrivateUnparseableIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "not-an-ip"
	resp := httptest.NewRecorder()
	ctx := humatest.NewContext(&huma.Operation{}, req, resp)
	if isPrivate(ctx) {
		t.Fatal("expected false for unparseable IP")
	}
}

func TestIsPrivateForwardedForMultiple(t *testing.T) {
	prev := config.Cfg
	config.Cfg = &config.Config{
		Server: config.ServerConfig{InternalCIDRs: []string{"10.0.0.0/8"}},
	}
	SetPrivateCIDRs(config.Cfg.Server.InternalCIDRs)
	defer func() { config.Cfg = prev }()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.10:1234"
	req.Header.Set("X-Forwarded-For", "10.0.0.5, 203.0.113.10")
	resp := httptest.NewRecorder()
	ctx := humatest.NewContext(&huma.Operation{}, req, resp)
	if !isPrivate(ctx) {
		t.Fatal("expected private from first X-Forwarded-For entry")
	}
}

// --- buildSchemaLink ---

func TestBuildSchemaLinkEmptyRef(t *testing.T) {
	if got := buildSchemaLink("", false); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestBuildSchemaLinkInternalConfig(t *testing.T) {
	prev := config.Cfg
	config.Cfg = &config.Config{
		OpenAPI: config.OpenAPIConfig{
			Servers: config.ServersConfig{Internal: "https://internal:8080"},
			Docs: config.DocsConfig{
				Internal: config.DocConfig{URL: "/docs/internal", Schema: "/schema/v2"},
			},
		},
	}
	defer func() { config.Cfg = prev }()

	got := buildSchemaLink("#/components/schemas/Foo", true)
	if got == "" {
		t.Fatal("expected non-empty link")
	}
	if !contains(got, "/docs/internal/schema/v2/Foo.json") {
		t.Fatalf("expected internal schema path, got %q", got)
	}
	if !contains(got, "https://internal:8080") {
		t.Fatalf("expected internal base URL, got %q", got)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsSub(s, sub))
}

func containsSub(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// --- SchemaLinkTransformer ---

func TestSchemaLinkTransformerNonNumericStatus(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()
	ctx := humatest.NewContext(&huma.Operation{}, req, resp)

	v, err := SchemaLinkTransformer(ctx, "abc", nil)
	if err != nil || v != nil {
		t.Fatalf("expected passthrough for non-numeric status, got %v %v", v, err)
	}
}

func TestSchemaLinkTransformerSetsInstance(t *testing.T) {
	prev := config.Cfg
	config.Cfg = &config.Config{
		OpenAPI: config.OpenAPIConfig{
			Servers: config.ServersConfig{Public: "https://pub"},
			Docs:    config.DocsConfig{Public: config.DocConfig{URL: "/docs", Schema: "/schema"}},
		},
	}
	defer func() { config.Cfg = prev }()

	req := httptest.NewRequest(http.MethodGet, "/items/42", nil)
	req.RemoteAddr = "203.0.113.10:1234"
	resp := httptest.NewRecorder()
	ctx := humatest.NewContext(&huma.Operation{}, req, resp)

	m := map[string]any{"type": "/errors/1", "title": "bad", "status": float64(400)}
	v, err := SchemaLinkTransformer(ctx, "400", m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := v.(map[string]any)
	if out["instance"] != "/items/42" {
		t.Fatalf("expected instance to be set, got %v", out["instance"])
	}
}

func TestSchemaLinkTransformerPreservesExistingInstance(t *testing.T) {
	prev := config.Cfg
	config.Cfg = &config.Config{
		OpenAPI: config.OpenAPIConfig{
			Servers: config.ServersConfig{Public: "https://pub"},
			Docs:    config.DocsConfig{Public: config.DocConfig{URL: "/docs", Schema: "/schema"}},
		},
	}
	defer func() { config.Cfg = prev }()

	req := httptest.NewRequest(http.MethodGet, "/items/42", nil)
	req.RemoteAddr = "203.0.113.10:1234"
	resp := httptest.NewRecorder()
	ctx := humatest.NewContext(&huma.Operation{}, req, resp)

	m := map[string]any{"instance": "/already-set"}
	v, err := SchemaLinkTransformer(ctx, "500", m)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := v.(map[string]any)
	if out["instance"] != "/already-set" {
		t.Fatalf("expected existing instance preserved, got %v", out["instance"])
	}
}

// --- addSchemaHeader ---

func TestAddSchemaHeaderNilSchema(t *testing.T) {
	r := &huma.Response{Description: "test"}
	addSchemaHeader(r, nil)
	if r.Headers != nil {
		t.Fatal("expected no headers for nil schema")
	}
}

func TestAddSchemaHeaderEmptyRef(t *testing.T) {
	r := &huma.Response{Description: "test"}
	addSchemaHeader(r, &huma.Schema{Ref: ""})
	if r.Headers != nil {
		t.Fatal("expected no headers for empty ref")
	}
}

func TestBuildSchemaLinkFallbackToRESTHost(t *testing.T) {
	prev := config.Cfg
	config.Cfg = &config.Config{
		REST: config.ListenerConfig{Host: "myhost", Port: "9090"},
		OpenAPI: config.OpenAPIConfig{
			Servers: config.ServersConfig{Public: ""},
			Docs:    config.DocsConfig{Public: config.DocConfig{URL: "/docs", Schema: "/s"}},
		},
	}
	defer func() { config.Cfg = prev }()

	got := buildSchemaLink("#/components/schemas/X", false)
	if !containsSub(got, "//myhost:9090") {
		t.Fatalf("expected REST host fallback, got %q", got)
	}
}

func TestRewriteSchemaLinksLinkEmptyFromBuild(t *testing.T) {
	// This tests the path where buildSchemaLink returns "" because ref is empty.
	// The only way to hit line 65 is to have ref be empty after the loop,
	// which means all content types have nil or empty schema refs.
	// We already skip when ref is empty at line 61 so this is defensive.
	// We can still exercise it by having Content with nil MediaType.
	spec := &huma.OpenAPI{Paths: map[string]*huma.PathItem{
		"/": {Get: &huma.Operation{Responses: map[string]*huma.Response{
			"200": {
				Headers: map[string]*huma.Param{
					"Link": {Examples: map[string]*huma.Example{"schema": {Value: "old"}}},
				},
				Content: map[string]*huma.MediaType{
					"application/json": nil,
				},
			},
		}}},
	}}
	RewriteSchemaLinks(spec, false)
}
