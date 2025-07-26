package routes

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestSystemRouter(t *testing.T) {
	r := chi.NewRouter()
	SystemRouter(r)
	var cnt int
	var routes []string
	err := chi.Walk(r, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		cnt++
		routes = append(routes, method+" "+route)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if cnt != 3 {
		t.Fatalf("routes %d", cnt)
	}
	expected := map[string]bool{"GET /echo": true, "GET /crash": true, "GET /long": true}
	for _, rt := range routes {
		if !expected[rt] {
			t.Fatalf("unexpected %s", rt)
		}
	}
}
