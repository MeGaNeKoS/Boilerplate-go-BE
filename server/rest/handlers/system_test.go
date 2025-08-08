package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/danielgtaylor/huma/v2/humatest"

	"project-template/infrastructure/config"
	"project-template/infrastructure/dto/response"
	systemdto "project-template/infrastructure/dto/system"
	"project-template/pkg/code"
	"project-template/server/rest/handlers/resthuma"
)

type mockSystemService struct {
	echo  func(context.Context) (any, *code.Code)
	crash func(context.Context)
	long  func(context.Context, int) (any, *code.Code)
}

func (m mockSystemService) Echo(ctx context.Context) (any, *code.Code) {
	if m.echo != nil {
		return m.echo(ctx)
	}
	return nil, nil
}

func (m mockSystemService) Crash(ctx context.Context) {
	if m.crash != nil {
		m.crash(ctx)
	}
}

func (m mockSystemService) Long(ctx context.Context, s int) (any, *code.Code) {
	if m.long != nil {
		return m.long(ctx, s)
	}
	return nil, nil
}

func decodeSys[T any](t *testing.T, resp *resthuma.Response[*response.GenericResponse[T]]) response.GenericResponse[T] {
	rec := httptest.NewRecorder()
	ctx := humatest.NewContext(nil, httptest.NewRequest(http.MethodGet, "/", nil), rec)
	resp.Body(ctx)
	var out response.GenericResponse[T]
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return out
}

func TestEcho(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	orig := systemService
	systemService = mockSystemService{echo: func(context.Context) (any, *code.Code) { return "ok", nil }}
	defer func() { systemService = orig }()
	resp, err := Echo(context.Background(), &resthuma.Empty{})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	out := decodeSys(t, resp)
	if out.Body != "ok" {
		t.Fatalf("body %#v", out.Body)
	}
}

func TestEchoError(t *testing.T) {
	orig := systemService
	systemService = mockSystemService{echo: func(context.Context) (any, *code.Code) { return nil, code.ErrInternalServerError }}
	defer func() { systemService = orig }()
	if _, err := Echo(context.Background(), &resthuma.Empty{}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestCrash(t *testing.T) {
	orig := systemService
	systemService = mockSystemService{crash: func(context.Context) { panic("boom") }}
	defer func() { systemService = orig }()
	defer func() {
		if recover() == nil {
			t.Fatalf("expected panic")
		}
	}()
	_, _ = Crash(context.Background(), &resthuma.Empty{})
}

func TestLong(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	pid := fmt.Sprintf("%d", os.Getpid())
	orig := systemService
	systemService = mockSystemService{long: func(context.Context, int) (any, *code.Code) { return pid, nil }}
	defer func() { systemService = orig }()
	resp, err := Long(context.Background(), &systemdto.LongInput{Sleep: 0})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	out := decodeSys(t, resp)
	if out.Body != pid {
		t.Fatalf("body %#v", out.Body)
	}
}

func TestLongError(t *testing.T) {
	orig := systemService
	systemService = mockSystemService{long: func(context.Context, int) (any, *code.Code) { return nil, code.ErrInternalServerError }}
	defer func() { systemService = orig }()
	if _, err := Long(context.Background(), &systemdto.LongInput{Sleep: 0}); err == nil {
		t.Fatalf("expected error")
	}
}
