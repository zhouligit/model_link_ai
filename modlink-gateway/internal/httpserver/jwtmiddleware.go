package httpserver

import (
	"net/http"
	"strings"

	"github.com/modlinkcloud/modlink-gateway/internal/auth"
	"github.com/modlinkcloud/modlink-gateway/internal/config"
	"github.com/modlinkcloud/modlink-gateway/internal/shared/envelope"
)

func BearerJWT(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := r.Header.Get("Authorization")
			if !strings.HasPrefix(strings.ToLower(h), "bearer ") {
				envelope.Err(w, r, http.StatusUnauthorized, 40101, "UNAUTHORIZED", nil)
				return
			}
			raw := strings.TrimSpace(h[7:])
			cl, err := auth.ParseAccess(cfg.JWT.Secret, raw)
			if err != nil {
				envelope.Err(w, r, http.StatusUnauthorized, 40102, "INVALID_TOKEN", nil)
				return
			}
			next.ServeHTTP(w, r.WithContext(WithClaims(r.Context(), cl)))
		})
	}
}

// RequireAdmin returns 403 if role != admin
func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cl, ok := ClaimsFrom(r.Context())
		if !ok || cl.Role != "admin" {
			envelope.Err(w, r, http.StatusForbidden, 40301, "FORBIDDEN", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}
