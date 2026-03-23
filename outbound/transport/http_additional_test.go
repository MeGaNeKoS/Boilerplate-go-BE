package transport

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/bouk/monkey"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"project-template/infrastructure/config"
	"project-template/infrastructure/dto/response"
	"project-template/pb"
	codepkg "project-template/pkg/code"
	"project-template/pkg/logger"
)

// stubLogger is a minimal logger implementation for tests.
type stubLogger struct{ parent string }

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
func (s stubLogger) ParentID() string            { return s.parent }
func (stubLogger) ChildID() string               { return "" }
func (stubLogger) CloseLogFile()                 {}

type errCloseReader struct{ io.Reader }

func (errCloseReader) Close() error { return fmt.Errorf("close") }

// ensure stubLogger implements logger.Logger
var _ logger.Logger = stubLogger{}

func TestHandleResponseNon2xx(t *testing.T) {
	resp := &http.Response{StatusCode: 404, Body: io.NopCloser(bytes.NewBufferString(`{"x":1}`))}
	var out struct {
		X int `json:"x"`
	}
	r, err := handleResponse(resp, &out, stubLogger{})
	if err != nil || r.HTTPCode != 404 || out.X != 1 {
		t.Fatalf("unexpected %#v err %v", r, err)
	}
}

func TestSendHTTPRequestHandleResponseErrorAndCloseWarn(t *testing.T) {
	config.Cfg = &config.Config{Server: config.ServerConfig{Timeout: config.TimeoutConfig{Server: 1}, SkipTLSVerify: true}}
	o := &HTTPOutbound{Host: "https://example", Path: "/", Method: http.MethodGet, Response: &struct{}{}}

	monkey.Patch(sendHTTPRequest, func(_ *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Body: errCloseReader{strings.NewReader("{}")}}, nil
	})
	defer monkey.Unpatch(sendHTTPRequest)

	monkey.Patch(handleResponse, func(_ *http.Response, _ interface{}, _ logger.Logger) (response.HttpResponse, error) {
		return response.HttpResponse{}, errors.New("bad")
	})
	defer monkey.Unpatch(handleResponse)

	_, code := o.SendHTTPRequest(stubLogger{})
	if code == nil || code.Message != codepkg.ErrReadExternalRequestFailed.Message {
		t.Fatalf("expected read external error, got %#v", code)
	}
}

func TestGRPCOutboundInvokeConnError(t *testing.T) {
	monkey.Patch(grpc.NewClient, func(string, ...grpc.DialOption) (*grpc.ClientConn, error) {
		return nil, errors.New("dial")
	})
	defer monkey.Unpatch(grpc.NewClient)

	ob := GRPCOutbound{Host: "bad", Method: "/none", Response: &pb.Item{}}
	if _, err := ob.Invoke(context.Background(), stubLogger{}); err == nil {
		t.Fatal("expected dial error")
	}
}

func TestGRPCOutboundInvokeInvokeError(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := grpc.NewServer()
	go func() { _ = srv.Serve(lis) }()
	defer srv.GracefulStop()

	original := grpc.NewClient
	conn, errDial := original(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if errDial != nil {
		t.Fatal(errDial)
	}
	var gotMD metadata.MD
	monkey.PatchInstanceMethod(reflect.TypeOf(conn), "Invoke", func(_ *grpc.ClientConn, ctx context.Context, _ string, _ interface{}, _ interface{}, _ ...grpc.CallOption) error {
		gotMD, _ = metadata.FromOutgoingContext(ctx)
		return errors.New("invoke")
	})
	monkey.PatchInstanceMethod(reflect.TypeOf(conn), "Close", func(_ *grpc.ClientConn) error {
		return errors.New("close")
	})
	monkey.Patch(grpc.NewClient, func(string, ...grpc.DialOption) (*grpc.ClientConn, error) {
		return conn, nil
	})
	defer func() {
		monkey.Unpatch(grpc.NewClient)
		monkey.UnpatchAll()
	}()

	ob := GRPCOutbound{
		Host:     lis.Addr().String(),
		Method:   "/pb.ItemService/GetItem",
		Request:  &pb.ItemID{Id: 1},
		Response: &pb.Item{},
		Metadata: map[string]string{"k": "v"},
	}
	_, err = ob.Invoke(context.Background(), stubLogger{})
	if err == nil || !strings.Contains(err.Error(), "invoke") {
		t.Fatalf("expected invoke error, got %v", err)
	}
	if got, want := gotMD.Get("k"), []string{"v"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("metadata not passed: %v", got)
	}
}
