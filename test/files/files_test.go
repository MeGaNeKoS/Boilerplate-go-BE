//go:build !grpc && !kafka

package files_test

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"project-template/cmd"
	"project-template/infrastructure/config"
	"project-template/test/testutil"
)

func setupServer(t *testing.T) *http.Server {
	t.Helper()
	cwd, _ := os.Getwd()
	_ = os.Chdir("../..")
	t.Cleanup(func() { _ = os.Chdir(cwd) })
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
	err = mw.Close()
	if err != nil {
		return
	}

	req := httptest.NewRequest(http.MethodPost, "/api/files/upload", &b)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d, body: %s", rec.Code, rec.Body.String())
	}
}

func TestUploadFileIntegrationMissingFile(t *testing.T) {
	srv := setupServer(t)
	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	err := mw.Close()
	if err != nil {
		return
	}
	req := httptest.NewRequest(http.MethodPost, "/api/files/upload", &b)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestDownloadFileIntegration(t *testing.T) {
	srv := setupServer(t)
	expected, err := os.ReadFile("public/public.txt")
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/files/download/public.txt", nil)
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d", rec.Code)
	}
	if v := rec.Header().Get("Headers"); v != "" {
		t.Fatalf("unexpected Headers header: %q", v)
	}
	if cd := rec.Header().Get("Content-Disposition"); cd == "" {
		t.Fatalf("missing Content-Disposition header")
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Fatalf("content type %q", ct)
	}
	if !bytes.Equal(rec.Body.Bytes(), expected) {
		t.Fatalf("body %s", rec.Body.String())
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
	err = mw.Close()
	if err != nil {
		return
	}

	req := httptest.NewRequest(http.MethodPost, "/api/files/form", &b)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestFormWithFileIntegrationMissingFile(t *testing.T) {
	srv := setupServer(t)
	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	_ = mw.WriteField("name", "a")
	err := mw.Close()
	if err != nil {
		return
	}

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
