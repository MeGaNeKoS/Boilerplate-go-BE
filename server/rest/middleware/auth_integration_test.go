package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	neomachi "github.com/MeGaNeKoS/neoma/adapters/neomachi/v5"
	"github.com/MeGaNeKoS/neoma/core"
	"github.com/MeGaNeKoS/neoma/middleware"
	"github.com/MeGaNeKoS/neoma/neoma"
	"github.com/bouk/monkey"
	"github.com/go-chi/chi/v5"
)

func TestWithSecurityRegistersScheme(t *testing.T) {
	defer monkey.UnpatchAll()
	var authCalled bool
	monkey.Patch(AuthMiddleware, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authCalled = true
			next.ServeHTTP(w, r)
		})
	})

	r := chi.NewRouter()
	cfg := neoma.DefaultConfig("x", "1")
	cfg.CreateHooks = nil
	api := neomachi.New(r, cfg)
	g := middleware.NewGroup(api, "/")
	g.WithSecurity("bearerAuth", &core.SecurityScheme{
		Type:         "http",
		Scheme:       "bearer",
		BearerFormat: "JWT",
	}, WrapHTTPMiddleware(AuthMiddleware))

	if _, ok := api.OpenAPI().Components.SecuritySchemes["bearerAuth"]; !ok {
		t.Fatalf("scheme not registered")
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	ctx := neomachi.NewContext(&core.Operation{Method: http.MethodGet, Path: "/"}, req, w)

	nextCalled := false
	g.Middlewares().Handler(func(c core.Context) { nextCalled = true })(ctx)

	if !authCalled {
		t.Fatal("auth middleware not called")
	}
	if !nextCalled {
		t.Fatal("next not called")
	}
}

func TestWrapHTTPMiddleware(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	ctx := neomachi.NewContext(&core.Operation{Method: http.MethodGet, Path: "/"}, req, w)

	t.Run("calls next", func(t *testing.T) {
		called := false
		mw := WrapHTTPMiddleware(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
			})
		})
		mw(ctx, func(core.Context) { called = true })
		if !called {
			t.Fatal("next not called")
		}
	})

	t.Run("skips next when middleware stops", func(t *testing.T) {
		called := false
		mw := WrapHTTPMiddleware(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
		})
		mw(ctx, func(core.Context) { called = true })
		if called {
			t.Fatal("next called")
		}
	})
}
