package handlers

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/MeGaNeKoS/neoma/core"

	"project-template/infrastructure/config"
	"project-template/pkg/code"
	systemdto "project-template/server/rest/dto/system"
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

func TestEcho(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	orig := systemService
	systemService = mockSystemService{echo: func(context.Context) (any, *code.Code) { return "ok", nil }}
	defer func() { systemService = orig }()
	resp, err := Echo(context.Background(), &core.Empty{})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	out := resp.Body
	if out.Body != "ok" {
		t.Fatalf("body %#v", out.Body)
	}
}

func TestEchoError(t *testing.T) {
	orig := systemService
	systemService = mockSystemService{echo: func(context.Context) (any, *code.Code) { return nil, code.ErrInternalServerError }}
	defer func() { systemService = orig }()
	if _, err := Echo(context.Background(), &core.Empty{}); err == nil {
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
	_, _ = Crash(context.Background(), &core.Empty{})
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
	out := resp.Body
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

func TestEchoTypeAssertionFailure(t *testing.T) {
	orig := systemService
	systemService = mockSystemService{echo: func(context.Context) (any, *code.Code) { return 42, nil }}
	defer func() { systemService = orig }()
	_, err := Echo(context.Background(), &core.Empty{})
	if err == nil {
		t.Fatalf("expected error for non-string return")
	}
}

func TestLongTypeAssertionFailure(t *testing.T) {
	orig := systemService
	systemService = mockSystemService{long: func(context.Context, int) (any, *code.Code) { return 42, nil }}
	defer func() { systemService = orig }()
	_, err := Long(context.Background(), &systemdto.LongInput{Sleep: 0})
	if err == nil {
		t.Fatalf("expected error for non-string return")
	}
}
