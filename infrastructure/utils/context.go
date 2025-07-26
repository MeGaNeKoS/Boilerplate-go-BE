package utils

import (
	"context"
	"database/sql"

	"project-template/infrastructure/dto"
	"project-template/pkg/logger"
)

type ctxKey struct {
	name string
}

var (
	txKey        = &ctxKey{name: "tx"}
	userKey      = &ctxKey{name: "user"}
	loggerCtxKey = &ctxKey{name: "request-logger"}
	repoKey      = &ctxKey{name: "repo"}
	outboundKey  = &ctxKey{name: "outbound"}
	tokenKey     = &ctxKey{name: "token"}
)

// SetUserCtx stores the authenticated user into the provided context and
// returns the new context instance.
func SetUserCtx(ctx context.Context, user *dto.JWTUser) context.Context {
	return context.WithValue(ctx, userKey, user)
}

// GetUserCtx retrieves the user information previously stored in the context.
// The boolean return indicates whether the value was present.
func GetUserCtx(ctx context.Context) (*dto.JWTUser, bool) {
	user, ok := ctx.Value(userKey).(*dto.JWTUser)
	return user, ok
}

// SetTxCtx associates an SQL transaction with the context so repository methods
// can reuse it.
func SetTxCtx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, txKey, tx)
}

// GetTxCxt returns the SQL transaction stored in the context, if any.
func GetTxCxt(ctx context.Context) *sql.Tx {
	tx, ok := ctx.Value(txKey).(*sql.Tx)
	if ok {
		return tx
	}
	return nil
}

// SetLoggerToContext attaches a request-scoped logger to the context.
func SetLoggerToContext(ctx context.Context, log logger.Logger) context.Context {
	return context.WithValue(ctx, loggerCtxKey, log)
}

// GetLoggerFromContext retrieves a logger previously stored in the context.
func GetLoggerFromContext(ctx context.Context) logger.Logger {
	if log, ok := ctx.Value(loggerCtxKey).(logger.Logger); ok {
		return log
	}
	return nil
}

// SetRepoCtx stores the repository implementation into the context.
func SetRepoCtx(ctx context.Context, r any) context.Context {
	return context.WithValue(ctx, repoKey, r)
}

// GetRepoCtx retrieves the repository implementation from the context.
func GetRepoCtx(ctx context.Context) any {
	return ctx.Value(repoKey)
}

// SetOutboundCtx stores outbound clients into the context.
func SetOutboundCtx(ctx context.Context, o any) context.Context {
	return context.WithValue(ctx, outboundKey, o)
}

// GetOutboundCtx returns the outbound implementation from the context.
func GetOutboundCtx(ctx context.Context) any {
	return ctx.Value(outboundKey)
}

// SetTokenCtx stores the authorization token in the context.
func SetTokenCtx(ctx context.Context, token SecureString) context.Context {
	return context.WithValue(ctx, tokenKey, token)
}

// GetTokenCtx retrieves the authorization token from the context.
func GetTokenCtx(ctx context.Context) (SecureString, bool) {
	tok, ok := ctx.Value(tokenKey).(SecureString)
	return tok, ok
}
