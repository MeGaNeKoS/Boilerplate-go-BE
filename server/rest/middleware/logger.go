package middleware

import (
	"net/http"
	"project-template/infrastructure/config"
	"project-template/infrastructure/utils"
	"project-template/pkg/logger"
)

// LoggerMiddleware injects a request-scoped logger into the request context.
func LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parentId := r.Header.Get("Parent-Id")
		if parentId == "" {
			identifier, err := utils.UniqueIdByTime(86400)
			if err != nil {
				http.Error(w, "ID generation failed", http.StatusInternalServerError)
				return
			}
			parentId = "-" + identifier
		}

		childId, err := utils.UniqueIdByTime(86400)
		if err != nil {
			http.Error(w, "ID generation failed", http.StatusInternalServerError)
			return
		}

		log, err := logger.NewLogger(config.Cfg.LogTarget, parentId, childId)
		if err != nil {
			http.Error(w, "Logger init failed", http.StatusInternalServerError)
			return
		}

		ctx := utils.SetLoggerToContext(r.Context(), log)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
