package transport

import (
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"project-template/infrastructure/config"
	"project-template/pb"
	codepkg "project-template/pkg/code"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type stubHTTPLogger struct{}

func (stubHTTPLogger) Debug(string)                  {}
func (stubHTTPLogger) DebugF(string, ...interface{}) {}
func (stubHTTPLogger) Info(string)                    {}
func (stubHTTPLogger) InfoF(string, ...interface{})  {}
func (stubHTTPLogger) Warn(string)                    {}
func (stubHTTPLogger) WarnF(string, ...interface{})  {}
func (stubHTTPLogger) Error(string)                   {}
func (stubHTTPLogger) ErrorF(string, ...interface{}) {}
func (stubHTTPLogger) Fatal(string)                   {}
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

func TestHTTPOutboundWithHeaders(t *testing.T) {
	t.Run("merge into existing", func(t *testing.T) {
		o := &HTTPOutbound{Headers: map[string]string{"A": "1"}}
		got := o.WithHeaders(map[string]string{"B": "2", "A": "3"})
		if got != o {
			t.Fatal("returned pointer differs")
		}
		if len(o.Headers) != 2 || o.Headers["A"] != "3" || o.Headers["B"] != "2" {
			t.Fatalf("unexpected headers %#v", o.Headers)
		}
		o.WithHeaders(map[string]string{})
		if len(o.Headers) != 2 {
			t.Fatalf("empty merge altered headers %#v", o.Headers)
		}
	})

	t.Run("allocate when nil", func(t *testing.T) {
		o := &HTTPOutbound{}
		o.WithHeaders(map[string]string{"C": "4"})
		if len(o.Headers) != 1 || o.Headers["C"] != "4" {
			t.Fatalf("expected map created, got %#v", o.Headers)
		}
	})
}

func TestHTTPOutboundReplaceHeaders(t *testing.T) {
	o := &HTTPOutbound{Headers: map[string]string{"A": "1"}}
	got := o.ReplaceHeaders(map[string]string{"B": "2"})
	if got != o {
		t.Fatal("returned pointer differs")
	}
	if len(o.Headers) != 1 || o.Headers["B"] != "2" || o.Headers["A"] != "" {
		t.Fatalf("replace failed %#v", o.Headers)
	}
	o.ReplaceHeaders(map[string]string{})
	if o.Headers != nil {
		t.Fatalf("expected headers nil, got %#v", o.Headers)
	}
}

func TestHTTPOutboundWithResponse(t *testing.T) {
	o := &HTTPOutbound{}
	var resp struct{ X int }
	got := o.WithResponse(&resp)
	if got != o {
		t.Fatal("returned pointer differs")
	}
	if o.Response != &resp {
		t.Fatalf("response not set %#v", o.Response)
	}
}

func TestGRPCOutboundWithMethod(t *testing.T) {
	o := &GRPCOutbound{}
	got := o.WithMethod("/svc/m")
	if got != o {
		t.Fatal("returned pointer differs")
	}
	if o.Method != "/svc/m" {
		t.Fatalf("method not set %q", o.Method)
	}
}

func TestGRPCOutboundWithRequest(t *testing.T) {
	o := &GRPCOutbound{}
	req := pb.ItemID{Id: 1}
	got := o.WithRequest(&req)
	if got != o {
		t.Fatal("returned pointer differs")
	}
	if o.Request != &req {
		t.Fatalf("request not set %#v", o.Request)
	}
}

func TestGRPCOutboundWithResponse(t *testing.T) {
	o := &GRPCOutbound{}
	var resp pb.Item
	got := o.WithResponse(&resp)
	if got != o {
		t.Fatal("returned pointer differs")
	}
	if o.Response != &resp {
		t.Fatalf("response not set %#v", o.Response)
	}
}

func TestGRPCOutboundWithMetadata(t *testing.T) {
	t.Run("merge into existing", func(t *testing.T) {
		o := &GRPCOutbound{Metadata: map[string]string{"A": "1"}}
		got := o.WithMetadata(map[string]string{"B": "2", "A": "3"})
		if got != o {
			t.Fatal("returned pointer differs")
		}
		if len(o.Metadata) != 2 || o.Metadata["A"] != "3" || o.Metadata["B"] != "2" {
			t.Fatalf("unexpected metadata %#v", o.Metadata)
		}
		o.WithMetadata(map[string]string{})
		if len(o.Metadata) != 2 {
			t.Fatalf("empty merge altered metadata %#v", o.Metadata)
		}
	})

	t.Run("allocate when nil", func(t *testing.T) {
		o := &GRPCOutbound{}
		o.WithMetadata(map[string]string{"C": "4"})
		if len(o.Metadata) != 1 || o.Metadata["C"] != "4" {
			t.Fatalf("expected map created, got %#v", o.Metadata)
		}
	})
}

func TestGRPCOutboundReplaceMetadata(t *testing.T) {
	o := &GRPCOutbound{Metadata: map[string]string{"A": "1"}}
	got := o.ReplaceMetadata(map[string]string{"B": "2"})
	if got != o {
		t.Fatal("returned pointer differs")
	}
	if len(o.Metadata) != 1 || o.Metadata["B"] != "2" || o.Metadata["A"] != "" {
		t.Fatalf("replace failed %#v", o.Metadata)
	}
	o.ReplaceMetadata(map[string]string{})
	if o.Metadata != nil {
		t.Fatalf("expected metadata nil, got %#v", o.Metadata)
	}
}

func TestBuildURL(t *testing.T) {
	type q struct {
		Q string `url:"q"`
	}
	u, err := buildURL(HTTPOutbound{Host: "https://x", Path: "/p", QueryParam: q{Q: "v"}})
	if err != nil || u != "https://x/p?q=v" {
		t.Fatalf("bad url %s err %v", u, err)
	}
	u, err = buildURL(HTTPOutbound{Host: "https://x", Path: "/p"})
	if err != nil || u != "https://x/p" {
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
	req, err := buildHTTPRequest(HTTPOutbound{Method: http.MethodPost, Headers: map[string]string{"H": "V"}}, "https://x", []byte("b"))
	if err != nil {
		t.Fatalf("error %v", err)
	}
	if req.Method != http.MethodPost || req.URL.String() != "https://x" {
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
	var auth, parent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		parent = r.Header.Get("Parent-Id")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	config.Cfg = &config.Config{Server: config.ServerConfig{Timeout: config.TimeoutConfig{Server: 1}, SkipTLSVerify: true}}

	stub := stubHTTPLogger{}
	o := (&HTTPOutbound{Host: srv.URL, Response: &struct {
		Ok bool `json:"ok"`
	}{}}).
		WithPath("/").
		WithMethod(http.MethodGet).
		WithHeader("Authorization", "Bearer t").
		WithHeader("Parent-Id", stub.ParentID())

	resp, code := o.SendHTTPRequest(stub)
	if code != nil || resp.HTTPCode != 200 {
		t.Fatalf("unexpected code %#v resp %#v", code, resp)
	}
	if auth != "Bearer t" || parent != stub.ParentID() {
		t.Fatalf("expected headers Authorization=Bearer t and Parent-Id=%s, got %s and %s", stub.ParentID(), auth, parent)
	}
}

func TestSendHTTPRequestErrors(t *testing.T) {
	config.Cfg = &config.Config{Server: config.ServerConfig{Timeout: config.TimeoutConfig{Server: 1}, SkipTLSVerify: true}}

	// buildURL error
	o := &HTTPOutbound{Host: "https://x", Path: "/", Method: http.MethodGet, QueryParam: "bad", Response: &struct{}{}}
	_, code := o.SendHTTPRequest(stubHTTPLogger{})
	if code == nil || code.Message != codepkg.ErrCreateRequestUrl.Message {
		t.Fatalf("expected url error")
	}
	if strings.Contains(code.Message, "https://") {
		t.Fatalf("error message leaked URL: %s", code.Message)
	}

	// marshal body error
	o = &HTTPOutbound{Host: "https://x", Path: "/", Method: http.MethodGet, Body: make(chan int), Response: &struct{}{}}
	_, code = o.SendHTTPRequest(stubHTTPLogger{})
	if code == nil || code.Message != codepkg.ErrCreateRequestPayload.Message {
		t.Fatalf("expected payload error")
	}
	if strings.Contains(code.Message, "https://") {
		t.Fatalf("error message leaked URL: %s", code.Message)
	}

	// build request error
	o = &HTTPOutbound{Host: "https://x", Path: "/", Method: "bad\n", Response: &struct{}{}}
	_, code = o.SendHTTPRequest(stubHTTPLogger{})
	if code == nil || code.Message != codepkg.ErrCreateRequestHeaders.Message {
		t.Fatalf("expected request error")
	}
	if strings.Contains(code.Message, "https://") {
		t.Fatalf("error message leaked URL: %s", code.Message)
	}

	// send error
	o = &HTTPOutbound{Host: "https://127.0.0.1:1", Path: "/", Method: http.MethodGet, Response: &struct{}{}}
	_, code = o.SendHTTPRequest(stubHTTPLogger{})
	if code == nil || code.Message != codepkg.ErrFireExternalRequestFailed.Message {
		t.Fatalf("expected send error")
	}
	if strings.Contains(code.Message, "https://") {
		t.Fatalf("error message leaked URL: %s", code.Message)
	}
}

type grpcSvc struct {
	pb.UnimplementedItemServiceServer
	auth, parent string
}

func (s *grpcSvc) GetItem(ctx context.Context, in *pb.ItemID) (*pb.Item, error) {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if vals := md.Get("Authorization"); len(vals) > 0 {
			s.auth = vals[0]
		}
		if vals := md.Get("Parent-Id"); len(vals) > 0 {
			s.parent = vals[0]
		}
	}
	return &pb.Item{Id: in.Id, Name: "n"}, nil
}

func TestGRPCOutboundInvoke(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	srv := grpc.NewServer()
	svc := &grpcSvc{}
	pb.RegisterItemServiceServer(srv, svc)
	go func() {
		err := srv.Serve(lis)
		if err != nil {
			t.Errorf("failed to serve: %v", err)
			return
		}
	}()
	defer srv.Stop()

	ob := GRPCOutbound{
		Host:     lis.Addr().String(),
		Method:   "/pb.ItemService/GetItem",
		Request:  &pb.ItemID{Id: 5},
		Response: &pb.Item{},
		Metadata: map[string]string{
			"Authorization": "Bearer t",
			"Parent-Id":     "p",
		},
	}
	resp, err := ob.Invoke(context.Background(), stubHTTPLogger{})
	if err != nil {
		t.Fatalf("Invoke error: %v", err)
	}
	item := resp.(*pb.Item)
	if item.GetId() != 5 || item.GetName() != "n" {
		t.Fatalf("unexpected item %#v", item)
	}
	if svc.auth != "Bearer t" || svc.parent != "p" {
		t.Fatalf("expected metadata Authorization=Bearer t and Parent-Id=p, got %s and %s", svc.auth, svc.parent)
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

func (badReader) Read(_ []byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func (badReader) Close() error               { return nil }
func TestEmptyResponse(t *testing.T) {
	r := emptyResponse()
	if r.RawResponsePayload != nil || r.HTTPCode != 0 {
		t.Fatalf("unexpected %#v", r)
	}
}
