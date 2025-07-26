package handlers

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"project-template/infrastructure/config"
	"project-template/infrastructure/dto/response"
)

func TestUploadFileHandler(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	fw, err := mw.CreateFormFile("file", "a.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = fw.Write([]byte("data"))
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/", &b)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	UploadFileHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("code %d", rec.Code)
	}
	var resp response.GenericResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	body := resp.Body.(map[string]interface{})
	if body["message"] != "uploaded" {
		t.Fatalf("body %#v", body)
	}
}

func TestUploadFileHandlerError(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	UploadFileHandler(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestDownloadFileHandler(t *testing.T) {
	rec := httptest.NewRecorder()
	DownloadFileHandler(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Header().Get("Content-Type") != "application/octet-stream" {
		t.Fatalf("content type %s", rec.Header().Get("Content-Type"))
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("sample file")) {
		t.Fatalf("body %s", rec.Body.String())
	}
}

func TestFormWithFileHandler(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	_ = mw.WriteField("name", "bob")
	fw, err := mw.CreateFormFile("attachment", "b.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = fw.Write([]byte("data"))
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/", &b)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	FormWithFileHandler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d", rec.Code)
	}
	var resp response.GenericResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	body := resp.Body.(map[string]interface{})
	if body["name"] != "bob" {
		t.Fatalf("body %#v", body)
	}
}

func TestFormWithFileHandlerError(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Content-Type", "multipart/form-data")
	rec := httptest.NewRecorder()
	FormWithFileHandler(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestFormWithFileHandlerMissingFile(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	_ = mw.WriteField("name", "a")
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/", &b)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	FormWithFileHandler(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code %d", rec.Code)
	}
}
