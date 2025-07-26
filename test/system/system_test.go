//go:build !grpc && !kafka

package system_test

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"unsafe"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/bouk/monkey"

	"project-template/cmd"
	"project-template/infrastructure/config"
	"project-template/infrastructure/dto/response"
	"project-template/outbound/transport"
	codepkg "project-template/pkg/code"
	"project-template/pkg/logger"
	"project-template/test/testutil"
)

func setupServer(t *testing.T) (*http.Server, sqlmock.Sqlmock) {
	t.Helper()
	config.Cfg = &config.Config{
		AppName:   "APP",
		Server:    config.ServerConfig{Endpoint: config.EndpointConfig{Based: "/api"}, Timeout: config.TimeoutConfig{Server: 1}},
		REST:      config.ListenerConfig{Host: "127.0.0.1", Port: "0"},
		LogTarget: config.LogConfig{Path: t.TempDir(), FileName: "app.log"},
	}

	sqlDB, mock := testutil.NewDBMock(t)
	patchDB := testutil.PatchDB(sqlDB)
	t.Cleanup(patchDB.Unpatch)

	patchLogger := testutil.PatchLogger()
	t.Cleanup(patchLogger.Unpatch)

	patchHTTP := monkey.PatchInstanceMethod(reflect.TypeOf(&transport.HTTPOutbound{}), "SendHTTPRequest", func(_ *transport.HTTPOutbound, _ logger.Logger) (response.HttpResponse, *codepkg.Code) {
		return response.HttpResponse{HTTPCode: http.StatusOK, RawResponsePayload: &response.GenericResponse{}}, nil
	})
	t.Cleanup(patchHTTP.Unpatch)

	srv := cmd.GetRESTServer(config.Cfg, testutil.StubLogger{})
	sv := reflect.ValueOf(srv).Elem().FieldByName("httpServer")
	httpSrv := reflect.NewAt(sv.Type(), unsafe.Pointer(sv.UnsafeAddr())).Elem().Interface().(*http.Server)
	return httpSrv, mock
}

func TestSystemEchoIntegration(t *testing.T) {
	httpSrv, _ := setupServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/system/echo", nil)
	rec := httptest.NewRecorder()

	httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("code %d", rec.Code)
	}
}
func TestSystemLongIntegration(t *testing.T) {
	httpSrv, _ := setupServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/system/long?sleep=0", nil)
	rec := httptest.NewRecorder()

	httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("code %d", rec.Code)
	}
}

func TestSystemCrashIntegration(t *testing.T) {
	httpSrv, _ := setupServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/system/crash", nil)
	rec := httptest.NewRecorder()

	httpSrv.Handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("code %d", rec.Code)
	}
	if rec.Header().Get("X-Recovered-By") != "RecoverMiddleware" {
		t.Fatalf("missing header")
	}
}
