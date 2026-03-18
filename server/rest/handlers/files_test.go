package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unsafe"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/humatest"

	"project-template/infrastructure/config"
	"project-template/infrastructure/dto/files"
	"project-template/infrastructure/dto/response"
)

// stubFile implements multipart.File in memory.
type stubFile struct{ *bytes.Reader }

func (stubFile) Close() error { return nil }

func setUploadData(m *huma.MultipartFormFiles[files.UploadForm], data *files.UploadForm) {
	v := reflect.ValueOf(m).Elem().FieldByName("data")
	reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem().Set(reflect.ValueOf(data))
}

func setFormData(m *huma.MultipartFormFiles[files.FormBody], data *files.FormBody) {
	v := reflect.ValueOf(m).Elem().FieldByName("data")
	reflect.NewAt(v.Type(), unsafe.Pointer(v.UnsafeAddr())).Elem().Set(reflect.ValueOf(data))
}

func TestUploadFile(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	in := &files.UploadInput{}
	setUploadData(&in.RawBody, &files.UploadForm{File: huma.FormFile{File: &stubFile{bytes.NewReader([]byte("data"))}, IsSet: true}})
	resp, err := UploadFile(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	ctx := humatest.NewContext(nil, httptest.NewRequest(http.MethodPost, "/", nil), rec)
	resp.Body(ctx)
	var out response.GenericResponse[files.UploadOutput]
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Body.Message != "uploaded" {
		t.Fatalf("body %#v", out.Body)
	}
}

func TestUploadFileError(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	_, err := UploadFile(context.Background(), &files.UploadInput{})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestDownloadFile(t *testing.T) {
	cwd, _ := os.Getwd()
	_ = os.Chdir("../../..")
	defer func(dir string) {
		err := os.Chdir(dir)
		if err != nil {
			t.Fatalf("failed to change directory: %v", err)
		}
	}(cwd)
	expected, err := os.ReadFile("public/public.txt")
	if err != nil {
		t.Fatal(err)
	}
	resp, err := DownloadFile(context.Background(), &files.DownloadInput{Name: "public.txt"})
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	ctx := humatest.NewContext(nil, httptest.NewRequest(http.MethodGet, "/", nil), rec)
	resp.Body(ctx)
	if !bytes.Equal(rec.Body.Bytes(), expected) {
		t.Fatalf("body %s", rec.Body.Bytes())
	}
	disp := resp.GetHeaders().Get("Content-Disposition")
	if disp == "" {
		t.Fatalf("missing disposition")
	}
	if !strings.Contains(disp, `filename="`) {
		t.Fatalf("filename not quoted in Content-Disposition: %s", disp)
	}
	if resp.GetHeaders().Get("Content-Type") != "text/plain; charset=utf-8" {
		t.Fatalf("content type %q", resp.GetHeaders().Get("Content-Type"))
	}
}

func TestDownloadFileNotFound(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	cwd, _ := os.Getwd()
	_ = os.Chdir("../../..")
	defer func(dir string) {
		err := os.Chdir(dir)
		if err != nil {
			t.Fatalf("failed to change directory: %v", err)
		}
	}(cwd)
	_, err := DownloadFile(context.Background(), &files.DownloadInput{Name: "missing.txt"})
	if err == nil {
		t.Fatalf("expected error")
	}
	var herr huma.StatusError
	if !errors.As(err, &herr) || herr.GetStatus() != http.StatusNotFound {
		t.Fatalf("status %v", err)
	}
}

func TestDownloadFileInternalError(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	cwd, _ := os.Getwd()
	_ = os.Chdir("../../..")
	defer func() { _ = os.Chdir(cwd) }()
	_, err := DownloadFile(context.Background(), &files.DownloadInput{Name: ""})
	if err == nil {
		t.Fatalf("expected error")
	}
	var herr huma.StatusError
	if !errors.As(err, &herr) || herr.GetStatus() != http.StatusInternalServerError {
		t.Fatalf("status %v", err)
	}
}

func TestDownloadFileMimeFallback(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	cwd, _ := os.Getwd()
	_ = os.Chdir("../../..")
	defer func(dir string) {
		err := os.Chdir(dir)
		if err != nil {
			t.Fatalf("failed to change directory: %v", err)
		}
	}(cwd)
	path := filepath.Join("public", "nomime")
	if err := os.WriteFile(path, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	defer func(name string) {
		err := os.Remove(name)
		if err != nil {
			t.Fatalf("failed to remove file %s: %v", name, err)
		}
	}(path)
	resp, err := DownloadFile(context.Background(), &files.DownloadInput{Name: "nomime"})
	if err != nil {
		t.Fatal(err)
	}
	if ct := resp.GetHeaders().Get("Content-Type"); ct != "text/plain; charset=utf-8" {
		t.Fatalf("content type %q", ct)
	}
}

func TestFormWithFile(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	in := &files.FormInput{}
	setFormData(&in.RawBody, &files.FormBody{Name: "bob", Attachment: huma.FormFile{File: &stubFile{bytes.NewReader([]byte("data"))}, IsSet: true}})
	resp, err := FormWithFile(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	ctx := humatest.NewContext(nil, httptest.NewRequest(http.MethodPost, "/", nil), rec)
	resp.Body(ctx)
	var out response.GenericResponse[files.FormOutput]
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Body.Name != "bob" {
		t.Fatalf("body %#v", out.Body)
	}
}

func TestFormWithFileError(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	in := &files.FormInput{}
	setFormData(&in.RawBody, &files.FormBody{Name: "x"})
	_, err := FormWithFile(context.Background(), in)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestFormWithFileMissingFile(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	in := &files.FormInput{}
	setFormData(&in.RawBody, &files.FormBody{Name: "a"})
	_, err := FormWithFile(context.Background(), in)
	if err == nil {
		t.Fatalf("expected error")
	}
}
