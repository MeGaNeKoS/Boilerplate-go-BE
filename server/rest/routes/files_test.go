package routes

import (
	"net/http"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	humachi "github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

func TestFilesRouter(t *testing.T) {
	r := chi.NewRouter()
	api := humachi.New(r, huma.DefaultConfig("x", "1"))
	FilesRouter(huma.NewGroup(api, ""))
	got := map[string]bool{}
	err := chi.Walk(r, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		got[method+" "+route] = true
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range []string{"POST /upload", "GET /download/{name}", "POST /form"} {
		if !got[e] {
			t.Fatalf("missing %s", e)
		}
	}
}
