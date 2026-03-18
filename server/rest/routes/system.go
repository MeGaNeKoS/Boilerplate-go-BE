package routes

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"project-template/pkg/code"
	"project-template/server/rest/handlers"
	restutils "project-template/server/rest/utils"
)

// SystemRouter registers system endpoints using Huma.
func SystemRouter(g *huma.Group) {
	restutils.Register(g, huma.Operation{
		OperationID:   "echo",
		Method:        http.MethodGet,
		Path:          "/echo",
		Summary:       "Echo",
		Tags:          []string{"system", restutils.InternalTag()},
		DefaultStatus: http.StatusOK,
		Responses: restutils.ResponseMapFromHandler("echo",
			restutils.Successes(
				restutils.NewSuccess(http.StatusOK, "OK", "echo"),
			),
			handlers.Echo,
		),
	}, handlers.Echo)

	restutils.Register(g, huma.Operation{
		OperationID:   "crash",
		Method:        http.MethodGet,
		Path:          "/crash",
		Summary:       "Crash",
		Tags:          []string{"system", restutils.InternalTag()},
		Description:   "Intentionally crashes the application to trigger a restart.",
		DefaultStatus: http.StatusInternalServerError,
		Responses: restutils.ResponseMapFromHandler("crash",
			restutils.NoSuccesses(),
			handlers.Crash,
			code.ErrInternalServerError,
		),
	}, handlers.Crash)

	restutils.Register(g, huma.Operation{
		OperationID:   "long",
		Method:        http.MethodGet,
		Path:          "/long",
		Summary:       "Long",
		Tags:          []string{"system", restutils.InternalTag()},
		DefaultStatus: http.StatusOK,
		Responses: restutils.ResponseMapFromHandler("long",
			restutils.Successes(
				restutils.NewSuccess(http.StatusOK, "Delayed", "1234"),
			),
			handlers.Long,
		),
	}, handlers.Long)
}
