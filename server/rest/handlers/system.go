package handlers

import (
	"context"
	"net/http"

	"github.com/MeGaNeKoS/neoma/core"

	"project-template/infrastructure/dto/response"
	"project-template/pkg/code"
	"project-template/server/rest"
	systemdto "project-template/server/rest/dto/system"
	"project-template/services/system"
)

var systemService = system.NewService()

// StringOutput is the response type for string-returning endpoints.
type StringOutput struct {
	Status int                                    `yaml:"-"`
	Body   *response.GenericResponse[string]      `json:"body"`
}

// AnyOutput is the response type for untyped endpoints.
type AnyOutput struct {
	Status int                                `yaml:"-"`
	Body   *response.GenericResponse[any]     `json:"body"`
}

// Echo returns an echo response.
func Echo(ctx context.Context, _ *core.Empty) (*StringOutput, error) {
	resp, errCode := systemService.Echo(ctx)
	if errCode != nil {
		return nil, errCode
	}
	str, ok := resp.(string)
	if !ok {
		return nil, code.ErrInternalServerError
	}
	return &StringOutput{
		Status: http.StatusOK,
		Body:   rest.SuccessEnvelope(http.StatusOK, str),
	}, nil
}

// Crash triggers a panic to test recovery.
func Crash(ctx context.Context, _ *core.Empty) (*AnyOutput, error) {
	systemService.Crash(ctx)
	return &AnyOutput{
		Status: http.StatusOK,
		Body:   rest.SuccessEnvelope[any](http.StatusOK, nil),
	}, nil
}

// Long waits for the specified time before responding.
func Long(ctx context.Context, in *systemdto.LongInput) (*StringOutput, error) {
	resp, errCode := systemService.Long(ctx, in.Sleep)
	if errCode != nil {
		return nil, errCode
	}
	pid, ok := resp.(string)
	if !ok {
		return nil, code.ErrInternalServerError
	}
	return &StringOutput{
		Status: http.StatusOK,
		Body:   rest.SuccessEnvelope(http.StatusOK, pid),
	}, nil
}
