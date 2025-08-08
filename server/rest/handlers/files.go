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
	"project-template/server/rest/handlers/resthuma"
)

// UploadFile accepts a file and returns a success message.
func UploadFile(ctx context.Context, in *files.UploadInput) (*resthuma.Response[*response.GenericResponse[files.UploadOutput]], error) {
	data := in.RawBody.Data()
	if data == nil || !data.File.IsSet {
		return nil, resthuma.NewError(code.ErrPayloadError)
	}
	_, _ = io.Copy(io.Discard, data.File)
	return resthuma.SuccessResponse(http.StatusOK, files.UploadOutput{Message: "uploaded"}), nil
}

// DownloadFile streams a file from the "public" directory by name.
func DownloadFile(ctx context.Context, in *files.DownloadInput) (*resthuma.Response[[]byte], error) {
	path := filepath.Join("public", in.Name)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, resthuma.NewError(code.ErrNotFound)
		}
		return nil, resthuma.NewError(code.ErrInternalServerError)
	}
	mt := mime.TypeByExtension(filepath.Ext(in.Name))
	if mt == "" {
		mt = http.DetectContentType(data)
	}
	return resthuma.NewResponse(http.StatusOK, data).
		Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", in.Name)).
		Header("Content-Type", mt), nil
}

// FormWithFile handles a multipart form with a file.
func FormWithFile(ctx context.Context, in *files.FormInput) (*resthuma.Response[*response.GenericResponse[files.FormOutput]], error) {
	data := in.RawBody.Data()
	if data == nil || !data.Attachment.IsSet {
		return nil, resthuma.NewError(code.ErrPayloadError)
	}
	_, _ = io.Copy(io.Discard, data.Attachment)
	return resthuma.SuccessResponse(http.StatusOK, files.FormOutput{Name: data.Name}), nil
}
