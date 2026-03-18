package routes

import (
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/danielgtaylor/huma/v2/humatest"
	"github.com/go-chi/chi/v5"

	"project-template/infrastructure/config"
)

func TestItemsRouter(t *testing.T) {
	r := chi.NewRouter()
	cfg := huma.DefaultConfig("x", "1")
	cfg.CreateHooks = nil
	api := humachi.New(r, cfg)
	ItemsRouter(huma.NewGroup(api, "/items"))
	got := map[string]bool{}
	err := chi.Walk(r, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		got[method+" "+route] = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range []string{"GET /items", "POST /items", "GET /items/{id}", "PUT /items/{id}", "DELETE /items/{id}"} {
		if !got[e] {
			t.Fatalf("missing %s", e)
		}
	}
}

func TestItemsRouterValidation(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	_, api := humatest.New(t)
	ItemsRouter(huma.NewGroup(api, "/items"))
	spec := api.OpenAPI()
	op := spec.Paths["/items"].Get
	var limit *huma.Param
	for _, p := range op.Parameters {
		if p.Name == "limit" && p.In == "query" {
			limit = p
		}
	}
	if limit == nil || limit.Schema == nil || limit.Schema.Maximum == nil || *limit.Schema.Maximum != 100 {
		t.Fatalf("limit maximum not set: %#v", limit)
	}
	itemSchema := spec.Components.Schemas.Map()["Item"]
	if itemSchema == nil {
		t.Fatalf("item schema missing")
	}
	name := itemSchema.Properties["name"]
	if name == nil || len(name.Enum) != 2 || name.Enum[0] != "sample" || name.Enum[1] != "example" {
		t.Fatalf("name enum incorrect: %#v", name)
	}
}
