package middleware

import (
	"net/http"
	"strings"

	"project-template/pkg/code"
)

// statusInterceptWriter intercepts 404/405 from chi's default handlers to
// replace plain-text bodies with Problem Details JSON.
type statusInterceptWriter struct {
	http.ResponseWriter
	status      int
	intercepted bool
}

func (w *statusInterceptWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	if statusCode == http.StatusNotFound || statusCode == http.StatusMethodNotAllowed {
		// Skip if the response already has a JSON content type (e.g. from neoma).
		if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "json") {
			w.intercepted = true
			return
		}
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *statusInterceptWriter) Write(b []byte) (int, error) {
	if w.intercepted {
		return len(b), nil
	}
	return w.ResponseWriter.Write(b)
}

func (w *statusInterceptWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// ErrorFormatMiddleware replaces chi's default plain-text 404 and 405 responses
// with Problem Details JSON. The Allow header on 405 responses is preserved.
func ErrorFormatMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		iw := &statusInterceptWriter{ResponseWriter: w}
		next.ServeHTTP(iw, r)

		if !iw.intercepted {
			return
		}

		var errCode *code.Code
		if iw.status == http.StatusNotFound {
			errCode = code.ErrNotFound
		} else {
			errCode = code.ErrMethodNotAllowed
		}

		writeError(w, errCode)
	})
}
