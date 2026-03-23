package routes

import (
	"net/http"
	"testing"

	neomachi "github.com/MeGaNeKoS/neoma/adapters/neomachi/v5"
	"github.com/MeGaNeKoS/neoma/middleware"
	"github.com/MeGaNeKoS/neoma/neoma"
	"github.com/go-chi/chi/v5"
)

func TestFilesRouter(t *testing.T) {
	r := chi.NewRouter()
	cfg := neoma.DefaultConfig("x", "1")
	cfg.CreateHooks = nil
	api := neomachi.New(r, cfg)
	FilesRouter(middleware.NewGroup(api, ""))
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
