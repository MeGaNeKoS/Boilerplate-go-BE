package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bouk/monkey"

	"project-template/infrastructure/config"
	"project-template/infrastructure/dto/response"
	"project-template/infrastructure/supervisor"
	"project-template/infrastructure/utils"
)

type stubLogger struct{}

func (stubLogger) Debug(string)                  {}
func (stubLogger) DebugF(string, ...interface{}) {}
func (stubLogger) Info(string)                    {}
func (stubLogger) InfoF(string, ...interface{})  {}
func (stubLogger) Warn(string)                    {}
func (stubLogger) WarnF(string, ...interface{})  {}
func (stubLogger) Error(string)                   {}
func (stubLogger) ErrorF(string, ...interface{}) {}
func (stubLogger) Fatal(string)                   {}
func (stubLogger) FatalF(string, ...interface{}) {}
func (stubLogger) ParentID() string              { return "p" }
func (stubLogger) ChildID() string               { return "c" }
func (stubLogger) CloseLogFile()                 {}

func TestRecoverMiddleware(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP", LogTarget: config.LogConfig{Path: t.TempDir(), FileName: "app.log"}}

	mw := RecoverMiddleware(stubLogger{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test-path", nil)
	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	if rec.Header().Get("X-Recovered-By") != "RecoverMiddleware" {
		t.Fatalf("header not set")
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("expected application/problem+json, got %q", ct)
	}
	var pd response.ProblemDetail
	if err := json.NewDecoder(rec.Body).Decode(&pd); err != nil {
		t.Fatalf("failed to decode problem detail: %v", err)
	}
	if pd.Status != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", pd.Status)
	}
	if pd.Title == "" {
		t.Fatal("title is empty")
	}
	if pd.Type == "" {
		t.Fatal("type is empty")
	}
	if pd.Instance != "/test-path" {
		t.Fatalf("expected instance /test-path, got %q", pd.Instance)
	}
}

func TestRecoverMiddlewareWithReqLoggerAndRestartError(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP", LogTarget: config.LogConfig{Path: t.TempDir(), FileName: "app.log"}}

	var restartCalled bool
	monkey.Patch(supervisor.RequestRestart, func() error {
		restartCalled = true
		return errors.New("fail")
	})
	defer monkey.UnpatchAll()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	})
	mw := RecoverMiddleware(stubLogger{})(handler)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/panic-path", nil)
	req = req.WithContext(utils.SetLoggerToContext(req.Context(), stubLogger{}))
	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	if rec.Header().Get("X-Recovered-By") != "RecoverMiddleware" {
		t.Fatalf("header not set")
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("expected application/problem+json, got %q", ct)
	}
	var pd response.ProblemDetail
	if err := json.NewDecoder(rec.Body).Decode(&pd); err != nil {
		t.Fatalf("failed to decode problem detail: %v", err)
	}
	if pd.Status != http.StatusInternalServerError {
		t.Fatalf("expected status 500, got %d", pd.Status)
	}
	if pd.Instance != "/panic-path" {
		t.Fatalf("expected instance /panic-path, got %q", pd.Instance)
	}
	if !restartCalled {
		t.Fatal("restart not called")
	}
}
