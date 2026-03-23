package rest

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"

	"github.com/MeGaNeKoS/neoma/core"

	"project-template/infrastructure/dto/response"
)

var appName string

// Init sets the app name used in response envelopes. Must be called
// before any handlers are registered.
func Init(name string) { appName = name }

func statusCode(status int) string {
	return fmt.Sprintf("%s-%d", appName, status)
}

// SuccessEnvelope wraps a body in the standard GenericResponse envelope.
func SuccessEnvelope[T any](status int, body T) *response.GenericResponse[T] {
	if status == 0 {
		status = http.StatusOK
	}
	return &response.GenericResponse[T]{
		BaseResponse: response.BaseResponse{
			StatusCode:    statusCode(status),
			ReturnMessage: "Success",
		},
		Body: body,
	}
}

// ErrorEnvelope wraps an error in RFC 9457 Problem Details format.
func ErrorEnvelope(status int, msg string) *response.ErrorResponse {
	return response.NewErrorResponse(status, msg)
}

// ErrorHandler implements core.ErrorHandler using RFC 9457 Problem Details.
type ErrorHandler struct{}

func NewErrorHandler() *ErrorHandler { return &ErrorHandler{} }

func (h *ErrorHandler) NewError(status int, msg string, errs ...error) core.Error {
	resp := response.NewErrorResponse(status, msg)
	for _, err := range errs {
		if err == nil {
			continue
		}
		var detail core.ErrorDetailer
		if errors.As(err, &detail) {
			d := detail.ErrorDetail()
			resp.Errors = append(resp.Errors, &response.ErrorDetail{
				Location: d.Location,
				Message:  d.Message,
				Value:    d.Value,
			})
		} else {
			resp.Errors = append(resp.Errors, &response.ErrorDetail{
				Message: err.Error(),
			})
		}
	}
	return resp
}

func (h *ErrorHandler) NewErrorWithContext(ctx core.Context, status int, msg string, errs ...error) core.Error {
	resp, _ := h.NewError(status, msg, errs...).(*response.ErrorResponse)
	if ctx != nil {
		if host := ctx.Host(); host != "" {
			resp.Instance = ctx.URL().Path
		}
	}
	return resp
}

func (h *ErrorHandler) ErrorSchema(registry core.Registry) *core.Schema {
	return registry.Schema(reflect.TypeOf(response.ErrorResponse{}), true, "")
}

func (h *ErrorHandler) ErrorContentType(ct string) string {
	if ct == "application/json" {
		return "application/problem+json"
	}
	return ct
}
