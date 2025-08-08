package handlers

import (
	"context"
	"net/http"

	"project-template/infrastructure/dto/response"
	systemdto "project-template/infrastructure/dto/system"
	"project-template/server/rest/handlers/resthuma"
	"project-template/services/system"
)

var systemService = system.NewService()

// Echo returns an echo response.
func Echo(ctx context.Context, _ *resthuma.Empty) (*resthuma.Response[*response.GenericResponse[string]], error) {
	resp, errCode := systemService.Echo(ctx)
	if errCode != nil {
		return nil, resthuma.NewError(errCode)
	}
	str, _ := resp.(string)
	return resthuma.SuccessResponse(http.StatusOK, str), nil
}

// Crash triggers a panic to test recovery.
func Crash(ctx context.Context, _ *resthuma.Empty) (*resthuma.Response[*response.GenericResponse[any]], error) {
	systemService.Crash(ctx)
	return resthuma.SuccessResponse[any](http.StatusOK, nil), nil
}

// Long waits for the specified time before responding.
func Long(ctx context.Context, in *systemdto.LongInput) (*resthuma.Response[*response.GenericResponse[string]], error) {
	resp, errCode := systemService.Long(ctx, in.Sleep)
	if errCode != nil {
		return nil, resthuma.NewError(errCode)
	}
	pid, _ := resp.(string)
	return resthuma.SuccessResponse(http.StatusOK, pid), nil
}
