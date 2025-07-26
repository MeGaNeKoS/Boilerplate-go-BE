package handlers

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"project-template/infrastructure/config"
	"project-template/pkg/code"
)

func TestEchoCrashAndLongHandlers(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	rec := httptest.NewRecorder()

	EchoHandler(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("echo")) {
		t.Fatalf("bad body")
	}

	rec = httptest.NewRecorder()
	LongHandler(rec, httptest.NewRequest(http.MethodGet, "/?sleep=0", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("code %d", rec.Code)
	}

	defer func() { recover() }()
	CrashHandler(rec, httptest.NewRequest(http.MethodGet, "/", nil))
}

// stubSystemService allows forcing handlers into the error branches.
type stubSystemService struct {
	echoCode *code.Code
	longCode *code.Code
}

func (s stubSystemService) Echo(ctx context.Context) (interface{}, *code.Code) {
	return nil, s.echoCode
}

func (stubSystemService) Crash(ctx context.Context) {}

func (s stubSystemService) Long(ctx context.Context, _ int) (interface{}, *code.Code) {
	return nil, s.longCode
}

// TestEchoHandlerError exercises the error handling path of EchoHandler.
func TestEchoHandlerError(t *testing.T) {
	orig := systemService
	systemService = stubSystemService{echoCode: &code.ErrInternalServerError}
	defer func() { systemService = orig }()

	rec := httptest.NewRecorder()
	EchoHandler(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("code %d", rec.Code)
	}
}

// TestLongHandlerError exercises the error handling path of LongHandler.
func TestLongHandlerError(t *testing.T) {
	orig := systemService
	systemService = stubSystemService{longCode: &code.ErrInternalServerError}
	defer func() { systemService = orig }()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/?sleep=0", nil)
	LongHandler(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("code %d", rec.Code)
	}
}
