package routes

import (
	"github.com/go-chi/chi/v5"
)

// SystemRouter registers system endpoints under the provided router.
func SystemRouter(r chi.Router) {
	for _, rt := range SystemRouteDefs {
		r.MethodFunc(rt.Method, rt.Pattern, rt.Handler)
	}
}
