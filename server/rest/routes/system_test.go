package routes

import (
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

func TestSystemRouter(t *testing.T) {
	r := chi.NewRouter()
	cfg := huma.DefaultConfig("x", "1")
	cfg.CreateHooks = nil
	api := humachi.New(r, cfg)
	SystemRouter(huma.NewGroup(api, ""))
	got := map[string]bool{}
	err := chi.Walk(r, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		got[method+" "+route] = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range []string{"GET /echo", "GET /crash", "GET /long"} {
		if !got[e] {
			t.Fatalf("missing %s", e)
		}
	}
}
