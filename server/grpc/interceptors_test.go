package grpc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"project-template/infrastructure/supervisor"

	"github.com/bouk/monkey"
)

type recLogger struct {
	info string
	err  string
}

func (r *recLogger) DebugF(string, ...interface{})             {}
func (r *recLogger) InfoF(format string, args ...interface{})  { r.info = fmt.Sprintf(format, args...) }
func (r *recLogger) WarnF(string, ...interface{})              {}
func (r *recLogger) ErrorF(format string, args ...interface{}) { r.err = fmt.Sprintf(format, args...) }
func (r *recLogger) FatalF(string, ...interface{})             {}
func (r *recLogger) ParentID() string                          { return "" }
func (r *recLogger) ChildID() string                           { return "" }
func (r *recLogger) CloseLogFile()                             {}

func TestLoggingUnaryServerInterceptor(t *testing.T) {
	lg := &recLogger{}
	interceptor := LoggingUnaryServerInterceptor(lg)
	handler := func(_ context.Context, req interface{}) (interface{}, error) { return "ok", nil }
	resp, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "Foo"}, handler)
	if err != nil || resp != "ok" {
		t.Fatalf("unexpected result %v %v", resp, err)
	}
	if !strings.Contains(lg.info, "Foo") {
		t.Fatalf("info not logged: %s", lg.info)
	}
}

func TestLoggingUnaryServerInterceptorError(t *testing.T) {
	lg := &recLogger{}
	interceptor := LoggingUnaryServerInterceptor(lg)
	wantErr := status.Error(codes.NotFound, "missing")
	handler := func(_ context.Context, req interface{}) (interface{}, error) { return nil, wantErr }
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "Bar"}, handler)
	if err != wantErr {
		t.Fatalf("unexpected error %v", err)
	}
	if !strings.Contains(lg.err, "missing") {
		t.Fatalf("error not logged: %s", lg.err)
	}
}

func TestRecoverUnaryServerInterceptor(t *testing.T) {
	lg := &recLogger{}
	interceptor := RecoverUnaryServerInterceptor(lg)
	handler := func(_ context.Context, req interface{}) (interface{}, error) { panic("boom") }
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "Baz"}, handler)
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Internal {
		t.Fatalf("expected internal error, got %v", err)
	}
	if !strings.Contains(lg.err, "Recovered panic") {
		t.Fatalf("no panic log")
	}
}

func TestRecoverUnaryServerInterceptorNoPanic(t *testing.T) {
	lg := &recLogger{}
	interceptor := RecoverUnaryServerInterceptor(lg)
	handler := func(_ context.Context, req interface{}) (interface{}, error) { return "ok", nil }
	resp, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "Baz"}, handler)
	if err != nil || resp != "ok" {
		t.Fatalf("unexpected result %v %v", resp, err)
	}
	if lg.err != "" {
		t.Fatalf("unexpected error log %s", lg.err)
	}
}

func TestRecoverUnaryServerInterceptorRestartError(t *testing.T) {
	lg := &recLogger{}
	done := make(chan struct{})
	var once sync.Once
	monkey.Patch(supervisor.RequestRestart, func() error {
		once.Do(func() { close(done) })
		return errors.New("bad")
	})
	defer monkey.Unpatch(supervisor.RequestRestart)
	interceptor := RecoverUnaryServerInterceptor(lg)
	handler := func(_ context.Context, req interface{}) (interface{}, error) { panic("boom") }
	_, _ = interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "Baz"}, handler)
	<-done
	time.Sleep(10 * time.Millisecond)
	if !strings.Contains(lg.err, "restart request failed") {
		t.Fatalf("missing restart error log")
	}
}
