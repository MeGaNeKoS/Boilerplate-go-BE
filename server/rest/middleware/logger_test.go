package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"project-template/infrastructure/config"
	"project-template/infrastructure/utils"
	"project-template/pkg/logger"

	"github.com/bouk/monkey"
)

type recordLogger struct{ parent string }

func (r recordLogger) DebugF(string, ...interface{}) {}
func (r recordLogger) InfoF(string, ...interface{})  {}
func (r recordLogger) WarnF(string, ...interface{})  {}
func (r recordLogger) ErrorF(string, ...interface{}) {}
func (r recordLogger) FatalF(string, ...interface{}) {}
func (r recordLogger) ParentID() string              { return r.parent }
func (r recordLogger) ChildID() string               { return "" }
func (r recordLogger) CloseLogFile()                 {}

func TestLoggerMiddlewareInjectsLogger(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP", LogTarget: config.LogConfig{Path: t.TempDir(), FileName: "app.log"}}
	monkey.Patch(logger.NewLogger, func(cfg config.LogConfig, p, c string) (logger.Logger, error) {
		return recordLogger{parent: p}, nil
	})
	defer monkey.UnpatchAll()

	var pid string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := utils.GetLoggerFromContext(r.Context())
		if log == nil {
			t.Fatal("logger missing")
		}
		pid = log.ParentID()
	})

	mw := LoggerMiddleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Parent-Id", "pid")
	mw.ServeHTTP(httptest.NewRecorder(), req)

	if pid != "pid" {
		t.Fatalf("expected parent id pid, got %s", pid)
	}
}

func TestLoggerMiddlewareGeneratesParentID(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP", LogTarget: config.LogConfig{Path: t.TempDir(), FileName: "app.log"}}
	monkey.Patch(logger.NewLogger, func(cfg config.LogConfig, p, c string) (logger.Logger, error) {
		return recordLogger{parent: p}, nil
	})
	defer monkey.UnpatchAll()

	var got bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := utils.GetLoggerFromContext(r.Context())
		if log == nil || log.ParentID() == "" {
			t.Fatal("generated parent id missing")
		}
		got = true
	})

	mw := LoggerMiddleware(handler)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	mw.ServeHTTP(httptest.NewRecorder(), req)

	if !got {
		t.Fatal("handler not called")
	}
}

func TestLoggerMiddlewareInitError(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP", LogTarget: config.LogConfig{Path: t.TempDir(), FileName: "app.log"}}
	mw := LoggerMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called")
	}))

	// patch NewLogger to fail
	called := false
	monkey.Patch(logger.NewLogger, func(config.LogConfig, string, string) (logger.Logger, error) {
		called = true
		return nil, errors.New("fail")
	})
	defer monkey.UnpatchAll()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	mw.ServeHTTP(rec, req)

	if !called {
		t.Fatal("NewLogger not called")
	}
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", rec.Code)
	}
}
