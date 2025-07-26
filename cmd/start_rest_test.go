//go:build !grpc && !kafka

package cmd

import (
	"errors"
	"flag"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"project-template/docs"
	"project-template/infrastructure/config"
	"project-template/infrastructure/db"
	"project-template/infrastructure/utils"
	"project-template/pkg/logger"
	"project-template/server/rest/middleware"
	"project-template/server/rest/routes"

	"github.com/bouk/monkey"
	"github.com/coreos/go-systemd/v22/activation"
	"github.com/go-chi/chi/v5"
)

var serverClosedErr = http.ErrServerClosed

func patchServer(l logger.Logger, err error) func() {
	patchGet := monkey.Patch(GetRESTServer, func(*config.Config, logger.Logger) *RestServer {
		return &RestServer{logger: l}
	})
	patchServe := monkey.Patch((*RestServer).StartServer, func(*RestServer) error { return err })
	return func() { patchGet.Unpatch(); patchServe.Unpatch() }
}

func TestStartRESTRun(t *testing.T) {
	if os.Getenv("REST_CHILD") == "1" {
		path := os.Getenv("CFG_PATH")
		fs := flag.NewFlagSet("test", flag.ContinueOnError)
		flag.CommandLine = fs
		os.Args = []string{"cmd", "--config", path}
		patchLogger := monkey.Patch(logger.NewLogger, func(c config.LogConfig, p, cID string) (logger.Logger, error) { return stubLog{}, nil })
		patchDB := monkey.Patch(db.GetDBInstance, func() db.DB { return stubDB{} })
		patchMig := monkey.Patch(db.RunMigrations, func(string) error { return nil })
		patchJWT := monkey.Patch(utils.InitializeJWTService, func(private, public bool) error { return nil })
		defer patchLogger.Unpatch()
		defer patchDB.Unpatch()
		defer patchMig.Unpatch()
		defer patchJWT.Unpatch()
		Start(nil)
		return
	}

	cfgPath := writeRunConfig(t)
	f, err := os.CreateTemp("", "cfg-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())
	content, _ := os.ReadFile(cfgPath)
	// make rest port invalid to force failure
	data := strings.Replace(string(content), "Port: \"0\"", "Port: bad", 1)
	os.WriteFile(f.Name(), []byte(data), 0600)

	exe, _ := os.Executable()
	cmd := exec.Command(exe, "-test.run=TestStartRESTRun")
	cmd.Env = append(os.Environ(), "REST_CHILD=1", "CFG_PATH="+f.Name())
	if err = cmd.Run(); err != nil {
		t.Fatalf("child run error: %v", err)
	}
}

func TestStartRESTSuccess(t *testing.T) {
	patchParse := monkey.Patch(parseFlags, func(chan struct{}) (string, bool) { return "cfg", false })
	patchInit := monkey.Patch(initDependencies, func(string) (logger.Logger, error) { return stubLog{}, nil })
	patchGet := monkey.Patch(GetRESTServer, func(*config.Config, logger.Logger) *RestServer { return &RestServer{} })
	patchServe := monkey.Patch((*RestServer).StartServer, func(*RestServer) error { return nil })
	defer patchParse.Unpatch()
	defer patchInit.Unpatch()
	defer patchGet.Unpatch()
	defer patchServe.Unpatch()
	Start(nil)
}

func TestStartRESTInitError(t *testing.T) {
	patchParse := monkey.Patch(parseFlags, func(chan struct{}) (string, bool) { return "cfg", false })
	patchInit := monkey.Patch(initDependencies, func(string) (logger.Logger, error) { return nil, errors.New("bad") })
	defer patchParse.Unpatch()
	defer patchInit.Unpatch()

	Start(nil)
}

func writeRunConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cfg := `AppName: APP
Server:
  Environment: dev
  Endpoint:
    Based: "/api"
  Timeout:
    Read: 1
    Write: 1
    Idle: 1
REST:
  Host: 127.0.0.1
  Port: "0"
GRPC:
  Host: 127.0.0.1
  Port: "0"
Kafka:
  Brokers: ["b"]
  GroupID: "g"
  Topic: "t"
Database:
  Host: db
  Port: "3306"
  User: u
  Password: p
  DBName: test
  MigrationPath: file://migrations
  DirtyStrategy: retry
LogTarget:
  Path: ` + dir + `
  FileName: app.log
`
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(cfg), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestStartDoneChanSuccess(t *testing.T) {
	patchParse := monkey.Patch(parseFlags, func(chan struct{}) (string, bool) { return "cfg", false })
	patchInit := monkey.Patch(initDependencies, func(string) (logger.Logger, error) { return stubLog{}, nil })
	unpatchSrv := patchServer(stubLog{}, nil)
	defer patchParse.Unpatch()
	defer patchInit.Unpatch()
	defer unpatchSrv()

	done := make(chan struct{})
	Start(done)
	select {
	case <-done:
	default:
		t.Fatalf("done not closed")
	}
}

func TestStartDoneChanInitErr(t *testing.T) {
	patchParse := monkey.Patch(parseFlags, func(chan struct{}) (string, bool) { return "cfg", false })
	patchInit := monkey.Patch(initDependencies, func(string) (logger.Logger, error) { return nil, errors.New("bad") })
	defer patchParse.Unpatch()
	defer patchInit.Unpatch()

	done := make(chan struct{})
	Start(done)
	select {
	case <-done:
	default:
		t.Fatalf("done not closed")
	}
}

func TestStartDoneChanServeErr(t *testing.T) {
	patchParse := monkey.Patch(parseFlags, func(chan struct{}) (string, bool) { return "cfg", false })
	patchInit := monkey.Patch(initDependencies, func(string) (logger.Logger, error) { return stubLog{}, nil })
	unpatchSrv := patchServer(stubLog{}, errors.New("fail"))
	defer patchParse.Unpatch()
	defer patchInit.Unpatch()
	defer unpatchSrv()

	done := make(chan struct{})
	Start(done)
	select {
	case <-done:
	default:
		t.Fatalf("done not closed")
	}
}

func TestStartLogsServerClosed(t *testing.T) {
	rl := &recordLogger{}
	patchParse := monkey.Patch(parseFlags, func(chan struct{}) (string, bool) { return "cfg", false })
	patchInit := monkey.Patch(initDependencies, func(string) (logger.Logger, error) { return rl, nil })
	unpatchSrv := patchServer(rl, serverClosedErr)
	defer patchParse.Unpatch()
	defer patchInit.Unpatch()
	defer unpatchSrv()

	Start(nil)

	if len(rl.infos) == 0 || rl.infos[len(rl.infos)-1] != "server closed" {
		t.Fatalf("expected server closed log, got %v", rl.infos)
	}
}

func resetREST() {
	restInstance = nil
	restCreateOnce = sync.Once{}
	restShutdownOnce = sync.Once{}
}

func TestStartServerBadEnv(t *testing.T) {
	os.Setenv("SYSTEMD_SOCKET_ACTIVATION", "bad")
	defer os.Unsetenv("SYSTEMD_SOCKET_ACTIVATION")
	cfg := &config.Config{REST: config.ListenerConfig{Host: "127.0.0.1", Port: "0"}, Server: config.ServerConfig{Timeout: config.TimeoutConfig{Read: 1, Write: 1, Idle: 1}}}
	srv := GetRESTServer(cfg, stubLog{})
	err := srv.StartServer()
	if err == nil || !strings.Contains(err.Error(), "invalid value") {
		t.Fatalf("expected parse error got %v", err)
	}
}

func TestGetRESTServerSingleton(t *testing.T) {
	resetREST()
	cfg := &config.Config{}
	s1 := GetRESTServer(cfg, stubLog{})
	s2 := GetRESTServer(cfg, stubLog{})
	if s1 != s2 {
		t.Fatalf("expected singleton")
	}
}

func TestStartAndShutdown(t *testing.T) {
	resetREST()
	cfg := &config.Config{REST: config.ListenerConfig{Host: "127.0.0.1", Port: "0"}, Server: config.ServerConfig{Timeout: config.TimeoutConfig{Read: 1, Write: 1, Idle: 1}, Endpoint: config.EndpointConfig{Based: "/api"}}, LogTarget: config.LogConfig{Path: t.TempDir(), FileName: "app.log"}}
	srv := GetRESTServer(cfg, stubLog{})
	done := make(chan error, 1)
	go func() { done <- srv.StartServer() }()
	time.Sleep(100 * time.Millisecond)
	if err := RestShutdownServer(); err != nil {
		t.Fatalf("shutdown: %v", err)
	}
	err := <-done
	if err != nil && !strings.Contains(err.Error(), "Server closed") {
		t.Fatalf("serve error %v", err)
	}
	RestCloseListener()
	if srv.listener != nil {
		t.Fatalf("listener not nil")
	}
}

func TestRestCloseListener(t *testing.T) {
	resetREST()
	RestCloseListener()
}

func TestRestShutdownServer(t *testing.T) {
	resetREST()
	if err := RestShutdownServer(); err != nil {
		t.Fatalf("shutdown error: %v", err)
	}
}

func TestRestStartErrors(t *testing.T) {
	srv := &RestServer{}
	if err := srv.StartServer(); err == nil || err.Error() != "http server is not initialized" {
		t.Fatalf("expected not initialized, got %v", err)
	}

	srv = GetRESTServer(&config.Config{REST: config.ListenerConfig{Host: "127.0.0.1", Port: "0"}}, stubLog{})
	os.Setenv("SYSTEMD_SOCKET_ACTIVATION", "true")
	patchList := monkey.Patch(activation.Listeners, func() ([]net.Listener, error) { return nil, errors.New("boom") })
	if err := srv.StartServer(); err == nil || err.Error() != "boom" {
		t.Fatalf("expected boom, got %v", err)
	}
	patchList.Unpatch()

	patchList = monkey.Patch(activation.Listeners, func() ([]net.Listener, error) { return []net.Listener{}, nil })
	if err := srv.StartServer(); err == nil || err.Error() != "no systemd listeners found" {
		t.Fatalf("expected none, got %v", err)
	}
	patchList.Unpatch()
	os.Unsetenv("SYSTEMD_SOCKET_ACTIVATION")

	patchNet := monkey.Patch(net.Listen, func(network, address string) (net.Listener, error) { return nil, errors.New("fail") })
	if err := srv.StartServer(); err == nil || err.Error() != "fail" {
		t.Fatalf("expected fail, got %v", err)
	}
	patchNet.Unpatch()
}

func TestRestStartWithSystemdListener(t *testing.T) {
	resetREST()
	cfg := &config.Config{REST: config.ListenerConfig{Host: "127.0.0.1", Port: "0"}, Server: config.ServerConfig{Timeout: config.TimeoutConfig{Read: 1, Write: 1, Idle: 1}}}
	srv := GetRESTServer(cfg, stubLog{})
	ln, _ := net.Listen("tcp", "127.0.0.1:0")
	os.Setenv("SYSTEMD_SOCKET_ACTIVATION", "true")
	patchList := monkey.Patch(activation.Listeners, func() ([]net.Listener, error) { return []net.Listener{ln}, nil })
	patchServe := monkey.Patch((*http.Server).Serve, func(_ *http.Server, ln net.Listener) error { ln.Close(); return nil })
	err := srv.StartServer()
	patchServe.Unpatch()
	patchList.Unpatch()
	os.Unsetenv("SYSTEMD_SOCKET_ACTIVATION")
	if err != nil {
		t.Fatalf("unexpected %v", err)
	}
}

func TestGetRESTServerWalkError(t *testing.T) {
	resetREST()
	called := false
	patchWalk := monkey.Patch(chiWalk, func(chi.Routes, chi.WalkFunc) error {
		called = true
		return errors.New("walk")
	})
	defer patchWalk.Unpatch()

	cfg := &config.Config{}
	srv := GetRESTServer(cfg, stubLog{})
	if !called {
		t.Fatalf("patch not called")
	}
	if srv != nil {
		t.Fatalf("expected nil server when walk fails")
	}
}

func TestGetRESTServerEmptyPrefix(t *testing.T) {
	resetREST()
	oldGroups := routes.RouteGroups
	called := false
	routes.RouteGroups = []routes.RouteGroup{{
		Prefix: "",
		Routes: &[]routes.RouteDef{{
			Method:  http.MethodGet,
			Pattern: "/x",
			Handler: func(http.ResponseWriter, *http.Request) { called = true },
		}},
	}}
	defer func() { routes.RouteGroups = oldGroups }()

	savedSpec := docs.Spec
	defer func() { docs.Spec = savedSpec }()
	chiWalk = func(chi.Routes, chi.WalkFunc) error { return nil }
	defer func() { chiWalk = chi.Walk }()
	cfg := &config.Config{REST: config.ListenerConfig{Host: "127.0.0.1", Port: "0"}, Server: config.ServerConfig{Timeout: config.TimeoutConfig{Read: 1, Write: 1, Idle: 1}}}
	srv := GetRESTServer(cfg, stubLog{})
	if srv == nil {
		t.Fatalf("server nil")
	}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	srv.httpServer.Handler.ServeHTTP(rr, req)
	if !called {
		t.Fatalf("handler not called")
	}
}

func TestGetRESTServerAuthGroup(t *testing.T) {
	resetREST()
	oldGroups := routes.RouteGroups
	called := false
	routes.RouteGroups = []routes.RouteGroup{{
		Prefix:  "",
		UseAuth: true,
		Routes: &[]routes.RouteDef{{
			Method:  http.MethodGet,
			Pattern: "/x",
			Handler: func(http.ResponseWriter, *http.Request) { called = true },
		}},
	}}
	defer func() { routes.RouteGroups = oldGroups }()

	savedSpec := docs.Spec
	defer func() { docs.Spec = savedSpec }()
	chiWalk = func(chi.Routes, chi.WalkFunc) error { return nil }
	defer func() { chiWalk = chi.Walk }()
	authCalled := false
	patchAuth := monkey.Patch(middleware.AuthMiddleware, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authCalled = true
			next.ServeHTTP(w, r)
		})
	})
	defer patchAuth.Unpatch()

	cfg := &config.Config{REST: config.ListenerConfig{Host: "127.0.0.1", Port: "0"}, Server: config.ServerConfig{Timeout: config.TimeoutConfig{Read: 1, Write: 1, Idle: 1}}}
	srv := GetRESTServer(cfg, stubLog{})
	if srv == nil {
		t.Fatalf("server nil")
	}
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	srv.httpServer.Handler.ServeHTTP(rr, req)
	if !authCalled {
		t.Fatalf("auth middleware not called")
	}
	if !called {
		t.Fatalf("handler not called")
	}
}
