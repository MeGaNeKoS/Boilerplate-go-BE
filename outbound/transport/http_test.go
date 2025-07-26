package transport

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"project-template/infrastructure/config"
	"project-template/pb"
	codepkg "project-template/pkg/code"

	"google.golang.org/grpc"
)

type stubHTTPLogger struct{}

func (stubHTTPLogger) DebugF(string, ...interface{}) {}
func (stubHTTPLogger) InfoF(string, ...interface{})  {}
func (stubHTTPLogger) WarnF(string, ...interface{})  {}
func (stubHTTPLogger) ErrorF(string, ...interface{}) {}
func (stubHTTPLogger) FatalF(string, ...interface{}) {}
func (stubHTTPLogger) ParentID() string              { return "p" }
func (stubHTTPLogger) ChildID() string               { return "c" }
func (stubHTTPLogger) CloseLogFile()                 {}

func TestHTTPOutboundCopy(t *testing.T) {
	o := &HTTPOutbound{Path: "/a", Method: http.MethodGet, Headers: map[string]string{"K": "V"}}
	c := o.Copy()
	if c == o {
		t.Fatal("copy returned same pointer")
	}
	c.Headers["K"] = "X"
	if o.Headers["K"] == "X" {
		t.Fatal("copy is not deep")
	}
}

func TestBuildURL(t *testing.T) {
	type q struct {
		Q string `url:"q"`
	}
	u, err := buildURL(HTTPOutbound{Host: "http://x", Path: "/p", QueryParam: q{Q: "v"}})
	if err != nil || u != "http://x/p?q=v" {
		t.Fatalf("bad url %s err %v", u, err)
	}
	u, err = buildURL(HTTPOutbound{Host: "http://x", Path: "/p"})
	if err != nil || u != "http://x/p" {
		t.Fatalf("bad url %s err %v", u, err)
	}
}

func TestMarshalRequestBody(t *testing.T) {
	b, err := marshalRequestBody(nil)
	if err != nil || b != nil {
		t.Fatalf("expected nil body")
	}
	m := map[string]int{"a": 1}
	b, err = marshalRequestBody(m)
	if err != nil || string(b) != `{"a":1}` {
		t.Fatalf("marshal got %s err %v", string(b), err)
	}
}

func TestBuildHTTPRequest(t *testing.T) {
	req, err := buildHTTPRequest(HTTPOutbound{Method: http.MethodPost, Headers: map[string]string{"H": "V"}}, "http://x", []byte("b"))
	if err != nil {
		t.Fatalf("error %v", err)
	}
	if req.Method != http.MethodPost || req.URL.String() != "http://x" {
		t.Fatalf("unexpected request %v", req)
	}
	body, _ := io.ReadAll(req.Body)
	if string(body) != "b" || req.Header.Get("H") != "V" {
		t.Fatalf("headers/body not set")
	}
}

func TestHandleResponse(t *testing.T) {
	resp := &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString(`{"x":1}`))}
	var out struct {
		X int `json:"x"`
	}
	r, err := handleResponse(resp, &out, stubHTTPLogger{})
	if err != nil || out.X != 1 || r.HTTPCode != 200 {
		t.Fatalf("unexpected result %#v err %v", r, err)
	}
}

func TestHTTPOutboundSendHTTPRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	config.Cfg = &config.Config{Server: config.ServerConfig{Timeout: config.TimeoutConfig{Server: 1}, SkipTLSVerify: true}}
	o := &HTTPOutbound{Host: srv.URL, Path: "/", Method: http.MethodGet, Response: &struct {
		Ok bool `json:"ok"`
	}{}}
	resp, code := o.SendHTTPRequest(stubHTTPLogger{})
	if code != nil || resp.HTTPCode != 200 {
		t.Fatalf("unexpected code %#v resp %#v", code, resp)
	}
}

func TestSendHTTPRequestErrors(t *testing.T) {
	config.Cfg = &config.Config{Server: config.ServerConfig{Timeout: config.TimeoutConfig{Server: 1}, SkipTLSVerify: true}}

	// buildURL error
	o := &HTTPOutbound{Host: "http://x", Path: "/", Method: http.MethodGet, QueryParam: "bad", Response: &struct{}{}}
	_, code := o.SendHTTPRequest(stubHTTPLogger{})
	if code == nil || code.InternalCode != codepkg.ErrCreateRequestUrl.InternalCode {
		t.Fatalf("expected url error")
	}

	// marshal body error
	o = &HTTPOutbound{Host: "http://x", Path: "/", Method: http.MethodGet, Body: make(chan int), Response: &struct{}{}}
	_, code = o.SendHTTPRequest(stubHTTPLogger{})
	if code == nil || code.InternalCode != codepkg.ErrCreateRequestPayload.InternalCode {
		t.Fatalf("expected payload error")
	}

	// build request error
	o = &HTTPOutbound{Host: "http://x", Path: "/", Method: "bad\n", Response: &struct{}{}}
	_, code = o.SendHTTPRequest(stubHTTPLogger{})
	if code == nil || code.InternalCode != codepkg.ErrCreateRequestHeaders.InternalCode {
		t.Fatalf("expected request error")
	}

	// send error
	o = &HTTPOutbound{Host: "http://127.0.0.1:1", Path: "/", Method: http.MethodGet, Response: &struct{}{}}
	_, code = o.SendHTTPRequest(stubHTTPLogger{})
	if code == nil || code.InternalCode != codepkg.ErrFireExternalRequestFailed.InternalCode {
		t.Fatalf("expected send error")
	}
}

type grpcSvc struct {
	pb.UnimplementedItemServiceServer
}

func (grpcSvc) GetItem(_ context.Context, in *pb.ItemID) (*pb.Item, error) {
	return &pb.Item{Id: in.Id, Name: "n"}, nil
}

func TestGRPCOutboundInvoke(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := grpc.NewServer()
	pb.RegisterItemServiceServer(srv, grpcSvc{})
	go srv.Serve(lis)
	defer srv.Stop()

	ob := GRPCOutbound{
		Host:     lis.Addr().String(),
		Method:   "/pb.ItemService/GetItem",
		Request:  &pb.ItemID{Id: 5},
		Response: &pb.Item{},
	}
	resp, err := ob.Invoke(context.Background(), stubHTTPLogger{})
	if err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	item := resp.(*pb.Item)
	if item.GetId() != 5 || item.GetName() != "n" {
		t.Fatalf("unexpected item %#v", item)
	}
}

func TestHandleResponseErrors(t *testing.T) {
	r := &http.Response{StatusCode: 200, Body: io.NopCloser(badReader{})}
	var out struct{}
	if _, err := handleResponse(r, &out, stubHTTPLogger{}); err == nil {
		t.Fatal("expected read error")
	}

	r = &http.Response{StatusCode: 200, Body: io.NopCloser(bytes.NewBufferString("{"))}
	if _, err := handleResponse(r, &out, stubHTTPLogger{}); err == nil {
		t.Fatal("expected unmarshal error")
	}
}

type badReader struct{}

func (badReader) Read(p []byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func (badReader) Close() error               { return nil }
func TestEmptyResponse(t *testing.T) {
	r := emptyResponse()
	if r.RawResponsePayload != nil || r.HTTPCode != 0 {
		t.Fatalf("unexpected %#v", r)
	}
}
