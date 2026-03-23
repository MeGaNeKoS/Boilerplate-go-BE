package files

import "github.com/MeGaNeKoS/neoma/core"

// UploadInput holds a required file for upload operations.
type UploadInput struct {
	Body struct {
		File core.FormFile `form:"file" required:"true" doc:"File to upload"`
	}
}

// FormInput is used for multipart form submissions with a file.
type FormInput struct {
	Body struct {
		Name       string        `form:"name" example:"demo"`
		Attachment core.FormFile `form:"attachment"`
	}
}

// UploadOutput is returned after a successful upload.
type UploadOutput struct {
	Message string `json:"message" example:"uploaded"`
}

// FormOutput is returned after a successful multipart form submission.
type FormOutput struct {
	Name string `json:"name" example:"demo"`
}
