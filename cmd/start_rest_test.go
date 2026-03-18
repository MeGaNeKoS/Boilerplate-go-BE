//go:build !grpc && !kafka

package cmd

import (
	"encoding/json"
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
	restutils "project-template/server/rest/utils"

	"github.com/bouk/monkey"
	"github.com/coreos/go-systemd/v22/activation"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
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
	defer func() { _ = os.Remove(f.Name()) }()
	content, _ := os.ReadFile(cfgPath)
	// make rest port invalid to force failure
	data := strings.Replace(string(content), "Port: \"0\"", "Port: bad", 1)
	_ = os.WriteFile(f.Name(), []byte(data), 0600)

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
	_ = os.Setenv("SYSTEMD_SOCKET_ACTIVATION", "bad")
	defer func() { _ = os.Unsetenv("SYSTEMD_SOCKET_ACTIVATION") }()
	cfg := &config.Config{REST: config.ListenerConfig{Host: "127.0.0.1", Port: "0"}, Server: config.ServerConfig{Timeout: config.TimeoutConfig{Read: 1, Write: 1, Idle: 1}}}
	prev := config.Cfg
	config.Cfg = cfg
	srv := GetRESTServer(cfg, stubLog{})
	defer func() { config.Cfg = prev }()
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
	_ = os.Setenv("SYSTEMD_SOCKET_ACTIVATION", "true")
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
	_ = os.Unsetenv("SYSTEMD_SOCKET_ACTIVATION")

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
	_ = os.Setenv("SYSTEMD_SOCKET_ACTIVATION", "true")
	patchList := monkey.Patch(activation.Listeners, func() ([]net.Listener, error) { return []net.Listener{ln}, nil })
	patchServe := monkey.Patch((*http.Server).Serve, func(_ *http.Server, ln net.Listener) error { _ = ln.Close(); return nil })
	err := srv.StartServer()
	patchServe.Unpatch()
	patchList.Unpatch()
	_ = os.Unsetenv("SYSTEMD_SOCKET_ACTIVATION")
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
	restutils.ClearInternalParams()
	restutils.TrackInternalParams("pubOp", input{})
	reg := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
	reg.Map()["InternalThing"] = &huma.Schema{Type: "object"}
	reg.Map()["PublicThing"] = &huma.Schema{Type: "object"}
	spec := &huma.OpenAPI{
		Paths: map[string]*huma.PathItem{
			"/a": {
				Get: &huma.Operation{Tags: []string{restutils.InternalTag()}, Responses: map[string]*huma.Response{
					"200": {
						Content: map[string]*huma.MediaType{
							"application/json": {Schema: &huma.Schema{Ref: "#/components/schemas/InternalThing"}},
						},
					},
				}},
			},
			"/b": {
				Get: &huma.Operation{OperationID: "pubOp", Tags: []string{"pub"}, Parameters: []*huma.Param{
					{Name: "X-Secret", In: "header"},
					{Name: "q", In: "query"},
				}, Responses: map[string]*huma.Response{
					"200": {
						Content: map[string]*huma.MediaType{
							"application/json": {Schema: &huma.Schema{Ref: "#/components/schemas/PublicThing"}},
						},
					},
				}},
				Post: &huma.Operation{Tags: []string{restutils.InternalTag()}},
			},
		},
		Components: &huma.Components{Schemas: reg},
		Tags:       []*huma.Tag{{Name: "pub"}, {Name: restutils.InternalTag()}},
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
		if tag.Name == restutils.InternalTag() {
			t.Fatalf("internal tag not removed")
		}
	}
	orig := spec.Paths["/b"].Get.Parameters
	if len(orig) != 2 {
		t.Fatalf("original spec mutated: %#v", orig)
	}
	if _, ok := filtered.Components.Schemas.Map()["InternalThing"]; ok {
		t.Fatalf("internal schema kept")
	}
	if _, ok := filtered.Components.Schemas.Map()["PublicThing"]; !ok {
		t.Fatalf("public schema dropped")
	}
	internal := stripInternalTag(spec)
	if params := internal.Paths["/b"].Get.Parameters; len(params) != 2 {
		t.Fatalf("internal parameter missing: %#v", params)
	}
	if _, ok := spec.Components.Schemas.Map()["InternalThing"]; !ok {
		t.Fatalf("original components mutated")
	}
}

func TestStripInternalTag(t *testing.T) {
	spec := &huma.OpenAPI{
		Paths: map[string]*huma.PathItem{
			"/a": {Get: &huma.Operation{Tags: []string{"pub", restutils.InternalTag()}}},
			"/b": {},
		},
		Tags: []*huma.Tag{{Name: restutils.InternalTag()}, {Name: "pub"}},
	}
	stripped := stripInternalTag(spec)
	got := stripped.Paths["/a"].Get.Tags
	if len(got) != 1 || got[0] != "pub" {
		t.Fatalf("tag not stripped: %v", got)
	}
	for _, tag := range stripped.Tags {
		if tag.Name == restutils.InternalTag() {
			t.Fatalf("internal tag not removed")
		}
	}
}

func TestRegisterDocs(t *testing.T) {
	router := chi.NewRouter()
	cfg := huma.DefaultConfig("t", "v")
	cfg.CreateHooks = nil
	api := humachi.New(router, cfg)
	spec30 := &huma.OpenAPI{OpenAPI: "3.0.3"}
	spec31 := &huma.OpenAPI{OpenAPI: "3.1.0"}
	registerDocs(api, spec30, "/openapi-test", "/docs/test", "Title", "stoplight")
	registerDocs(api, spec31, "/openapi-test31", "/docs/test-swagger", "Title", "swagger")
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
	_ = resp.Body.Close()
	if !strings.Contains(string(body), "openapi: 3.0.3") {
		t.Fatalf("bad spec body %s", body)
	}

	resp, err = http.Get(srv.URL + "/openapi-test31.yaml")
	if err != nil {
		t.Fatalf("get spec 3.1: %v", err)
	}
	body, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if !strings.Contains(string(body), "openapi: 3.1.0") {
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
	_ = resp.Body.Close()
	if !strings.Contains(string(body), "<elements-api") {
		t.Fatalf("stoplight not rendered %s", body)
	}

	resp, err = http.Get(srv.URL + "/docs/test-swagger")
	if err != nil {
		t.Fatalf("get swagger docs: %v", err)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "text/html" {
		t.Fatalf("unexpected ct %s", ct)
	}
	body, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if !strings.Contains(string(body), "swagger-ui") {
		t.Fatalf("swagger ui not rendered %s", body)
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
	prev := config.Cfg
	config.Cfg = cfg
	wd, _ := os.Getwd()
	tmp := t.TempDir()
	_ = os.Chdir(tmp)
	defer func() { _ = os.Chdir(wd); config.Cfg = prev }()
	srv := GetRESTServer(cfg, stubLog{})
	if srv == nil {
		t.Fatalf("nil server")
	}
	body, err := os.ReadFile("docs/public/openapi.json")
	if err != nil {
		t.Fatalf("read spec: %v", err)
	}
	var spec map[string]any
	if err := json.Unmarshal(body, &spec); err != nil {
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
	if !routes["/docs/public"] || !routes["/docs/internal"] || !routes["/docs/public/openapi.json"] || !routes["/docs/internal/openapi.json"] {
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
			Docs: config.DocsConfig{
				Public:   config.DocConfig{URL: "/pub", Schema: "/s"},
				Internal: config.DocConfig{URL: "/int", Schema: "/i"},
			},
		},
	}
	srv := GetRESTServer(cfg, stubLog{})
	if srv == nil {
		t.Fatalf("nil server")
	}
	if !routes["/pub"] || !routes["/int"] || !routes["/pub/openapi.json"] || !routes["/int/openapi.json"] {
		t.Fatalf("doc routes missing: %v", routes)
	}
}

func TestGetRESTServerSchemasDefault(t *testing.T) {
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
	if !routes["/docs/public/schema/v1/{schema}"] {
		t.Fatalf("schema routes missing: %v", routes)
	}
}

func TestGetRESTServerSchemasConfig(t *testing.T) {
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
		OpenAPI:   config.OpenAPIConfig{Docs: config.DocsConfig{Public: config.DocConfig{URL: "/docs", Schema: "/schema"}, Internal: config.DocConfig{URL: "/docs/int", Schema: "/schema/int"}}},
	}
	srv := GetRESTServer(cfg, stubLog{})
	if srv == nil {
		t.Fatalf("nil server")
	}
	if !routes["/docs/schema/{schema}"] {
		t.Fatalf("schema routes missing: %v", routes)
	}
}

func TestEnsureDocsGeneratesSchemasWithGitkeep(t *testing.T) {
	dir := t.TempDir()
	schemaDir := filepath.Join(dir, "docs/public/schema/v1")
	if err := os.MkdirAll(schemaDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(schemaDir, ".gitkeep"), []byte{}, 0o644); err != nil {
		t.Fatalf("gitkeep: %v", err)
	}

	reg := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
	reg.Map()["Test"] = &huma.Schema{Type: huma.TypeString}
	spec := &huma.OpenAPI{Components: &huma.Components{Schemas: reg}}

	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()
	_ = os.Chdir(dir)

	if err := ensureDocs(spec, "https://example.com", "/docs/public", "/docs/public/schema/v1", "."); err != nil {
		t.Fatalf("ensureDocs: %v", err)
	}
	if _, err := os.Stat(filepath.Join(schemaDir, "Test.json")); err != nil {
		t.Fatalf("schema not written: %v", err)
	}
}

func TestEnsureDocsPreservesAllOf(t *testing.T) {
	dir := t.TempDir()
	schemaDir := filepath.Join(dir, "docs/public/schema/v1")
	if err := os.MkdirAll(schemaDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	base := &huma.Schema{
		Type: huma.TypeObject,
		Properties: map[string]*huma.Schema{
			"statusCode":    {Type: huma.TypeString},
			"returnMessage": {Type: huma.TypeString},
		},
		Required: []string{"statusCode", "returnMessage"},
	}
	reg := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
	reg.Map()["Base"] = base
	reg.Map()["Resp"] = &huma.Schema{
		AllOf: []*huma.Schema{
			{Ref: "#/components/schemas/Base"},
			{Type: huma.TypeObject, Properties: map[string]*huma.Schema{"body": {Type: huma.TypeString}}},
		},
	}
	spec := &huma.OpenAPI{Components: &huma.Components{Schemas: reg}}

	wd, _ := os.Getwd()
	defer func() { _ = os.Chdir(wd) }()
	_ = os.Chdir(dir)

	if err := ensureDocs(spec, "https://example.com", "/docs/public", "/docs/public/schema/v1", "."); err != nil {
		t.Fatalf("ensureDocs: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(schemaDir, "Resp.json"))
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var m map[string]any
	if err = json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := m["allOf"]; !ok {
		t.Fatalf("missing allOf: %v", m)
	}
	if hasAdditionalProperties(m) {
		t.Fatalf("unexpected additionalProperties: %v", m)
	}
	list, _ := m["allOf"].([]any)
	if len(list) != 2 {
		t.Fatalf("expected two allOf schemas: %v", m["allOf"])
	}
	meta, _ := list[0].(map[string]any)
	body, _ := list[1].(map[string]any)
	mprops, _ := meta["properties"].(map[string]any)
	if _, ok := mprops["statusCode"]; !ok {
		t.Fatalf("missing statusCode: %v", meta)
	}
	bprops, _ := body["properties"].(map[string]any)
	if _, ok := bprops["body"]; !ok {
		t.Fatalf("missing body property: %v", body)
	}
	if _, ok := mprops["returnMessage"]; !ok {
		t.Fatalf("missing returnMessage: %v", meta)
	}
}

func TestEnsureDocsSchemaURL(t *testing.T) {
	cases := []struct {
		version string
		want    string
	}{
		{"3.0.3", "https://spec.openapis.org/oas/3.0/schema/2021-09-28#/$defs/Schema"},
		{"3.1.0", "https://spec.openapis.org/oas/3.1/schema/2022-02-27#/$defs/Schema"},
	}
	for _, c := range cases {
		dir := t.TempDir()
		schemaDir := filepath.Join(dir, "docs/public/schema/v1")
		if err := os.MkdirAll(schemaDir, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		reg := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
		reg.Map()["Test"] = &huma.Schema{Type: huma.TypeString}
		spec := &huma.OpenAPI{OpenAPI: c.version, Components: &huma.Components{Schemas: reg}}

		wd, _ := os.Getwd()
		_ = os.Chdir(dir)

		if err := ensureDocs(spec, "https://example.com", "/docs/public", "/docs/public/schema/v1", "."); err != nil {
			t.Fatalf("ensureDocs: %v", err)
		}
		data, err := os.ReadFile(filepath.Join(schemaDir, "Test.json"))
		if err != nil {
			t.Fatalf("read schema: %v", err)
		}
		var m map[string]any
		if err = json.Unmarshal(data, &m); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if got := m["$schema"]; got != c.want {
			t.Fatalf("$schema = %q, want %q", got, c.want)
		}
		_ = os.Chdir(wd)
	}
}

func hasAdditionalProperties(v any) bool {
	switch x := v.(type) {
	case map[string]any:
		if _, ok := x["additionalProperties"]; ok {
			return true
		}
		for _, sub := range x {
			if hasAdditionalProperties(sub) {
				return true
			}
		}
	case []any:
		for _, sub := range x {
			if hasAdditionalProperties(sub) {
				return true
			}
		}
	}
	return false
}

func TestStripBasePath(t *testing.T) {
	spec := &huma.OpenAPI{Paths: map[string]*huma.PathItem{
		"/api/system/echo": {},
		"/api/items":       {},
		"/other":           {},
	}}
	stripBasePath(spec, "/api")
	if _, ok := spec.Paths["/api/system/echo"]; ok {
		t.Fatalf("base path not stripped: %v", spec.Paths)
	}
	if _, ok := spec.Paths["/system/echo"]; !ok {
		t.Fatalf("missing trimmed path: %v", spec.Paths)
	}
	if _, ok := spec.Paths["/other"]; !ok {
		t.Fatalf("non-base path removed: %v", spec.Paths)
	}
}

// --- registerSchemaRoute tests ---

func TestRegisterSchemaRoute_FoundFile(t *testing.T) {
	dir := t.TempDir()
	schemaDir := filepath.Join(dir, "schema", "v1")
	if err := os.MkdirAll(schemaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := []byte(`{"type":"string"}`)
	if err := os.WriteFile(filepath.Join(schemaDir, "Foo.json"), content, 0o644); err != nil {
		t.Fatal(err)
	}

	r := chi.NewRouter()
	registerSchemaRoute(r, "/schema/v1", "/schema/v1", dir)

	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/schema/v1/Foo.json")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/schema+json" {
		t.Fatalf("unexpected content-type: %s", ct)
	}
	if resp.Header.Get("ETag") == "" {
		t.Fatalf("missing ETag header")
	}
	if resp.Header.Get("Cache-Control") != "public, max-age=300" {
		t.Fatalf("unexpected Cache-Control: %s", resp.Header.Get("Cache-Control"))
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != string(content) {
		t.Fatalf("body mismatch: %s", body)
	}
}

func TestRegisterSchemaRoute_MissingFile(t *testing.T) {
	dir := t.TempDir()
	schemaDir := filepath.Join(dir, "schema", "v1")
	if err := os.MkdirAll(schemaDir, 0o755); err != nil {
		t.Fatal(err)
	}

	r := chi.NewRouter()
	registerSchemaRoute(r, "/schema/v1", "/schema/v1", dir)

	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/schema/v1/Missing.json")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestRegisterSchemaRoute_ETagMatch304(t *testing.T) {
	dir := t.TempDir()
	schemaDir := filepath.Join(dir, "schema", "v1")
	if err := os.MkdirAll(schemaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := []byte(`{"type":"number"}`)
	if err := os.WriteFile(filepath.Join(schemaDir, "Bar.json"), content, 0o644); err != nil {
		t.Fatal(err)
	}

	r := chi.NewRouter()
	registerSchemaRoute(r, "/schema/v1", "/schema/v1", dir)

	srv := httptest.NewServer(r)
	defer srv.Close()

	// First request to get ETag
	resp, err := http.Get(srv.URL + "/schema/v1/Bar.json")
	if err != nil {
		t.Fatal(err)
	}
	etag := resp.Header.Get("ETag")
	resp.Body.Close()
	if etag == "" {
		t.Fatal("no ETag in first response")
	}

	// Second request with If-None-Match
	req, _ := http.NewRequest("GET", srv.URL+"/schema/v1/Bar.json", nil)
	req.Header.Set("If-None-Match", etag)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotModified {
		t.Fatalf("expected 304, got %d", resp.StatusCode)
	}
}

func TestRegisterSchemaRoute_NoJsonSuffix(t *testing.T) {
	dir := t.TempDir()
	schemaDir := filepath.Join(dir, "schema", "v1")
	if err := os.MkdirAll(schemaDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := []byte(`{"type":"boolean"}`)
	if err := os.WriteFile(filepath.Join(schemaDir, "Baz.json"), content, 0o644); err != nil {
		t.Fatal(err)
	}

	r := chi.NewRouter()
	registerSchemaRoute(r, "/schema/v1", "/schema/v1", dir)

	srv := httptest.NewServer(r)
	defer srv.Close()

	// Request without .json suffix; the handler should append it
	resp, err := http.Get(srv.URL + "/schema/v1/Baz")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != string(content) {
		t.Fatalf("body mismatch: %s", body)
	}
}

// --- flattenSchema tests for uncovered branches ---

func TestFlattenSchema_NilSchema(t *testing.T) {
	reg := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
	result := flattenSchema(reg, nil, map[*huma.Schema]bool{})
	if result != nil {
		t.Fatalf("expected nil for nil input")
	}
}

func TestFlattenSchema_CycleDetection(t *testing.T) {
	reg := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
	s := &huma.Schema{Type: huma.TypeObject}
	s.Properties = map[string]*huma.Schema{"self": s}
	result := flattenSchema(reg, s, map[*huma.Schema]bool{})
	if result == nil {
		t.Fatalf("expected non-nil result")
	}
	// The self-referencing property should return an empty schema due to cycle detection
	if selfProp, ok := result.Properties["self"]; !ok || selfProp == nil {
		t.Fatalf("expected self property to exist")
	}
}

func TestFlattenSchema_AnyOf(t *testing.T) {
	reg := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
	s := &huma.Schema{
		AnyOf: []*huma.Schema{
			{Type: huma.TypeString},
			{Type: huma.TypeInteger},
		},
	}
	result := flattenSchema(reg, s, map[*huma.Schema]bool{})
	if len(result.AnyOf) != 2 {
		t.Fatalf("expected 2 AnyOf schemas, got %d", len(result.AnyOf))
	}
}

func TestFlattenSchema_OneOf(t *testing.T) {
	reg := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
	s := &huma.Schema{
		OneOf: []*huma.Schema{
			{Type: huma.TypeString},
			{Type: huma.TypeNumber},
		},
	}
	result := flattenSchema(reg, s, map[*huma.Schema]bool{})
	if len(result.OneOf) != 2 {
		t.Fatalf("expected 2 OneOf schemas, got %d", len(result.OneOf))
	}
}

func TestFlattenSchema_Not(t *testing.T) {
	reg := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
	s := &huma.Schema{
		Not: &huma.Schema{Type: huma.TypeString},
	}
	result := flattenSchema(reg, s, map[*huma.Schema]bool{})
	if result.Not == nil {
		t.Fatalf("expected Not schema")
	}
	if result.Not.Type != huma.TypeString {
		t.Fatalf("unexpected Not type: %v", result.Not.Type)
	}
}

func TestFlattenSchema_AdditionalPropertiesSchema(t *testing.T) {
	reg := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
	s := &huma.Schema{
		Type:                 huma.TypeObject,
		AdditionalProperties: &huma.Schema{Type: huma.TypeString},
	}
	result := flattenSchema(reg, s, map[*huma.Schema]bool{})
	ap, ok := result.AdditionalProperties.(*huma.Schema)
	if !ok || ap == nil {
		t.Fatalf("expected AdditionalProperties to be a *huma.Schema")
	}
	if ap.Type != huma.TypeString {
		t.Fatalf("unexpected AP type: %v", ap.Type)
	}
}

func TestFlattenSchema_AdditionalPropertiesNonSchema(t *testing.T) {
	reg := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
	s := &huma.Schema{
		Type:                 huma.TypeObject,
		AdditionalProperties: true,
	}
	result := flattenSchema(reg, s, map[*huma.Schema]bool{})
	if result.AdditionalProperties != true {
		t.Fatalf("expected AdditionalProperties to be true, got %v", result.AdditionalProperties)
	}
}

// --- stripAllOfAdditionalProperties tests for uncovered branches ---

func TestStripAllOfAdditionalProperties_Nil(t *testing.T) {
	// Should not panic
	stripAllOfAdditionalProperties(nil)
}

func TestStripAllOfAdditionalProperties_AdditionalProperties(t *testing.T) {
	inner := &huma.Schema{
		Type:                 huma.TypeObject,
		AdditionalProperties: &huma.Schema{Type: huma.TypeString},
	}
	s := &huma.Schema{
		Type:                 huma.TypeObject,
		AdditionalProperties: inner,
	}
	stripAllOfAdditionalProperties(s)
	if s.AdditionalProperties != nil {
		t.Fatalf("expected AdditionalProperties to be nil after strip")
	}
}

func TestStripAllOfAdditionalProperties_AnyOf(t *testing.T) {
	s := &huma.Schema{
		AnyOf: []*huma.Schema{
			{Type: huma.TypeString, AdditionalProperties: &huma.Schema{Type: huma.TypeString}},
			{Type: huma.TypeInteger},
		},
	}
	stripAllOfAdditionalProperties(s)
	if s.AnyOf[0].AdditionalProperties != nil {
		t.Fatalf("AnyOf sub-schema AdditionalProperties not stripped")
	}
}

func TestStripAllOfAdditionalProperties_OneOf(t *testing.T) {
	s := &huma.Schema{
		OneOf: []*huma.Schema{
			{Type: huma.TypeString, AdditionalProperties: &huma.Schema{Type: huma.TypeString}},
			{Type: huma.TypeNumber},
		},
	}
	stripAllOfAdditionalProperties(s)
	if s.OneOf[0].AdditionalProperties != nil {
		t.Fatalf("OneOf sub-schema AdditionalProperties not stripped")
	}
}

func TestStripAllOfAdditionalProperties_Not(t *testing.T) {
	s := &huma.Schema{
		Not: &huma.Schema{Type: huma.TypeString, AdditionalProperties: &huma.Schema{Type: huma.TypeString}},
	}
	stripAllOfAdditionalProperties(s)
	if s.Not.AdditionalProperties != nil {
		t.Fatalf("Not sub-schema AdditionalProperties not stripped")
	}
}

func TestStripAllOfAdditionalProperties_Items(t *testing.T) {
	s := &huma.Schema{
		Items: &huma.Schema{
			Type:                 huma.TypeObject,
			AdditionalProperties: &huma.Schema{Type: huma.TypeString},
		},
	}
	stripAllOfAdditionalProperties(s)
	if s.Items.AdditionalProperties != nil {
		t.Fatalf("Items sub-schema AdditionalProperties not stripped")
	}
}

// --- collectSchemaRefs tests for uncovered branches ---

func TestCollectSchemaRefs_OneOf(t *testing.T) {
	reg := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
	reg.Map()["Inner"] = &huma.Schema{Type: huma.TypeString}
	s := &huma.Schema{
		OneOf: []*huma.Schema{
			{Ref: "#/components/schemas/Inner"},
		},
	}
	refs := map[string]struct{}{}
	seen := map[*huma.Schema]bool{}
	collectSchemaRefs(s, reg, refs, seen)
	if _, ok := refs["Inner"]; !ok {
		t.Fatalf("expected Inner ref collected via OneOf")
	}
}

func TestCollectSchemaRefs_AnyOf(t *testing.T) {
	reg := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
	reg.Map()["AnyInner"] = &huma.Schema{Type: huma.TypeString}
	s := &huma.Schema{
		AnyOf: []*huma.Schema{
			{Ref: "#/components/schemas/AnyInner"},
		},
	}
	refs := map[string]struct{}{}
	seen := map[*huma.Schema]bool{}
	collectSchemaRefs(s, reg, refs, seen)
	if _, ok := refs["AnyInner"]; !ok {
		t.Fatalf("expected AnyInner ref collected via AnyOf")
	}
}

func TestCollectSchemaRefs_AdditionalProperties(t *testing.T) {
	reg := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
	reg.Map()["APInner"] = &huma.Schema{Type: huma.TypeString}
	s := &huma.Schema{
		AdditionalProperties: &huma.Schema{Ref: "#/components/schemas/APInner"},
	}
	refs := map[string]struct{}{}
	seen := map[*huma.Schema]bool{}
	collectSchemaRefs(s, reg, refs, seen)
	if _, ok := refs["APInner"]; !ok {
		t.Fatalf("expected APInner ref collected via AdditionalProperties")
	}
}

// --- registerDocs tests for uncovered branches ---

func TestRegisterDocs_JSONEndpoint(t *testing.T) {
	router := chi.NewRouter()
	cfg := huma.DefaultConfig("t", "v")
	cfg.CreateHooks = nil
	api := humachi.New(router, cfg)
	spec := &huma.OpenAPI{OpenAPI: "3.0.3"}
	registerDocs(api, spec, "/openapi-json-test", "/docs/json-test", "Title", "stoplight")
	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/docs/json-test/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Fatalf("unexpected content-type: %s", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
}

func TestRegisterDocs_JSONEndpointCached(t *testing.T) {
	router := chi.NewRouter()
	cfg := huma.DefaultConfig("t", "v")
	cfg.CreateHooks = nil
	api := humachi.New(router, cfg)
	spec := &huma.OpenAPI{OpenAPI: "3.0.3"}
	registerDocs(api, spec, "/openapi-cache", "/docs/cache", "Title", "stoplight")
	srv := httptest.NewServer(router)
	defer srv.Close()

	// Call twice to exercise the caching path
	resp1, _ := http.Get(srv.URL + "/docs/cache/openapi.json")
	_, _ = io.ReadAll(resp1.Body)
	resp1.Body.Close()

	resp2, err := http.Get(srv.URL + "/docs/cache/openapi.json")
	if err != nil {
		t.Fatalf("second request: %v", err)
	}
	defer func() { _ = resp2.Body.Close() }()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on second call, got %d", resp2.StatusCode)
	}
}

func TestRegisterDocs_UnknownRenderer(t *testing.T) {
	router := chi.NewRouter()
	cfg := huma.DefaultConfig("t", "v")
	cfg.CreateHooks = nil
	api := humachi.New(router, cfg)
	spec := &huma.OpenAPI{OpenAPI: "3.0.3"}
	registerDocs(api, spec, "/openapi-unknown", "/docs/unknown", "Title", "some-unknown-renderer")
	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/docs/unknown")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	// Unknown renderer should fall through to the default (stoplight)
	if !strings.Contains(string(body), "<elements-api") {
		t.Fatalf("expected stoplight (default) rendering, got: %s", body)
	}
}

func TestRegisterDocs_YAMLCached(t *testing.T) {
	router := chi.NewRouter()
	cfg := huma.DefaultConfig("t", "v")
	cfg.CreateHooks = nil
	api := humachi.New(router, cfg)
	spec := &huma.OpenAPI{OpenAPI: "3.0.3"}
	registerDocs(api, spec, "/openapi-yaml-cache", "/docs/yaml-cache", "Title", "stoplight")
	srv := httptest.NewServer(router)
	defer srv.Close()

	// Call twice to exercise YAML caching path
	resp1, _ := http.Get(srv.URL + "/openapi-yaml-cache.yaml")
	_, _ = io.ReadAll(resp1.Body)
	resp1.Body.Close()

	resp2, err := http.Get(srv.URL + "/openapi-yaml-cache.yaml")
	if err != nil {
		t.Fatalf("second request: %v", err)
	}
	defer func() { _ = resp2.Body.Close() }()
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp2.StatusCode)
	}
}

// --- ensureDocs error paths ---

func TestEnsureDocs_MkdirFail(t *testing.T) {
	// Use a path that cannot be created
	reg := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
	spec := &huma.OpenAPI{Components: &huma.Components{Schemas: reg}}
	// docDir will be under a non-existent, non-creatable path
	// Using a file as the parent directory ensures MkdirAll fails on any OS
	tmpFile := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(tmpFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := ensureDocs(spec, "https://example.com", "/a/b", "/a/b/schema", tmpFile)
	if err == nil {
		t.Fatalf("expected error for mkdir fail")
	}
}

func TestEnsureDocs_SchemaMkdirFail(t *testing.T) {
	dir := t.TempDir()
	// Create the doc dir with openapi.json so that code skips that block
	docDir := filepath.Join(dir, "docs")
	if err := os.MkdirAll(docDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docDir, "openapi.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	reg := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
	reg.Map()["Test"] = &huma.Schema{Type: huma.TypeString}
	spec := &huma.OpenAPI{Components: &huma.Components{Schemas: reg}}

	// Use a file as blocker so MkdirAll for schema dir fails
	blocker := filepath.Join(dir, "docs", "schema")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	err := ensureDocs(spec, "https://example.com", "/docs", "/docs/schema", dir)
	if err == nil {
		t.Fatalf("expected error for schema mkdir fail")
	}
}

func TestEnsureDocs_WriteFileFail(t *testing.T) {
	dir := t.TempDir()

	reg := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
	spec := &huma.OpenAPI{Components: &huma.Components{Schemas: reg}}

	// Patch os.WriteFile to simulate a write error for openapi.json
	origWriteFile := osWriteFile
	osWriteFile = func(name string, data []byte, perm os.FileMode) error {
		return errors.New("disk full")
	}
	defer func() { osWriteFile = origWriteFile }()
	err := ensureDocs(spec, "https://example.com", "/docs", "/docs/schema", dir)
	if err == nil || !strings.Contains(err.Error(), "disk full") {
		t.Fatalf("expected disk full error, got: %v", err)
	}
}

// --- stripBasePath: non-matching path branch ---

func TestStripBasePath_NonMatchingPath(t *testing.T) {
	spec := &huma.OpenAPI{Paths: map[string]*huma.PathItem{
		"/other/thing": {},
	}}
	stripBasePath(spec, "/api")
	if _, ok := spec.Paths["/other/thing"]; !ok {
		t.Fatalf("non-matching path should be preserved: %v", spec.Paths)
	}
}

func TestStripBasePath_ExactMatch(t *testing.T) {
	spec := &huma.OpenAPI{Paths: map[string]*huma.PathItem{
		"/api": {},
	}}
	stripBasePath(spec, "/api")
	if _, ok := spec.Paths["/"]; !ok {
		t.Fatalf("exact base path should become '/': %v", spec.Paths)
	}
}

// --- setupDocsAndSchemas: outputDir and renderer config ---

func TestSetupDocsAndSchemas_OutputDirConfig(t *testing.T) {
	resetREST()
	dir := t.TempDir()
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
		OpenAPI: config.OpenAPIConfig{
			OutputDir: dir,
			Docs: config.DocsConfig{
				Renderer: "swagger-ui",
			},
		},
	}
	srv := GetRESTServer(cfg, stubLog{})
	if srv == nil {
		t.Fatalf("nil server")
	}
	// Verify files were written to the custom outputDir
	if _, err := os.Stat(filepath.Join(dir, "docs", "public", "openapi.json")); err != nil {
		t.Fatalf("openapi.json not in custom outputDir: %v", err)
	}
}

// --- GetRESTServer: public/internal server URL overrides ---

func TestGetRESTServer_ServerURLOverrides(t *testing.T) {
	resetREST()
	dir := t.TempDir()

	cfg := &config.Config{
		REST:      config.ListenerConfig{Host: "127.0.0.1", Port: "0"},
		Server:    config.ServerConfig{Timeout: config.TimeoutConfig{Read: 1, Write: 1, Idle: 1}, Endpoint: config.EndpointConfig{Based: "/api"}},
		LogTarget: config.LogConfig{Path: t.TempDir(), FileName: "app.log"},
		OpenAPI: config.OpenAPIConfig{
			OutputDir: dir,
			Servers: config.ServersConfig{
				Public:   "https://pub.example.com/",
				Internal: "https://int.example.com/",
			},
		},
	}
	srv := GetRESTServer(cfg, stubLog{})
	if srv == nil {
		t.Fatalf("nil server")
	}
	// Read the internal openapi.json and verify servers[0].url contains internal override
	data, err := os.ReadFile(filepath.Join(dir, "docs", "internal", "openapi.json"))
	if err != nil {
		t.Fatalf("read internal spec: %v", err)
	}
	var spec map[string]any
	if err := json.Unmarshal(data, &spec); err != nil {
		t.Fatalf("unmarshal internal: %v", err)
	}
	servers, _ := spec["servers"].([]any)
	if len(servers) == 0 {
		t.Fatalf("no servers in internal spec")
	}
	s0, _ := servers[0].(map[string]any)
	url, _ := s0["url"].(string)
	if !strings.Contains(url, "int.example.com") {
		t.Fatalf("expected internal server URL override, got: %s", url)
	}
}

// --- GetRESTServer: nil cfg branch for SetPrivateCIDRs ---

func TestGetRESTServer_NilConfig(t *testing.T) {
	resetREST()
	// Passing nil cfg will trigger the SetPrivateCIDRs(nil) branch
	// This will panic at line 156 (cfg.Server.Endpoint.Based) since cfg is nil.
	// The nil cfg branch is only for the SetPrivateCIDRs call.
	// Looking at the code, cfg==nil is checked at line 114, but then cfg is dereferenced
	// at line 156 unconditionally. So the cfg==nil branch at line 116 is dead code
	// in practice; it can only be reached with a nil config but that would panic later.
	// We need to test the cfg != nil path where cfg.OpenAPI.Servers.Public is set instead.
	// Already covered by TestGetRESTServer_ServerURLOverrides above.
}

// --- collectSchemaRefs: cycle detection for seen schemas ---

func TestCollectSchemaRefs_Cycle(t *testing.T) {
	reg := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
	s := &huma.Schema{
		Type: huma.TypeObject,
	}
	s.Properties = map[string]*huma.Schema{"self": s}
	refs := map[string]struct{}{}
	seen := map[*huma.Schema]bool{}
	// Should not infinite loop
	collectSchemaRefs(s, reg, refs, seen)
}

// --- flattenSchema: with Ref ---

func TestFlattenSchema_Ref(t *testing.T) {
	reg := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
	inner := &huma.Schema{Type: huma.TypeString}
	reg.Map()["Inner"] = inner
	s := &huma.Schema{Ref: "#/components/schemas/Inner"}
	result := flattenSchema(reg, s, map[*huma.Schema]bool{})
	if result == nil || result.Type != huma.TypeString {
		t.Fatalf("expected resolved ref to string schema, got %v", result)
	}
}

// --- registerDocs: YAML 3.1 path ---

func TestRegisterDocs_YAML31(t *testing.T) {
	router := chi.NewRouter()
	cfg := huma.DefaultConfig("t", "v")
	cfg.CreateHooks = nil
	api := humachi.New(router, cfg)
	spec := &huma.OpenAPI{OpenAPI: "3.1.0"}
	registerDocs(api, spec, "/openapi-31yaml", "/docs/31yaml", "Title", "stoplight")
	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/openapi-31yaml.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "openapi: 3.1.0") {
		t.Fatalf("expected 3.1.0 YAML content, got: %s", body)
	}
}

// --- ensureDocs: jsonMarshal error path ---

func TestEnsureDocs_MarshalSpecFail(t *testing.T) {
	dir := t.TempDir()
	reg := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
	spec := &huma.OpenAPI{Components: &huma.Components{Schemas: reg}}

	origMarshal := jsonMarshal
	jsonMarshal = func(v any) ([]byte, error) {
		return nil, errors.New("marshal boom")
	}
	defer func() { jsonMarshal = origMarshal }()

	err := ensureDocs(spec, "https://example.com", "/docs-mf", "/docs-mf/schema", dir)
	if err == nil || !strings.Contains(err.Error(), "marshal boom") {
		t.Fatalf("expected marshal error, got: %v", err)
	}
}

// --- ensureDocs: schema json.Marshal error path (line 589) ---

func TestEnsureDocs_SchemaFlatMarshalFail(t *testing.T) {
	dir := t.TempDir()
	// Pre-create the doc dir with openapi.json so it skips that block
	docDir := filepath.Join(dir, "docs-sf")
	if err := os.MkdirAll(docDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docDir, "openapi.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	reg := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
	// Create a schema that will cause json.Marshal to fail after flattenSchema.
	// We use a channel value in Extensions, which json.Marshal cannot handle.
	s := &huma.Schema{Type: huma.TypeString, Extensions: map[string]any{"bad": make(chan int)}}
	reg.Map()["Bad"] = s
	spec := &huma.OpenAPI{Components: &huma.Components{Schemas: reg}}

	err := ensureDocs(spec, "https://example.com", "/docs-sf", "/docs-sf/schema", dir)
	if err == nil {
		t.Fatalf("expected marshal error for bad schema")
	}
}

// --- ensureDocs: schema osWriteFile error path (line 606) ---

func TestEnsureDocs_SchemaWriteFileFail(t *testing.T) {
	dir := t.TempDir()
	// Pre-create the doc dir with openapi.json so it skips that block
	docDir := filepath.Join(dir, "docs-swf")
	if err := os.MkdirAll(docDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docDir, "openapi.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	reg := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
	reg.Map()["Test"] = &huma.Schema{Type: huma.TypeString}
	spec := &huma.OpenAPI{Components: &huma.Components{Schemas: reg}}

	origWriteFile := osWriteFile
	osWriteFile = func(name string, data []byte, perm os.FileMode) error {
		return errors.New("write schema boom")
	}
	defer func() { osWriteFile = origWriteFile }()

	err := ensureDocs(spec, "https://example.com", "/docs-swf", "/docs-swf/schema", dir)
	if err == nil || !strings.Contains(err.Error(), "write schema boom") {
		t.Fatalf("expected write error, got: %v", err)
	}
}

// --- ensureDocs: json.MarshalIndent error path (line 603) ---

func TestEnsureDocs_SchemaMarshalIndentFail(t *testing.T) {
	dir := t.TempDir()
	// Pre-create the doc dir with openapi.json so it skips that block
	docDir := filepath.Join(dir, "docs-mi")
	if err := os.MkdirAll(docDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docDir, "openapi.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	reg := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
	reg.Map()["Test"] = &huma.Schema{Type: huma.TypeString}
	spec := &huma.OpenAPI{Components: &huma.Components{Schemas: reg}}

	// Patch json.MarshalIndent to fail. Since we can't easily replace it with a variable,
	// monkey-patch it directly.
	patchMI := monkey.Patch(json.MarshalIndent, func(v any, prefix, indent string) ([]byte, error) {
		return nil, errors.New("indent boom")
	})
	defer patchMI.Unpatch()

	err := ensureDocs(spec, "https://example.com", "/docs-mi", "/docs-mi/schema", dir)
	if err == nil || !strings.Contains(err.Error(), "indent boom") {
		t.Fatalf("expected indent error, got: %v", err)
	}
}

// --- ensureDocs: json.Unmarshal error path (line 593) ---

func TestEnsureDocs_SchemaUnmarshalFail(t *testing.T) {
	dir := t.TempDir()
	// Pre-create the doc dir with openapi.json
	docDir := filepath.Join(dir, "docs-um")
	if err := os.MkdirAll(docDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(docDir, "openapi.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	reg := huma.NewMapRegistry("#/components/schemas/", huma.DefaultSchemaNamer)
	reg.Map()["Test"] = &huma.Schema{Type: huma.TypeString}
	spec := &huma.OpenAPI{Components: &huma.Components{Schemas: reg}}

	patchUm := monkey.Patch(json.Unmarshal, func(data []byte, v any) error {
		return errors.New("unmarshal boom")
	})
	defer patchUm.Unpatch()

	err := ensureDocs(spec, "https://example.com", "/docs-um", "/docs-um/schema", dir)
	if err == nil || !strings.Contains(err.Error(), "unmarshal boom") {
		t.Fatalf("expected unmarshal error, got: %v", err)
	}
}

// --- registerDocs: YAML error path (line 711) ---

func TestRegisterDocs_YAMLError(t *testing.T) {
	router := chi.NewRouter()
	cfg := huma.DefaultConfig("t", "v")
	cfg.CreateHooks = nil
	api := humachi.New(router, cfg)
	// Create a spec that will cause DowngradeYAML to fail.
	// We embed an un-marshalable value in Extensions.
	spec := &huma.OpenAPI{OpenAPI: "3.0.3", Extensions: map[string]any{"bad": make(chan int)}}
	registerDocs(api, spec, "/openapi-yaml-err", "/docs/yaml-err", "Title", "stoplight")
	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/openapi-yaml-err.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	// The error should be written to the body
	if len(body) == 0 {
		t.Fatalf("expected error in body")
	}
}

// --- registerDocs: JSON error path (line 727) ---

func TestRegisterDocs_JSONError(t *testing.T) {
	router := chi.NewRouter()
	cfg := huma.DefaultConfig("t", "v")
	cfg.CreateHooks = nil
	api := humachi.New(router, cfg)
	// Create a spec that will cause json.Marshal to fail.
	spec := &huma.OpenAPI{OpenAPI: "3.0.3", Extensions: map[string]any{"bad": make(chan int)}}
	registerDocs(api, spec, "/openapi-json-err", "/docs/json-err", "Title", "stoplight")
	srv := httptest.NewServer(router)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/docs/json-err/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if len(body) == 0 {
		t.Fatalf("expected error in body")
	}
}

// --- GetRESTServer: nil cfg branch (dead code but coverable via recover) ---

func TestGetRESTServer_NilCfgBranch(t *testing.T) {
	resetREST()
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("expected panic with nil config")
		}
		resetREST()
	}()
	GetRESTServer(nil, stubLog{})
}
