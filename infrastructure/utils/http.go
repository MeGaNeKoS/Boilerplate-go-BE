package utils

import (
	"fmt"
	"net/http"
	"project-template/infrastructure/config"
	"project-template/infrastructure/dto/response"
	"project-template/pkg/code"
)

type SuccessOptions struct {
	HTTPCode *int
	Message  *string
}

// GenerateErrorResponse builds a standardized error payload using the provided
// Code. An optional message overrides the default.
func GenerateErrorResponse(err code.Code, errMessage ...string) *response.HttpResponse {
	if len(errMessage) > 0 {
		err.Message = errMessage[0]
	}

	return &response.HttpResponse{
		HTTPCode: err.HTTPCode,
		RawResponsePayload: response.GenericResponse{
			BaseResponse: response.BaseResponse{
				StatusCode:    fmt.Sprintf("%s-ERR-%d", config.Cfg.AppName, err.InternalCode),
				ReturnMessage: err.Message,
			},
		},
	}
}

// GenerateSuccessResponse returns a generic success response containing the
// provided body. Options can override the HTTP status code and message.
func GenerateSuccessResponse(body interface{}, opts ...*SuccessOptions) *response.HttpResponse {
	httpCode := http.StatusOK
	msg := "Success"

	if len(opts) > 0 && opts[0] != nil {
		if opts[0].HTTPCode != nil {
			httpCode = *opts[0].HTTPCode
		}
		if opts[0].Message != nil {
			msg = *opts[0].Message
		}
	}

	return &response.HttpResponse{
		HTTPCode: httpCode,
		RawResponsePayload: response.GenericResponse{
			BaseResponse: response.BaseResponse{
				StatusCode:    fmt.Sprintf("%s-%d", config.Cfg.AppName, httpCode),
				ReturnMessage: msg,
			},
			Body: body,
		},
	}
}
