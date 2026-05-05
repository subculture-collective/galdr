package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

const internalAnalyticsTokenHeader = "X-Internal-Analytics-Token"

// RequireInternalAnalyticsToken protects cross-organization internal analytics APIs.
func RequireInternalAnalyticsToken(token string) func(http.Handler) http.Handler {
	configured := strings.TrimSpace(token)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if configured == "" {
				http.Error(w, `{"error":"internal analytics is not configured"}`, http.StatusForbidden)
				return
			}

			provided := r.Header.Get(internalAnalyticsTokenHeader)
			if provided == "" {
				http.Error(w, `{"error":"missing internal analytics token"}`, http.StatusUnauthorized)
				return
			}
			if subtle.ConstantTimeCompare([]byte(provided), []byte(configured)) != 1 {
				http.Error(w, `{"error":"invalid internal analytics token"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
