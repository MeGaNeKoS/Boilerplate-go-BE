package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"project-template/infrastructure/config"
	"project-template/server/rest"
)

func TestErrorFormatMiddleware404(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	rest.Init("APP")

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
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if body["type"] != "about:blank" {
		t.Fatalf("expected type about:blank, got %v", body["type"])
	}
	if body["detail"] != "Resource not found" {
		t.Fatalf("expected detail 'Resource not found', got %v", body["detail"])
	}
}

func TestErrorFormatMiddleware405(t *testing.T) {
	config.Cfg = &config.Config{AppName: "APP"}
	rest.Init("APP")

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
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}
	if body["title"] != "Method Not Allowed" {
		t.Fatalf("expected title 'Method Not Allowed', got %v", body["title"])
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

func TestErrorFormatMiddlewareJsonPassthrough(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
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
