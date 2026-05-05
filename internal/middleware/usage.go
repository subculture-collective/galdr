package middleware

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/auth"
)

type apiUsageRecorder interface {
	RecordAPIRequest(ctx context.Context, orgID uuid.UUID) error
}

// TrackAPIUsage counts authenticated API requests per organization.
func TrackAPIUsage(recorder apiUsageRecorder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if orgID, ok := auth.GetOrgID(r.Context()); ok {
				if err := recorder.RecordAPIRequest(r.Context(), orgID); err != nil {
					slog.Error("track api usage", "org_id", orgID, "error", err)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
