package middleware

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	neomachi "github.com/MeGaNeKoS/neoma/adapters/neomachi/v5"
	"github.com/MeGaNeKoS/neoma/core"
	"github.com/golang-jwt/jwt/v4"

	"project-template/infrastructure/config"
	"project-template/infrastructure/dto"
	"project-template/infrastructure/utils"
	"project-template/pkg/code"
	"project-template/server/rest"
)

// AuthMiddleware validates the Authorization header and stores the user info in
// the request context.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := utils.GetLoggerFromContext(r.Context())
		if log == nil {
			http.Error(w, "logger missing", http.StatusInternalServerError)
			return
		}

		realm := "example"
		if config.Cfg != nil && config.Cfg.AppName != "" {
			realm = config.Cfg.AppName
		}

		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if token == "" {
			h := http.Header{}
			h.Set("WWW-Authenticate", fmt.Sprintf(`Bearer realm="%s"`, realm))
			writeError(w, code.ErrMissingToken, h)
			return
		}

		var user dto.JWTUser
		if err := utils.GetJWTService().ParseJWT(token, &user); err != nil {
			log.ErrorF("This token invalid because: %v", err)
			h := http.Header{}
			var ve *jwt.ValidationError
			if errors.As(err, &ve) && ve.Errors&jwt.ValidationErrorExpired != 0 {
				h.Set("WWW-Authenticate", fmt.Sprintf(`Bearer realm="%s", error="invalid_token", error_description="The access token expired"`, realm))
				writeError(w, code.ErrTokenExpired, h)
			} else {
				h.Set("WWW-Authenticate", fmt.Sprintf(`Bearer realm="%s", error="invalid_token", error_description="The token is invalid"`, realm))
				writeError(w, code.ErrInvalidToken, h)
			}
			return
		}

		ctx := utils.SetUserCtx(r.Context(), &user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}


// WrapHTTPMiddleware adapts a standard HTTP middleware for use with neoma by
// unwrapping the request/response and invoking the next handler if the middleware
// chain continues.
func WrapHTTPMiddleware(mw func(http.Handler) http.Handler) func(core.Context, func(core.Context)) {
	return func(ctx core.Context, next func(core.Context)) {
		req, w := neomachi.Unwrap(ctx)
		called := false
		mw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			*req = *r
			called = true
			next(ctx)
		})).ServeHTTP(w, req)
		if !called {
			return
		}
	}
}

// writeError writes an error using the standard envelope format.
// Optional headers (e.g. WWW-Authenticate) are applied before writing.
func writeError(w http.ResponseWriter, c *code.Code, headers ...http.Header) {
	for _, h := range headers {
		for k, values := range h {
			for _, v := range values {
				w.Header().Add(k, v)
			}
		}
	}

	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(c.HTTPCode)
	_ = json.NewEncoder(w).Encode(rest.ErrorEnvelope(c.HTTPCode, c.Message))
}
