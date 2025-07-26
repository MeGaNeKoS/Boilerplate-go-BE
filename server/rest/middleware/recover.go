package middleware

import (
	"net/http"
	"project-template/infrastructure/supervisor"
	"project-template/infrastructure/utils"
	"project-template/pkg/logger"
	"runtime"
)

// RecoverMiddleware catches panics from handlers, logs the stack trace and
// initiates a graceful shutdown.
func RecoverMiddleware(global logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					stack := make([]byte, 8192)
					n := runtime.Stack(stack, false)

					// Use request logger if possible
					reqLog := utils.GetLoggerFromContext(r.Context())
					if reqLog == nil {
						reqLog = global
					}

					reqLog.ErrorF("Recovered panic: %v\nStack trace:\n%s", err, stack[:n])
					w.Header().Set("X-Recovered-By", "RecoverMiddleware")

					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
					if flusher, ok := w.(http.Flusher); ok {
						flusher.Flush()
					}

					if err := supervisor.RequestRestart(); err != nil {
						reqLog.ErrorF("restart request failed: %v", err)
					}
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
