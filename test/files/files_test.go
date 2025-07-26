//go:build !grpc && !kafka

package files_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"unsafe"

	"project-template/cmd"
	"project-template/infrastructure/config"
	"project-template/infrastructure/dto/response"
	"project-template/test/testutil"
)

func setupServer(t *testing.T) *http.Server {
	t.Helper()
	config.Cfg = &config.Config{
		AppName:   "APP",
		Server:    config.ServerConfig{Endpoint: config.EndpointConfig{Based: "/api"}, Timeout: config.TimeoutConfig{Server: 1}},
		REST:      config.ListenerConfig{Host: "127.0.0.1", Port: "0"},
		LogTarget: config.LogConfig{Path: t.TempDir(), FileName: "app.log"},
	}
	patchLogger := testutil.PatchLogger()
	t.Cleanup(patchLogger.Unpatch)
	srv := cmd.GetRESTServer(config.Cfg, testutil.StubLogger{})
	sv := reflect.ValueOf(srv).Elem().FieldByName("httpServer")
	httpSrv := reflect.NewAt(sv.Type(), unsafe.Pointer(sv.UnsafeAddr())).Elem().Interface().(*http.Server)
	return httpSrv
}

func TestUploadFileIntegration(t *testing.T) {
	srv := setupServer(t)
	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	fw, err := mw.CreateFormFile("file", "a.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = fw.Write([]byte("data"))
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/files/upload", &b)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
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

func TestUploadFileIntegrationMissingFile(t *testing.T) {
	srv := setupServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/files/upload", nil)
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestDownloadFileIntegration(t *testing.T) {
	srv := setupServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/files/download", nil)
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d", rec.Code)
	}
	if rec.Header().Get("Content-Disposition") == "" {
		t.Fatalf("missing disposition")
	}
}

func TestFormWithFileIntegration(t *testing.T) {
	srv := setupServer(t)
	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	_ = mw.WriteField("name", "bob")
	fw, err := mw.CreateFormFile("attachment", "b.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = fw.Write([]byte("data"))
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/files/form", &b)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
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

func TestFormWithFileIntegrationMissingFile(t *testing.T) {
	srv := setupServer(t)
	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	_ = mw.WriteField("name", "a")
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/files/form", &b)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestFormWithFileIntegrationBadForm(t *testing.T) {
	srv := setupServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/files/form", nil)
	req.Header.Set("Content-Type", "multipart/form-data")
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("code %d", rec.Code)
	}
}
