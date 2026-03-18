package handlers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"project-template/infrastructure/dto/files"
	"project-template/infrastructure/dto/response"
	"project-template/pkg/code"
	restutils "project-template/server/rest/utils"
)

// UploadFile accepts a file and returns a success message.
func UploadFile(_ context.Context, in *files.UploadInput) (*restutils.Response[*response.GenericResponse[files.UploadOutput]], error) {
	data := in.RawBody.Data()
	if data == nil || !data.File.IsSet {
		return nil, restutils.NewError(code.ErrPayloadError)
	}
	_, _ = io.Copy(io.Discard, data.File)
	return restutils.SuccessResponse(http.StatusOK, files.UploadOutput{Message: "uploaded"}), nil
}

// DownloadFile streams a file from the "public" directory by name.
func DownloadFile(_ context.Context, in *files.DownloadInput) (*restutils.Response[[]byte], error) {
	path := filepath.Join("public", in.Name)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, restutils.NewError(code.ErrNotFound)
		}
		return nil, restutils.NewError(code.ErrInternalServerError)
	}
	mt := mime.TypeByExtension(filepath.Ext(in.Name))
	if mt == "" {
		mt = http.DetectContentType(data)
	}
	return restutils.NewResponse(http.StatusOK, data).
		Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", in.Name)).
		Header("Content-Type", mt), nil
}

// FormWithFile handles a multipart form with a file.
func FormWithFile(_ context.Context, in *files.FormInput) (*restutils.Response[*response.GenericResponse[files.FormOutput]], error) {
	data := in.RawBody.Data()
	if data == nil || !data.Attachment.IsSet {
		return nil, restutils.NewError(code.ErrPayloadError)
	}
	_, _ = io.Copy(io.Discard, data.Attachment)
	return restutils.SuccessResponse(http.StatusOK, files.FormOutput{Name: data.Name}), nil
}
