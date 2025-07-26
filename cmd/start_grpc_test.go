//go:build grpc

package cmd

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"project-template/infrastructure/config"
	"project-template/pkg/logger"

	"github.com/bouk/monkey"
	grpcpkg "google.golang.org/grpc"
)

var serverClosedErr = grpcpkg.ErrServerStopped

func patchServer(l logger.Logger, err error) func() {
	patchGet := monkey.Patch(GetGRPCServer, func(*config.Config, logger.Logger) *GRPCServer {
		return &GRPCServer{logger: l}
	})
	patchServe := monkey.Patch((*GRPCServer).StartServer, func(*GRPCServer) error { return err })
	return func() { patchGet.Unpatch(); patchServe.Unpatch() }
}

type stubLogger struct{}

func (stubLogger) DebugF(string, ...interface{}) {}
func (stubLogger) InfoF(string, ...interface{})  {}
func (stubLogger) WarnF(string, ...interface{})  {}
func (stubLogger) ErrorF(string, ...interface{}) {}
func (stubLogger) FatalF(string, ...interface{}) {}
func (stubLogger) ParentID() string              { return "" }
func (stubLogger) ChildID() string               { return "" }
func (stubLogger) CloseLogFile()                 {}

func resetGRPC() {
	grpcInstance = nil
	grpcCreateOnce = sync.Once{}
	grpcShutdownOnce = sync.Once{}
}

func TestStartInitErrorGRPC(t *testing.T) {
	patchParse := monkey.Patch(parseFlags, func(chan struct{}) (string, bool) { return "cfg", false })
	patchInit := monkey.Patch(initDependencies, func(string) (logger.Logger, error) { return nil, errors.New("init") })
	defer patchParse.Unpatch()
	defer patchInit.Unpatch()

	Start(nil)
}

func TestStartServerErrorGRPC(t *testing.T) {
	patchParse := monkey.Patch(parseFlags, func(chan struct{}) (string, bool) { return "cfg", false })
	patchInit := monkey.Patch(initDependencies, func(string) (logger.Logger, error) { return stubLog{}, nil })
	patchGet := monkey.Patch(GetGRPCServer, func(*config.Config, logger.Logger) *GRPCServer { return &GRPCServer{} })
	patchServe := monkey.Patch((*GRPCServer).StartServer, func(*GRPCServer) error { return errors.New("serve") })
	defer patchParse.Unpatch()
	defer patchInit.Unpatch()
	defer patchGet.Unpatch()
	defer patchServe.Unpatch()

	Start(nil)
}

func TestStartSuccessGRPC(t *testing.T) {
	patchParse := monkey.Patch(parseFlags, func(chan struct{}) (string, bool) { return "cfg", false })
	patchInit := monkey.Patch(initDependencies, func(string) (logger.Logger, error) { return stubLog{}, nil })
	patchGet := monkey.Patch(GetGRPCServer, func(*config.Config, logger.Logger) *GRPCServer { return &GRPCServer{} })
	patchServe := monkey.Patch((*GRPCServer).StartServer, func(*GRPCServer) error { return nil })
	defer patchParse.Unpatch()
	defer patchInit.Unpatch()
	defer patchGet.Unpatch()
	defer patchServe.Unpatch()

	Start(nil)
}

func TestGetGRPCServerSingleton(t *testing.T) {
	resetGRPC()
	cfg := &config.Config{}
	s1 := GetGRPCServer(cfg, stubLogger{})
	s2 := GetGRPCServer(cfg, stubLogger{})
	if s1 != s2 {
		t.Fatalf("expected singleton instance")
	}
}

func TestGRPCStartServerError(t *testing.T) {
	resetGRPC()
	cfg := &config.Config{GRPC: config.ListenerConfig{Host: "127.0.0.1", Port: "bad"}}
	srv := GetGRPCServer(cfg, stubLogger{})
	if err := srv.StartServer(); err == nil {
		t.Fatalf("expected start error")
	}
}

func TestGRPCStartAndShutdown(t *testing.T) {
	resetGRPC()
	cfg := &config.Config{GRPC: config.ListenerConfig{Host: "127.0.0.1", Port: "0"}}
	srv := GetGRPCServer(cfg, stubLogger{})
	done := make(chan error, 1)
	go func() { done <- srv.StartServer() }()
	time.Sleep(100 * time.Millisecond)
	if err := GRPCShutdownServer(); err != nil {
		t.Fatalf("shutdown error: %v", err)
	}
	if err := <-done; err != nil && !strings.Contains(err.Error(), "closed network connection") {
		t.Fatalf("serve returned: %v", err)
	}
	if grpcInstance.listener != nil {
		t.Fatalf("listener not nil after shutdown")
	}
}

func TestGRPCCloseListener(t *testing.T) {
	resetGRPC()
	GRPCCloseListener()
}

func TestGRPCShutdownServer(t *testing.T) {
	resetGRPC()
	if err := GRPCShutdownServer(); err != nil {
		t.Fatalf("shutdown error: %v", err)
	}
}

func TestStartLogsServerClosedGRPC(t *testing.T) {
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

func TestStartExitGRPC(t *testing.T) {
	var called bool
	patchParse := monkey.Patch(parseFlags, func(ch chan struct{}) (string, bool) {
		if ch != nil {
			close(ch)
		}
		return "", true
	})
	patchGet := monkey.Patch(GetGRPCServer, func(*config.Config, logger.Logger) *GRPCServer {
		called = true
		return &GRPCServer{}
	})
	defer patchParse.Unpatch()
	defer patchGet.Unpatch()

	done := make(chan struct{})
	Start(done)
	select {
	case <-done:
	default:
		t.Fatalf("done not closed")
	}
	if called {
		t.Fatalf("server should not start")
	}
}

func TestStartDoneChanInitErrGRPC(t *testing.T) {
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

func TestStartDoneChanAfterServeGRPC(t *testing.T) {
	patchParse := monkey.Patch(parseFlags, func(chan struct{}) (string, bool) { return "cfg", false })
	patchInit := monkey.Patch(initDependencies, func(string) (logger.Logger, error) { return stubLog{}, nil })
	patchGet := monkey.Patch(GetGRPCServer, func(*config.Config, logger.Logger) *GRPCServer { return &GRPCServer{} })
	patchServe := monkey.Patch((*GRPCServer).StartServer, func(*GRPCServer) error { return nil })
	defer patchParse.Unpatch()
	defer patchInit.Unpatch()
	defer patchGet.Unpatch()
	defer patchServe.Unpatch()

	done := make(chan struct{})
	Start(done)
	select {
	case <-done:
	default:
		t.Fatalf("done not closed")
	}
}
