package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	"project-template/infrastructure/dto"
	"project-template/infrastructure/dto/response"
	"project-template/infrastructure/utils"
	"project-template/pkg/code"
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

func respond(w http.ResponseWriter, resp *response.HttpResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.HTTPCode)
	if resp.RawResponsePayload != nil {
		_ = json.NewEncoder(w).Encode(resp.RawResponsePayload)
	}
}
