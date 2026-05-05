package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequireInternalAnalyticsToken(t *testing.T) {
	tests := []struct {
		name       string
		configured string
		header     string
		wantStatus int
	}{
		{name: "missing configured token blocks", configured: "", header: "secret", wantStatus: http.StatusForbidden},
		{name: "missing header blocks", configured: "secret", header: "", wantStatus: http.StatusUnauthorized},
		{name: "wrong header blocks", configured: "secret", header: "wrong", wantStatus: http.StatusForbidden},
		{name: "matching token passes", configured: "secret", header: "secret", wantStatus: http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := RequireInternalAnalyticsToken(tt.configured)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			req := httptest.NewRequest(http.MethodGet, "/api/v1/billing/usage/analytics", nil)
			if tt.header != "" {
				req.Header.Set("X-Internal-Analytics-Token", tt.header)
			}
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d", tt.wantStatus, rr.Code)
			}
		})
	}
}
