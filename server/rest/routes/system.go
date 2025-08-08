package routes

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"project-template/pkg/code"
	"project-template/server/rest/handlers"
	"project-template/server/rest/handlers/resthuma"
	"project-template/server/rest/helpers"
)

// SystemRouter registers system endpoints using Huma.
func SystemRouter(g *huma.Group) {
	resthuma.Register(g, huma.Operation{
		OperationID:   "echo",
		Method:        http.MethodGet,
		Path:          "/echo",
		Summary:       "Echo",
		Tags:          []string{"system", helpers.InternalTag()},
		DefaultStatus: http.StatusOK,
		Responses: resthuma.ResponseMapFromHandler[string]("echo",
			[]resthuma.Success[string]{resthuma.NewSuccess(http.StatusOK, "OK", "echo")},
			handlers.Echo,
		),
	}, handlers.Echo)

	resthuma.Register(g, huma.Operation{
		OperationID:   "crash",
		Method:        http.MethodGet,
		Path:          "/crash",
		Summary:       "Crash",
		Tags:          []string{"system", helpers.InternalTag()},
		Description:   "Intentionally crashes the application to trigger a restart.",
		DefaultStatus: http.StatusInternalServerError,
		Responses: resthuma.ResponseMapFromHandler[any]("crash",
			nil,
			handlers.Crash,
			code.ErrInternalServerError,
		),
	}, handlers.Crash)

	resthuma.Register(g, huma.Operation{
		OperationID:   "long",
		Method:        http.MethodGet,
		Path:          "/long",
		Summary:       "Long",
		Tags:          []string{"system", helpers.InternalTag()},
		DefaultStatus: http.StatusOK,
		Responses: resthuma.ResponseMapFromHandler[string]("long",
			[]resthuma.Success[string]{resthuma.NewSuccess(http.StatusOK, "Delayed", "1234")},
			handlers.Long,
		),
	}, handlers.Long)
}
