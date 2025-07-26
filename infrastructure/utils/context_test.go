package utils

import (
	"context"
	"database/sql"
	"testing"

	"project-template/infrastructure/dto"
)

// dummyLogger satisfies the logger.Logger interface for context tests.
type dummyLogger struct{}

func (dummyLogger) DebugF(string, ...interface{}) {}
func (dummyLogger) InfoF(string, ...interface{})  {}
func (dummyLogger) WarnF(string, ...interface{})  {}
func (dummyLogger) ErrorF(string, ...interface{}) {}
func (dummyLogger) FatalF(string, ...interface{}) {}
func (dummyLogger) ParentID() string              { return "" }
func (dummyLogger) ChildID() string               { return "" }
func (dummyLogger) CloseLogFile()                 {}

type stubRepo struct{}
type stubOut struct{}

func TestContextHelpers(t *testing.T) {
	ctx := context.Background()
	user := &dto.JWTUser{Data: dto.JWTData{UserId: "u1"}}
	ctx = SetUserCtx(ctx, user)
	if got, ok := GetUserCtx(ctx); !ok || got.Data.UserId != "u1" {
		t.Fatalf("GetUserCtx mismatch: %#v %v", got, ok)
	}

	tx := &sql.Tx{}
	ctx = SetTxCtx(ctx, tx)
	if got := GetTxCxt(ctx); got != tx {
		t.Fatalf("GetTxCxt mismatch")
	}

	l := dummyLogger{}
	ctx = SetLoggerToContext(ctx, l)
	if got := GetLoggerFromContext(ctx); got == nil {
		t.Fatalf("expected logger from context")
	}

	ctx = SetRepoCtx(ctx, stubRepo{})
	ctx = SetOutboundCtx(ctx, stubOut{})
	if GetRepoCtx(ctx) == nil {
		t.Fatalf("expected repo")
	}
	if GetOutboundCtx(ctx) == nil {
		t.Fatalf("expected outbound")
	}

	empty := context.Background()
	if u, ok := GetUserCtx(empty); u != nil || ok {
		t.Fatalf("expected no user")
	}
	if GetTxCxt(empty) != nil {
		t.Fatalf("expected nil tx")
	}
	if GetLoggerFromContext(empty) != nil {
		t.Fatalf("expected nil logger")
	}
	if GetRepoCtx(empty) != nil {
		t.Fatalf("expected nil repo")
	}
	if GetOutboundCtx(empty) != nil {
		t.Fatalf("expected nil outbound")
	}
}

func TestTokenCtx(t *testing.T) {
	ctx := context.Background()
	tok := SecureString("tok")
	ctx = SetTokenCtx(ctx, tok)
	if got, ok := GetTokenCtx(ctx); !ok || got != tok {
		t.Fatalf("GetTokenCtx mismatch: %#v %v", got, ok)
	}
	if _, ok := GetTokenCtx(context.Background()); ok {
		t.Fatalf("expected no token")
	}
}
