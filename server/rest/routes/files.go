package routes

import (
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"project-template/infrastructure/dto/files"
	"project-template/server/rest/handlers"
	restutils "project-template/server/rest/utils"
)

// FilesRouter registers file endpoints using Huma.
func FilesRouter(g *huma.Group) {
	restutils.Register(g, huma.Operation{
		OperationID:   "uploadFile",
		Method:        http.MethodPost,
		Path:          "/upload",
		Summary:       "Upload file",
		DefaultStatus: http.StatusOK,
		Responses: restutils.ResponseMapFromHandler("uploadFile",
			restutils.Successes(
				restutils.NewSuccess(http.StatusOK, "Uploaded", files.UploadOutput{Message: "uploaded"}),
			),
			handlers.UploadFile,
		),
	}, handlers.UploadFile)

	restutils.Register(g, huma.Operation{
		OperationID:   "downloadFile",
		Method:        http.MethodGet,
		Path:          "/download/{name}",
		Summary:       "Download file",
		DefaultStatus: http.StatusOK,
		Responses: restutils.ResponseMapFromHandler("downloadFile",
			restutils.Successes(
				restutils.NewSuccess(http.StatusOK, "File", []byte("sample file"),
					restutils.WithContentType("application/octet-stream"),
					restutils.WithHeaders(map[string]*huma.Param{
						"Content-Disposition": {Schema: &huma.Schema{Type: "string"}},
					}),
				),
			),
			handlers.DownloadFile,
		),
	}, handlers.DownloadFile)

	restutils.Register(g, huma.Operation{
		OperationID:   "formWithFile",
		Method:        http.MethodPost,
		Path:          "/form",
		Summary:       "Multipart form",
		DefaultStatus: http.StatusOK,
		Responses: restutils.ResponseMapFromHandler("formWithFile",
			restutils.Successes(
				restutils.NewSuccess(http.StatusOK, "OK", files.FormOutput{Name: "sample"}),
			),
			handlers.FormWithFile,
		),
	}, handlers.FormWithFile)
}
