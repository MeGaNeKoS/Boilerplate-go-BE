package files

import "github.com/danielgtaylor/huma/v2"

// UploadForm represents multipart form data for uploads.
type UploadForm struct {
	File huma.FormFile `form:"file" required:"true" doc:"File to upload" contentType:"text/plain,application/octet-stream"`
}

// UploadInput holds a required file for upload operations.
type UploadInput struct {
	RawBody huma.MultipartFormFiles[UploadForm] `body:"" contentType:"multipart/form-data"`
}

// DownloadOutput is used for file downloads.
type DownloadOutput struct {
	Body []byte `contentType:"application/octet-stream"`
}

// FormBody represents multipart form data including an attachment.
type FormBody struct {
	Name       string        `form:"name" example:"\"demo\""`
	Attachment huma.FormFile `form:"attachment" contentType:"text/plain,application/octet-stream"`
}

// FormInput is used for multipart form submissions with a file.
type FormInput struct {
	RawBody huma.MultipartFormFiles[FormBody] `body:"" contentType:"multipart/form-data"`
}

// UploadOutput is returned after a successful upload.
type UploadOutput struct {
	Message string `json:"message" example:"\"uploaded\""`
}

// FormOutput is returned after a successful multipart form submission.
type FormOutput struct {
	Name string `json:"name" example:"\"demo\""`
}
