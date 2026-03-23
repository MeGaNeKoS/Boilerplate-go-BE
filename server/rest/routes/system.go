package routes

import (
	"net/http"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/middleware"
	"github.com/MeGaNeKoS/neoma/neoma"

	"project-template/server/rest/handlers"
)

// SystemRouter registers system endpoints.
func SystemRouter(g *middleware.Group) {
	neoma.Register(g, core.Operation{
		OperationID:   "echo",
		Method:        http.MethodGet,
		Path:          "/echo",
		Summary:       "Echo",
		Hidden:        true,
		DefaultStatus: http.StatusOK,
	}, handlers.Echo)

	neoma.Register(g, core.Operation{
		OperationID:   "crash",
		Method:        http.MethodGet,
		Path:          "/crash",
		Summary:       "Crash",
		Hidden:        true,
		Description:   "Intentionally crashes the application to trigger a restart.",
		DefaultStatus: http.StatusInternalServerError,
	}, handlers.Crash)

	neoma.Register(g, core.Operation{
		OperationID:   "long",
		Method:        http.MethodGet,
		Path:          "/long",
		Summary:       "Long",
		Hidden:        true,
		DefaultStatus: http.StatusOK,
	}, handlers.Long)
}
