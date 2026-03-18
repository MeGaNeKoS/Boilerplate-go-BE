package handlers

import (
	"context"
	"net/http"

	"project-template/infrastructure/dto/response"
	systemdto "project-template/infrastructure/dto/system"
	"project-template/pkg/code"
	restutils "project-template/server/rest/utils"
	"project-template/services/system"
)

var systemService = system.NewService()

// Echo returns an echo response.
func Echo(ctx context.Context, _ *restutils.Empty) (*restutils.Response[*response.GenericResponse[string]], error) {
	resp, errCode := systemService.Echo(ctx)
	if errCode != nil {
		return nil, restutils.NewError(errCode)
	}
	str, ok := resp.(string)
	if !ok {
		return nil, restutils.NewError(code.ErrInternalServerError)
	}
	return restutils.SuccessResponse(http.StatusOK, str), nil
}

// Crash triggers a panic to test recovery.
func Crash(ctx context.Context, _ *restutils.Empty) (*restutils.Response[*response.GenericResponse[any]], error) {
	systemService.Crash(ctx)
	return restutils.SuccessResponse[any](http.StatusOK, nil), nil
}

// Long waits for the specified time before responding.
func Long(ctx context.Context, in *systemdto.LongInput) (*restutils.Response[*response.GenericResponse[string]], error) {
	resp, errCode := systemService.Long(ctx, in.Sleep)
	if errCode != nil {
		return nil, restutils.NewError(errCode)
	}
	pid, ok := resp.(string)
	if !ok {
		return nil, restutils.NewError(code.ErrInternalServerError)
	}
	return restutils.SuccessResponse(http.StatusOK, pid), nil
}
