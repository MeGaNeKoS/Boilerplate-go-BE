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

	"github.com/MeGaNeKoS/neoma/core"

	"project-template/infrastructure/dto/response"
	"project-template/pkg/code"
	"project-template/server/rest"
	"project-template/server/rest/dto/files"
)

// UploadOutput is the response type for upload endpoints.
type UploadOutput struct {
	Status int                                           `yaml:"-"`
	Body   *response.GenericResponse[files.UploadOutput] `json:"body"`
}

// FormFileOutput is the response type for multipart form endpoints.
type FormFileOutput struct {
	Status int                                         `yaml:"-"`
	Body   *response.GenericResponse[files.FormOutput] `json:"body"`
}

// DownloadOutput is the response type for file download endpoints.
type DownloadOutput struct {
	Status             int    `yaml:"-"`
	ContentDisposition string `header:"Content-Disposition"`
	ContentType        string `header:"Content-Type"`
	Body               func(ctx core.Context, api core.API)
}

// UploadFile accepts a file and returns a success message.
func UploadFile(_ context.Context, in *files.UploadInput) (*UploadOutput, error) {
	if in.Body.File.File == nil {
		return nil, code.ErrPayloadError
	}
	_, _ = io.Copy(io.Discard, in.Body.File)
	return &UploadOutput{
		Status: http.StatusOK,
		Body:   rest.SuccessEnvelope(http.StatusOK, files.UploadOutput{Message: "uploaded"}),
	}, nil
}

// DownloadFile streams a file from the "public" directory by name.
func DownloadFile(_ context.Context, in *files.DownloadInput) (*DownloadOutput, error) {
	path := filepath.Join("public", in.Name)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, code.ErrNotFound
		}
		return nil, code.ErrInternalServerError
	}
	mt := mime.TypeByExtension(filepath.Ext(in.Name))
	if mt == "" {
		mt = http.DetectContentType(data)
	}
	return &DownloadOutput{
		Status:             http.StatusOK,
		ContentDisposition: fmt.Sprintf("attachment; filename=%q", in.Name),
		ContentType:        mt,
		Body: func(ctx core.Context, _ core.API) {
			_, _ = ctx.BodyWriter().Write(data)
		},
	}, nil
}

// FormWithFile handles a multipart form with a file.
func FormWithFile(_ context.Context, in *files.FormInput) (*FormFileOutput, error) {
	if in.Body.Attachment.File == nil {
		return nil, code.ErrPayloadError
	}
	_, _ = io.Copy(io.Discard, in.Body.Attachment)
	return &FormFileOutput{
		Status: http.StatusOK,
		Body:   rest.SuccessEnvelope(http.StatusOK, files.FormOutput{Name: in.Body.Name}),
	}, nil
}
