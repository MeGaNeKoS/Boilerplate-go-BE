package handlers

import (
	"io"
	"net/http"

	"project-template/infrastructure/utils"
	"project-template/pkg/code"
)

// UploadFileHandler accepts a file upload and returns a success message.
func UploadFileHandler(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("file")
	if err != nil {
		sendResponse(w, utils.GenerateErrorResponse(code.ErrPayloadError, err.Error()))
		return
	}
	defer file.Close()
	_, _ = io.Copy(io.Discard, file)
	sendResponse(w, utils.GenerateSuccessResponse(struct {
		Message string `json:"message"`
	}{Message: "uploaded"}))
}

// DownloadFileHandler returns a small text file for download.
func DownloadFileHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=sample.txt")
	_, _ = w.Write([]byte("sample file"))
}

// FormWithFileHandler accepts a form field and an attached file.
func FormWithFileHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		sendResponse(w, utils.GenerateErrorResponse(code.ErrPayloadError, err.Error()))
		return
	}
	_, _, err := r.FormFile("attachment")
	if err != nil {
		sendResponse(w, utils.GenerateErrorResponse(code.ErrPayloadError, err.Error()))
		return
	}
	name := r.FormValue("name")
	sendResponse(w, utils.GenerateSuccessResponse(struct {
		Name string `json:"name"`
	}{Name: name}))
}
