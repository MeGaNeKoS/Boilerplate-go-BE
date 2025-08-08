package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bouk/monkey"
	"github.com/danielgtaylor/huma/v2"
	humachi "github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"project-template/infrastructure/dto/response"
	"project-template/server/rest/routes"
)

func TestHumaAuthMiddlewareRegistersScheme(t *testing.T) {
	defer monkey.UnpatchAll()
	var authCalled bool
	monkey.Patch(AuthMiddleware, func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authCalled = true
			next.ServeHTTP(w, r)
		})
	})

	r := chi.NewRouter()
	api := humachi.New(r, huma.DefaultConfig("x", "1"))
	g := huma.NewGroup(api, "/")
	mw := HumaAuthMiddleware(g)

	if _, ok := api.OpenAPI().Components.SecuritySchemes[routes.BearerScheme]; !ok {
		t.Fatalf("scheme not registered")
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	ctx := humachi.NewContext(&huma.Operation{Method: http.MethodGet, Path: "/"}, req, w)

	nextCalled := false
	mw(ctx, func(huma.Context) { nextCalled = true })

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
	ctx := humachi.NewContext(&huma.Operation{Method: http.MethodGet, Path: "/"}, req, w)

	t.Run("calls next", func(t *testing.T) {
		called := false
		mw := wrapHTTPMiddleware(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
			})
		})
		mw(ctx, func(huma.Context) { called = true })
		if !called {
			t.Fatal("next not called")
		}
	})

	t.Run("skips next when middleware stops", func(t *testing.T) {
		called := false
		mw := wrapHTTPMiddleware(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
		})
		mw(ctx, func(huma.Context) { called = true })
		if called {
			t.Fatal("next called")
		}
	})
}

func TestRespondVariants(t *testing.T) {
	t.Run("string payload", func(t *testing.T) {
		rec := httptest.NewRecorder()
		resp := &response.HttpResponse{HTTPCode: http.StatusOK, ContentType: "text/plain", RawResponsePayload: "hi"}
		respond(rec, resp)
		if rec.Body.String() != "hi" {
			t.Fatalf("want hi got %q", rec.Body.String())
		}
	})
	t.Run("reader payload", func(t *testing.T) {
		rec := httptest.NewRecorder()
		resp := &response.HttpResponse{HTTPCode: http.StatusOK, ContentType: "text/plain", RawResponsePayload: strings.NewReader("yo")}
		respond(rec, resp)
		if rec.Body.String() != "yo" {
			t.Fatalf("want yo got %q", rec.Body.String())
		}
	})

	t.Run("defaults content type and encodes json", func(t *testing.T) {
		rec := httptest.NewRecorder()
		resp := &response.HttpResponse{HTTPCode: http.StatusOK, RawResponsePayload: map[string]string{"a": "b"}}
		respond(rec, resp)
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Fatalf("want application/json got %q", ct)
		}
		if body := strings.TrimSpace(rec.Body.String()); body != "{\"a\":\"b\"}" {
			t.Fatalf("body %q", body)
		}
	})

	t.Run("nil payload", func(t *testing.T) {
		rec := httptest.NewRecorder()
		respond(rec, &response.HttpResponse{HTTPCode: http.StatusNoContent})
		if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
			t.Fatalf("want application/json got %q", ct)
		}
		if rec.Body.Len() != 0 {
			t.Fatalf("expected empty body got %q", rec.Body.String())
		}
	})

	t.Run("struct payload default case", func(t *testing.T) {
		rec := httptest.NewRecorder()
		resp := &response.HttpResponse{HTTPCode: http.StatusOK, ContentType: "text/plain", RawResponsePayload: struct{ A int }{A: 1}}
		respond(rec, resp)
		if body := strings.TrimSpace(rec.Body.String()); body != "{\"A\":1}" {
			t.Fatalf("body %q", body)
		}
	})
}
