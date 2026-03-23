package handlers

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	neomachi "github.com/MeGaNeKoS/neoma/adapters/neomachi/v5"
	"github.com/MeGaNeKoS/neoma/core"

	"project-template/infrastructure/config"
	"project-template/server/rest/dto/files"
	"project-template/pkg/code"
)

// stubFile implements multipart.File in memory.
type stubFile struct{ *bytes.Reader }

func (stubFile) Close() error { return nil }

func TestUploadFile(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	in := &files.UploadInput{}
	in.Body.File = core.FormFile{File: &stubFile{bytes.NewReader([]byte("data"))}}
	resp, err := UploadFile(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	out := resp.Body
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
	ctx := neomachi.NewContext(nil, httptest.NewRequest(http.MethodGet, "/", nil), rec)
	resp.Body(ctx, nil)
	if !bytes.Equal(rec.Body.Bytes(), expected) {
		t.Fatalf("body %s", rec.Body.Bytes())
	}
	disp := resp.ContentDisposition
	if disp == "" {
		t.Fatalf("missing disposition")
	}
	if !strings.Contains(disp, `filename="`) {
		t.Fatalf("filename not quoted in Content-Disposition: %s", disp)
	}
	if resp.ContentType != "text/plain; charset=utf-8" {
		t.Fatalf("content type %q", resp.ContentType)
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
	var herr *code.Code
	if !errors.As(err, &herr) || herr.HTTPCode != http.StatusNotFound {
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
	var herr *code.Code
	if !errors.As(err, &herr) || herr.HTTPCode != http.StatusInternalServerError {
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
	if ct := resp.ContentType; ct != "text/plain; charset=utf-8" {
		t.Fatalf("content type %q", ct)
	}
}

func TestFormWithFile(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	in := &files.FormInput{}
	in.Body.Name = "bob"
	in.Body.Attachment = core.FormFile{File: &stubFile{bytes.NewReader([]byte("data"))}}
	resp, err := FormWithFile(context.Background(), in)
	if err != nil {
		t.Fatal(err)
	}
	out := resp.Body
	if out.Body.Name != "bob" {
		t.Fatalf("body %#v", out.Body)
	}
}

func TestFormWithFileError(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	_, err := FormWithFile(context.Background(), &files.FormInput{})
	if err == nil {
		t.Fatalf("expected error")
	}
}
