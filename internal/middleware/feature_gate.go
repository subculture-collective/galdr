package middleware

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/auth"
	billing "github.com/onnwee/pulse-score/internal/service/billing"
)

type featureAccessChecker interface {
	CanAccess(ctx context.Context, orgID uuid.UUID, featureName string) (*billing.FeatureDecision, error)
}

// RequireFeature enforces plan feature access for the current org.
func RequireFeature(limitsSvc featureAccessChecker, featureName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			orgID, ok := auth.GetOrgID(r.Context())
			if !ok {
				writeFeatureGateJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}

			decision, err := limitsSvc.CanAccess(r.Context(), orgID, featureName)
			if err != nil {
				writeFeatureGateJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
				return
			}

			if !decision.Allowed {
				writeFeatureGateJSON(w, http.StatusPaymentRequired, map[string]any{
					"error":                    "feature unavailable on current plan",
					"current_plan":             decision.CurrentPlan,
					"feature":                  decision.Feature,
					"recommended_upgrade_tier": decision.RecommendedUpgradeTier,
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireIntegrationLimit enforces integration connection limits for the current org.
func RequireIntegrationLimit(limitsSvc *billing.LimitsService, provider string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			orgID, ok := auth.GetOrgID(r.Context())
			if !ok {
				writeFeatureGateJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}

			decision, err := limitsSvc.CheckIntegrationLimit(r.Context(), orgID, provider)
			if err != nil {
				writeFeatureGateJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
				return
			}

			if !decision.Allowed {
				writeFeatureGateJSON(w, http.StatusPaymentRequired, map[string]any{
					"error":                    "plan limit reached",
					"current_plan":             decision.CurrentPlan,
					"limit_type":               decision.LimitType,
					"current_usage":            decision.CurrentUsage,
					"limit":                    decision.Limit,
					"recommended_upgrade_tier": decision.RecommendedUpgradeTier,
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireCustomerLimit enforces customer creation limits for the current org.
func RequireCustomerLimit(limitsSvc *billing.LimitsService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			orgID, ok := auth.GetOrgID(r.Context())
			if !ok {
				writeFeatureGateJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}

			decision, err := limitsSvc.CheckCustomerLimit(r.Context(), orgID)
			if err != nil {
				writeFeatureGateJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
				return
			}

			if !decision.Allowed {
				writeFeatureGateJSON(w, http.StatusPaymentRequired, map[string]any{
					"error":                    "plan limit reached",
					"current_plan":             decision.CurrentPlan,
					"limit_type":               decision.LimitType,
					"current_usage":            decision.CurrentUsage,
					"limit":                    decision.Limit,
					"recommended_upgrade_tier": decision.RecommendedUpgradeTier,
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func writeFeatureGateJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
