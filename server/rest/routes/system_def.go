package routes

import (
	"net/http"

	"project-template/pkg/code"
	"project-template/server/rest/handlers"
)

// SystemRouteDefs defines all /system endpoints.
var SystemRouteDefs = []RouteDef{
	{
		Method:      http.MethodGet,
		Pattern:     "/echo",
		Handler:     handlers.EchoHandler,
		Summary:     "Echo message",
		Description: "Return the same payload for connectivity testing.",
		OperationID: "systemEcho",
		Tag:         "system",
		Responses: []ResponseDef{
			{
				Model:       new(interface{}),
				Description: "Echoed payload",
			},
			{ErrCode: &code.ErrInternalServerError},
		},
	},
	{
		Method:      http.MethodGet,
		Pattern:     "/crash",
		Handler:     handlers.CrashHandler,
		Summary:     "Crash server",
		Description: "Force a crash to test recovery procedures.",
		OperationID: "systemCrash",
		Tag:         "system",
		Responses: []ResponseDef{
			{
				Model:       new(struct{}),
				Description: "Server will crash after responding",
			},
			{ErrCode: &code.ErrInternalServerError},
		},
	},
	{
		Method:      http.MethodGet,
		Pattern:     "/long",
		Handler:     handlers.LongHandler,
		Summary:     "Long operation",
		Description: "Demonstrate a long running request.",
		OperationID: "systemLong",
		Tag:         "system",
		Req: &ReqDef{
			Model: new(struct {
				Sleep int `query:"sleep"`
			}),
			Description: "Seconds to wait before responding",
		},
		Responses: []ResponseDef{
			{
				Model:       new(interface{}),
				Description: "PID of the process after delay",
			},
			{ErrCode: &code.ErrInternalServerError},
		},
	},
}
