package middleware

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	humachi "github.com/danielgtaylor/huma/v2/adapters/humachi"

	"project-template/infrastructure/dto"
	"project-template/infrastructure/dto/response"
	"project-template/infrastructure/utils"
	"project-template/pkg/code"
	"project-template/server/rest/routes"
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

		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		var user dto.JWTUser
		if err := utils.GetJWTService().ParseJWT(token, &user); err != nil {
			log.ErrorF("This token invalid because: %v", err)
			respond(w, utils.GenerateErrorResponse(code.ErrTokenExpired))
			return
		}

		ctx := utils.SetUserCtx(r.Context(), &user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// HumaAuthMiddleware returns a Huma-compatible middleware that wraps the native
// AuthMiddleware and registers the security scheme on the provided group.
func HumaAuthMiddleware(g *huma.Group) func(huma.Context, func(huma.Context)) {
	api := g.API
	comps := api.OpenAPI().Components
	if comps.SecuritySchemes == nil {
		comps.SecuritySchemes = map[string]*huma.SecurityScheme{}
	}
	if _, ok := comps.SecuritySchemes[routes.BearerScheme]; !ok {
		comps.SecuritySchemes[routes.BearerScheme] = &huma.SecurityScheme{
			Type:         "http",
			Scheme:       "bearer",
			BearerFormat: "JWT",
		}
	}
	routes.UseSecurity(g, routes.BearerScheme)
	return wrapHTTPMiddleware(AuthMiddleware)
}

// wrapHTTPMiddleware adapts a standard HTTP middleware for use with Huma by
// unwrapping the request/response and invoking the next handler if the middleware
// chain continues.
func wrapHTTPMiddleware(mw func(http.Handler) http.Handler) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		req, w := humachi.Unwrap(ctx)
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

func respond(w http.ResponseWriter, resp *response.HttpResponse) {
	ct := resp.ContentType
	if ct == "" {
		ct = "application/json"
	}
	w.Header().Set("Content-Type", ct)
	w.WriteHeader(resp.HTTPCode)
	if resp.RawResponsePayload == nil {
		return
	}
	if strings.Contains(ct, "json") {
		_ = json.NewEncoder(w).Encode(resp.RawResponsePayload)
		return
	}
	switch p := resp.RawResponsePayload.(type) {
	case []byte:
		_, _ = w.Write(p)
	case string:
		_, _ = w.Write([]byte(p))
	case io.Reader:
		_, _ = io.Copy(w, p)
	default:
		_ = json.NewEncoder(w).Encode(p)
	}
}
