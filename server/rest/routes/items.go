package routes

import (
	"github.com/go-chi/chi/v5"
)

// ItemsRouter registers item endpoints under the provided router.
func ItemsRouter(r chi.Router) {
	for _, rt := range ItemRouteDefs {
		r.MethodFunc(rt.Method, rt.Pattern, rt.Handler)
	}
}
