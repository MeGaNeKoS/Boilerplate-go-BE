package routes

import (
	"context"
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

func TestUseDefaultTag(t *testing.T) {
	r := chi.NewRouter()
	cfg := huma.DefaultConfig("x", "1")
	cfg.CreateHooks = nil
	api := humachi.New(r, cfg)
	g := huma.NewGroup(api, "/items")
	UseDefaultTag(g, "/items")
	// operation without explicit tags
	huma.Register(g, huma.Operation{
		OperationID: "noTag",
		Method:      http.MethodGet,
		Path:        "/no",
		Summary:     "no tag",
	}, func(ctx context.Context, in *struct{}) (*struct{}, error) { return &struct{}{}, nil })
	// operation with explicit tag
	huma.Register(g, huma.Operation{
		OperationID: "withTag",
		Method:      http.MethodGet,
		Path:        "/with",
		Summary:     "with tag",
		Tags:        []string{"custom"},
	}, func(ctx context.Context, in *struct{}) (*struct{}, error) { return &struct{}{}, nil })

	spec := api.OpenAPI()
	if tags := spec.Paths["/items/no"].Get.Tags; len(tags) != 1 || tags[0] != "items" {
		t.Fatalf("default tag not applied: %#v", tags)
	}
	if tags := spec.Paths["/items/with"].Get.Tags; len(tags) != 1 || tags[0] != "custom" {
		t.Fatalf("explicit tag overridden: %#v", tags)
	}
}

func TestUseDefaultTagEmptyPrefix(t *testing.T) {
	r := chi.NewRouter()
	cfg := huma.DefaultConfig("x", "1")
	cfg.CreateHooks = nil
	api := humachi.New(r, cfg)
	g := huma.NewGroup(api, "/")
	UseDefaultTag(g, "/")

	huma.Register(g, huma.Operation{
		OperationID: "noTag",
		Method:      http.MethodGet,
		Path:        "/no",
		Summary:     "no tag",
	}, func(ctx context.Context, in *struct{}) (*struct{}, error) { return &struct{}{}, nil })

	spec := api.OpenAPI()
	found := false
	for _, p := range spec.Paths {
		found = true
		if tags := p.Get.Tags; len(tags) != 0 {
			t.Fatalf("unexpected tags: %#v", tags)
		}
	}
	if !found {
		t.Fatal("operation not registered")
	}
}

func TestNewGroupSetsDefaultTag(t *testing.T) {
	r := chi.NewRouter()
	cfg := huma.DefaultConfig("x", "1")
	cfg.CreateHooks = nil
	api := humachi.New(r, cfg)
	parent := huma.NewGroup(api, "")
	g := NewGroup(parent, "/items")

	huma.Register(g, huma.Operation{
		OperationID: "noTag",
		Method:      http.MethodGet,
		Path:        "/test",
		Summary:     "no tag",
	}, func(ctx context.Context, in *struct{}) (*struct{}, error) { return &struct{}{}, nil })

	spec := api.OpenAPI()
	if tags := spec.Paths["/items/test"].Get.Tags; len(tags) != 1 || tags[0] != "items" {
		t.Fatalf("default tag not applied: %#v", tags)
	}
}

func TestNewGroupEmptyPrefix(t *testing.T) {
	r := chi.NewRouter()
	cfg := huma.DefaultConfig("x", "1")
	cfg.CreateHooks = nil
	api := humachi.New(r, cfg)
	parent := huma.NewGroup(api, "")
	g := NewGroup(parent, "")

	huma.Register(g, huma.Operation{
		OperationID: "emptyPrefix",
		Method:      http.MethodGet,
		Path:        "/ep",
		Summary:     "empty prefix",
	}, func(ctx context.Context, in *struct{}) (*struct{}, error) { return &struct{}{}, nil })

	spec := api.OpenAPI()
	if tags := spec.Paths["/ep"].Get.Tags; len(tags) != 0 {
		t.Fatalf("expected no tags for empty prefix, got %#v", tags)
	}
}

func TestUseSecurity(t *testing.T) {
	r := chi.NewRouter()
	cfg := huma.DefaultConfig("x", "1")
	cfg.CreateHooks = nil
	api := humachi.New(r, cfg)
	g := huma.NewGroup(api, "")
	UseSecurity(g, "bearer")
	// operation without security
	huma.Register(g, huma.Operation{
		OperationID: "noSec",
		Method:      http.MethodGet,
		Path:        "/no",
		Summary:     "no security",
	}, func(ctx context.Context, in *struct{}) (*struct{}, error) { return &struct{}{}, nil })
	// operation with explicit security
	huma.Register(g, huma.Operation{
		OperationID: "withSec",
		Method:      http.MethodGet,
		Path:        "/with",
		Summary:     "with security",
		Security:    []map[string][]string{{"custom": {}}},
	}, func(ctx context.Context, in *struct{}) (*struct{}, error) { return &struct{}{}, nil })

	spec := api.OpenAPI()
	if sec := spec.Paths["/no"].Get.Security; len(sec) != 1 || sec[0]["bearer"] == nil {
		t.Fatalf("security requirement not applied: %#v", sec)
	}
	if sec := spec.Paths["/with"].Get.Security; len(sec) != 1 || sec[0]["custom"] == nil {
		t.Fatalf("explicit security overridden: %#v", sec)
	}
}
