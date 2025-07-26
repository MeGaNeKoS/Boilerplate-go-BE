package middleware

import (
	"net/http"
	"strings"

	"project-template/infrastructure/db"
	"project-template/infrastructure/utils"
	"project-template/outbound"
	repo "project-template/repositories"
)

// ServiceMiddleware constructs repository and outbound clients and stores them
// in the request context.
func ServiceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log := utils.GetLoggerFromContext(r.Context())
		if log == nil {
			http.Error(w, "logger missing", http.StatusInternalServerError)
			return
		}
		token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		repoImpl := repo.NewRepository(log, db.GetDBInstance())
		outbounds := outbound.NewOutbound(log)
		ctx := utils.SetRepoCtx(r.Context(), repoImpl)
		ctx = utils.SetOutboundCtx(ctx, outbounds)
		ctx = utils.SetTokenCtx(ctx, utils.SecureString(token))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
