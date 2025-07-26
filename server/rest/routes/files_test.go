package routes

import (
	"net/http"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestFilesRouter(t *testing.T) {
	r := chi.NewRouter()
	FilesRouter(r)
	var cnt int
	err := chi.Walk(r, func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		cnt++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if cnt != 3 {
		t.Fatalf("routes %d", cnt)
	}
}
