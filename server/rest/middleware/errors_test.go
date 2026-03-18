package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"project-template/infrastructure/config"
	"project-template/infrastructure/dto/response"
)

func TestErrorFormatMiddleware404(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	mw := ErrorFormatMiddleware(inner)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("expected application/problem+json, got %q", ct)
	}
	var pd response.ProblemDetail
	if err := json.NewDecoder(rec.Body).Decode(&pd); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if pd.Status != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", pd.Status)
	}
	if pd.Instance != "/missing" {
		t.Fatalf("expected instance /missing, got %q", pd.Instance)
	}
}

func TestErrorFormatMiddleware405(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}

	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Allow", "GET, POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
	})
	mw := ErrorFormatMiddleware(inner)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/items", nil)
	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rec.Code)
	}
	if allow := rec.Header().Get("Allow"); allow != "GET, POST" {
		t.Fatalf("expected Allow 'GET, POST', got %q", allow)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/problem+json" {
		t.Fatalf("expected application/problem+json, got %q", ct)
	}
	var pd response.ProblemDetail
	if err := json.NewDecoder(rec.Body).Decode(&pd); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if pd.Status != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", pd.Status)
	}
	if pd.Instance != "/items" {
		t.Fatalf("expected instance /items, got %q", pd.Instance)
	}
}

func TestErrorFormatMiddleware200Passthrough(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	})
	mw := ErrorFormatMiddleware(inner)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/plain" {
		t.Fatalf("expected text/plain, got %q", ct)
	}
	if rec.Body.String() != "hello" {
		t.Fatalf("expected body 'hello', got %q", rec.Body.String())
	}
}

func TestStatusInterceptWriterUnwrap(t *testing.T) {
	rec := httptest.NewRecorder()
	iw := &statusInterceptWriter{ResponseWriter: rec}
	if got := iw.Unwrap(); got != rec {
		t.Fatalf("Unwrap returned %v, want %v", got, rec)
	}
}

func TestErrorFormatMiddlewareProblemJsonPassthrough(t *testing.T) {
	// If the response already has application/problem+json, don't intercept.
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"title": "custom", "status": 404})
	})
	mw := ErrorFormatMiddleware(inner)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	mw.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if body["title"] != "custom" {
		t.Fatalf("expected original body, got %v", body)
	}
}
