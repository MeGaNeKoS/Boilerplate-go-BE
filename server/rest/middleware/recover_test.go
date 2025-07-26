package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"errors"

	"github.com/bouk/monkey"

	"project-template/infrastructure/config"
	"project-template/infrastructure/supervisor"
	"project-template/infrastructure/utils"
)

type stubLogger struct{}

func (stubLogger) DebugF(string, ...interface{}) {}
func (stubLogger) InfoF(string, ...interface{})  {}
func (stubLogger) WarnF(string, ...interface{})  {}
func (stubLogger) ErrorF(string, ...interface{}) {}
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
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	if rec.Header().Get("X-Recovered-By") != "RecoverMiddleware" {
		t.Fatalf("header not set")
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
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(utils.SetLoggerToContext(req.Context(), stubLogger{}))
	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	if rec.Header().Get("X-Recovered-By") != "RecoverMiddleware" {
		t.Fatalf("header not set")
	}
	if !restartCalled {
		t.Fatal("restart not called")
	}
}
