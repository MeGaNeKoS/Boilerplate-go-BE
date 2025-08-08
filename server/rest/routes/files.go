package routes

import (
	"net/http"
	"reflect"

	"github.com/danielgtaylor/huma/v2"

	"project-template/infrastructure/dto/files"
	"project-template/server/rest/handlers"
	"project-template/server/rest/handlers/resthuma"
)

// FilesRouter registers file endpoints using Huma.
func FilesRouter(g *huma.Group) {
	resthuma.Register(g, huma.Operation{
		OperationID:   "uploadFile",
		Method:        http.MethodPost,
		Path:          "/upload",
		Summary:       "Upload file",
		DefaultStatus: http.StatusOK,
		Responses: resthuma.ResponseMapFromHandler[files.UploadOutput]("uploadFile",
			[]resthuma.Success[files.UploadOutput]{resthuma.NewSuccess(http.StatusOK, "Uploaded", files.UploadOutput{Message: "uploaded"})},
			handlers.UploadFile,
		),
	}, handlers.UploadFile)

	resthuma.Register(g, huma.Operation{
		OperationID:   "downloadFile",
		Method:        http.MethodGet,
		Path:          "/download/{name}",
		Summary:       "Download file",
		DefaultStatus: http.StatusOK,
		Responses: func() map[string]*huma.Response {
			errMap := resthuma.ResponseMapFromHandler[any]("downloadFile", nil, handlers.DownloadFile)
			return map[string]*huma.Response{
				"200": {
					Description: "File",
					Headers: map[string]*huma.Param{
						"Content-Disposition": {Schema: &huma.Schema{Type: "string"}},
					},
					Content: map[string]*huma.MediaType{
						"application/octet-stream": {
							Schema: huma.SchemaFromType(g.OpenAPI().Components.Schemas, reflect.TypeOf([]byte{})),
							Examples: map[string]*huma.Example{
								"file": {Value: []byte("sample file")},
							},
						},
					},
				},
				"404": errMap["404"],
				"500": errMap["500"],
			}
		}(),
	}, handlers.DownloadFile)

	resthuma.Register(g, huma.Operation{
		OperationID:   "formWithFile",
		Method:        http.MethodPost,
		Path:          "/form",
		Summary:       "Multipart form",
		DefaultStatus: http.StatusOK,
		Responses: resthuma.ResponseMapFromHandler[files.FormOutput]("formWithFile",
			[]resthuma.Success[files.FormOutput]{resthuma.NewSuccess(http.StatusOK, "OK", files.FormOutput{Name: "sample"})},
			handlers.FormWithFile,
		),
	}, handlers.FormWithFile)
}
