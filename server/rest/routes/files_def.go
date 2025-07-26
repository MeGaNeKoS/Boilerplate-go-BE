package routes

import (
	"mime/multipart"
	"net/http"

	"project-template/pkg/code"
	"project-template/server/rest/handlers"
)

// FileRouteDefs defines file upload/download endpoints used by router and OpenAPI.
var FileRouteDefs = []RouteDef{
	{
		Method:      http.MethodPost,
		Pattern:     "/upload",
		Handler:     handlers.UploadFileHandler,
		Summary:     "Upload file",
		Description: "Upload a single file",
		OperationID: "uploadFile",
		Tag:         "files",
		Req: &ReqDef{
			Model: new(struct {
				File multipart.File `formData:"file" required:"true"`
			}),
			ContentType: "multipart/form-data",
			Description: "File payload",
		},
		Responses: []ResponseDef{{
			Model: new(struct {
				Message string `json:"message"`
			}),
			Description: "Upload result",
		}, {ErrCode: &code.ErrPayloadError}, {ErrCode: &code.ErrInternalServerError}},
	},
	{
		Method:      http.MethodGet,
		Pattern:     "/download",
		Handler:     handlers.DownloadFileHandler,
		Summary:     "Download file",
		Description: "Return a sample file",
		OperationID: "downloadFile",
		Tag:         "files",
		Responses: []ResponseDef{{
			Model:       new([]byte),
			ContentType: "application/octet-stream",
			Description: "File content",
		}, {ErrCode: &code.ErrInternalServerError}},
	},
	{
		Method:      http.MethodPost,
		Pattern:     "/form",
		Handler:     handlers.FormWithFileHandler,
		Summary:     "Form upload",
		Description: "Form fields with file attachment",
		OperationID: "formUpload",
		Tag:         "files",
		Req: &ReqDef{
			Model: new(struct {
				Name       string                `formData:"name"`
				Attachment *multipart.FileHeader `formData:"attachment"`
			}),
			ContentType: "multipart/form-data",
			Description: "Form with file",
		},
		Responses: []ResponseDef{{
			Model: new(struct {
				Name string `json:"name"`
			}),
			Description: "Form result",
		}, {ErrCode: &code.ErrPayloadError}, {ErrCode: &code.ErrInternalServerError}},
	},
}
