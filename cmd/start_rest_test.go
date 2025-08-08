//go:build !grpc && !kafka

package cmd

import (
	"errors"
	"flag"
	"io"
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

	"project-template/infrastructure/config"
	"project-template/infrastructure/db"
	"project-template/infrastructure/utils"
	"project-template/pkg/logger"
	"project-template/server/rest/helpers"

	"github.com/bouk/monkey"
	"github.com/coreos/go-systemd/v22/activation"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"gopkg.in/yaml.v3"
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

func TestFilterInternal(t *testing.T) {
	type input struct {
		Secret string `header:"X-Secret" internal:"true"`
		Q      string `query:"q"`
	}
	helpers.ClearInternalParams()
	helpers.TrackInternalParams("pubOp", input{})
	spec := &huma.OpenAPI{
		Paths: map[string]*huma.PathItem{
			"/a": {Get: &huma.Operation{Tags: []string{helpers.InternalTag()}}},
			"/b": {
				Get: &huma.Operation{OperationID: "pubOp", Tags: []string{"pub"}, Parameters: []*huma.Param{
					{Name: "X-Secret", In: "header"},
					{Name: "q", In: "query"},
				}},
				Post: &huma.Operation{Tags: []string{helpers.InternalTag()}},
			},
		},
		Tags: []*huma.Tag{{Name: "pub"}, {Name: helpers.InternalTag()}},
	}
	filtered := filterInternal(spec)
	if _, ok := filtered.Paths["/a"]; ok {
		t.Fatalf("internal path kept")
	}
	b := filtered.Paths["/b"]
	if b == nil || b.Post != nil || b.Get == nil {
		t.Fatalf("unexpected filtered path: %#v", b)
	}
	if len(b.Get.Parameters) != 1 || b.Get.Parameters[0].Name != "q" {
		t.Fatalf("internal parameter not removed: %#v", b.Get.Parameters)
	}
	for _, tag := range filtered.Tags {
		if tag.Name == helpers.InternalTag() {
			t.Fatalf("internal tag not removed")
		}
	}
	orig := spec.Paths["/b"].Get.Parameters
	if len(orig) != 2 {
		t.Fatalf("original spec mutated: %#v", orig)
	}
	internal := stripInternalTag(spec)
	if params := internal.Paths["/b"].Get.Parameters; len(params) != 2 {
		t.Fatalf("internal parameter missing: %#v", params)
	}
}

func TestStripInternalTag(t *testing.T) {
	spec := &huma.OpenAPI{
		Paths: map[string]*huma.PathItem{
			"/a": {Get: &huma.Operation{Tags: []string{"pub", helpers.InternalTag()}}},
			"/b": {},
		},
		Tags: []*huma.Tag{{Name: helpers.InternalTag()}, {Name: "pub"}},
	}
	stripped := stripInternalTag(spec)
	got := stripped.Paths["/a"].Get.Tags
	if len(got) != 1 || got[0] != "pub" {
		t.Fatalf("tag not stripped: %v", got)
	}
	for _, tag := range stripped.Tags {
		if tag.Name == helpers.InternalTag() {
			t.Fatalf("internal tag not removed")
		}
	}
}

func TestRegisterDocs(t *testing.T) {
	router := chi.NewRouter()
	api := humachi.New(router, huma.DefaultConfig("t", "v"))
	spec := &huma.OpenAPI{OpenAPI: "3.0.3"}
	registerDocs(api, spec, "/openapi-test", "/docs/test", "Title")
	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/openapi-test.yaml")
	if err != nil {
		t.Fatalf("get spec: %v", err)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/vnd.oai.openapi+yaml" {
		t.Fatalf("unexpected ct %s", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(body), "openapi: 3.0.3") {
		t.Fatalf("bad spec body %s", body)
	}

	resp, err = http.Get(srv.URL + "/docs/test")
	if err != nil {
		t.Fatalf("get docs: %v", err)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/html" {
		t.Fatalf("unexpected ct %s", ct)
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(body), "<title>Title</title>") {
		t.Fatalf("bad html body %s", body)
	}
}

func TestGetRESTServerVersionOverride(t *testing.T) {
	resetREST()
	cfg := &config.Config{
		Version:   "1.2",
		REST:      config.ListenerConfig{Host: "127.0.0.1", Port: "0"},
		Server:    config.ServerConfig{Timeout: config.TimeoutConfig{Read: 1, Write: 1, Idle: 1}, Endpoint: config.EndpointConfig{Based: "/api"}},
		LogTarget: config.LogConfig{Path: t.TempDir(), FileName: "app.log"},
		OpenAPI:   config.OpenAPIConfig{Version: "3.1.0"},
	}
	srv := GetRESTServer(cfg, stubLog{})
	if srv == nil {
		t.Fatalf("nil server")
	}
	ts := httptest.NewServer(srv.httpServer.Handler)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/openapi-public.yaml")
	if err != nil {
		t.Fatalf("get spec: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	var spec map[string]any
	if err := yaml.Unmarshal(body, &spec); err != nil {
		t.Fatalf("parse spec: %v", err)
	}
	info, _ := spec["info"].(map[string]any)
	if v, _ := info["version"].(string); v != "1.2" {
		t.Fatalf("info version missing: %v", v)
	}
}

func TestGetRESTServerDocDefault(t *testing.T) {
	resetREST()
	routes := map[string]bool{}
	patchWalk := monkey.Patch(chiWalk, func(r chi.Routes, fn chi.WalkFunc) error {
		return chi.Walk(r, func(method, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
			routes[route] = true
			return nil
		})
	})
	defer patchWalk.Unpatch()

	cfg := &config.Config{
		REST:      config.ListenerConfig{Host: "127.0.0.1", Port: "0"},
		Server:    config.ServerConfig{Timeout: config.TimeoutConfig{Read: 1, Write: 1, Idle: 1}, Endpoint: config.EndpointConfig{Based: "/api"}},
		LogTarget: config.LogConfig{Path: t.TempDir(), FileName: "app.log"},
	}
	srv := GetRESTServer(cfg, stubLog{})
	if srv == nil {
		t.Fatalf("nil server")
	}
	if !routes["/docs/public"] || !routes["/docs/internal"] {
		t.Fatalf("doc routes missing: %v", routes)
	}
}

func TestGetRESTServerDocConfig(t *testing.T) {
	resetREST()
	routes := map[string]bool{}
	patchWalk := monkey.Patch(chiWalk, func(r chi.Routes, fn chi.WalkFunc) error {
		return chi.Walk(r, func(method, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
			routes[route] = true
			return nil
		})
	})
	defer patchWalk.Unpatch()

	cfg := &config.Config{
		Version:   "1.2",
		REST:      config.ListenerConfig{Host: "127.0.0.1", Port: "0"},
		Server:    config.ServerConfig{Timeout: config.TimeoutConfig{Read: 1, Write: 1, Idle: 1}, Endpoint: config.EndpointConfig{Based: "/api"}},
		LogTarget: config.LogConfig{Path: t.TempDir(), FileName: "app.log"},
		OpenAPI: config.OpenAPIConfig{
			Version: "3.1.0",
			Docs:    config.DocsConfig{Public: "/pub", Internal: "/int"},
		},
	}
	srv := GetRESTServer(cfg, stubLog{})
	if srv == nil {
		t.Fatalf("nil server")
	}
	if !routes["/pub"] || !routes["/int"] {
		t.Fatalf("doc routes missing: %v", routes)
	}
}
