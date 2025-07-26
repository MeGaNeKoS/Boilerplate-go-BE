package routes

import "github.com/go-chi/chi/v5"

// FilesRouter registers file endpoints under the provided router.
func FilesRouter(r chi.Router) {
	for _, rt := range FileRouteDefs {
		r.MethodFunc(rt.Method, rt.Pattern, rt.Handler)
	}
}
