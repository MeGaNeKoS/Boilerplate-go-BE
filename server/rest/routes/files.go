package routes

import (
	"net/http"

	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/middleware"
	"github.com/MeGaNeKoS/neoma/neoma"

	"project-template/server/rest/handlers"
)

// FilesRouter registers file endpoints.
func FilesRouter(g *middleware.Group) {
	neoma.Register(g, core.Operation{
		OperationID:   "uploadFile",
		Method:        http.MethodPost,
		Path:          "/upload",
		Summary:       "Upload file",
		DefaultStatus: http.StatusOK,
	}, handlers.UploadFile)

	neoma.Register(g, core.Operation{
		OperationID:   "downloadFile",
		Method:        http.MethodGet,
		Path:          "/download/{name}",
		Summary:       "Download file",
		DefaultStatus: http.StatusOK,
	}, handlers.DownloadFile)

	neoma.Register(g, core.Operation{
		OperationID:   "formWithFile",
		Method:        http.MethodPost,
		Path:          "/form",
		Summary:       "Multipart form",
		DefaultStatus: http.StatusOK,
	}, handlers.FormWithFile)
}
