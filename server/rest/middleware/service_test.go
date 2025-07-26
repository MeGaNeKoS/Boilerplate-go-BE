package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServiceMiddlewareNoLogger(t *testing.T) {
	called := false
	handler := ServiceMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if called {
		t.Fatal("next should not be called")
	}
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 got %d", rec.Code)
	}
}
