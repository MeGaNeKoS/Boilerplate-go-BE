package utils

import (
	"fmt"

	"project-template/infrastructure/config"
	"project-template/infrastructure/dto/response"
	"project-template/pkg/code"
)

// GenerateErrorResponse builds a standardized error payload using the provided
// Code. An optional message overrides the default.
func GenerateErrorResponse(err *code.Code, errMessage ...string) *response.HttpResponse {
	e := *err
	if len(errMessage) > 0 {
		e.Message = errMessage[0]
	}

	app := ""
	if config.Cfg != nil {
		app = config.Cfg.AppName
	}

	pd := response.ProblemDetail{
		Type:   fmt.Sprintf("/errors/%d", e.InternalCode),
		Title:  e.Message,
		Status: e.HTTPCode,
		Code:   fmt.Sprintf("%s-ERR-%d", app, e.InternalCode),
	}

	return &response.HttpResponse{
		HTTPCode:           e.HTTPCode,
		ContentType:        "application/problem+json",
		RawResponsePayload: pd,
	}
}

// GenerateResponse builds a generic HttpResponse for successful cases.
// The contentType defaults to application/json when empty so callers can
// override it for endpoints that return other formats such as plain text or
// binary data.
func GenerateResponse(status int, payload interface{}, contentType string) *response.HttpResponse {
	ct := contentType
	if ct == "" {
		ct = "application/json"
	}
	return &response.HttpResponse{
		HTTPCode:           status,
		ContentType:        ct,
		RawResponsePayload: payload,
	}
}
