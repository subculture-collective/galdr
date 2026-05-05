package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/auth"
	billingcatalog "github.com/onnwee/pulse-score/internal/billing"
	billing "github.com/onnwee/pulse-score/internal/service/billing"
)

type mockFeatureAccessChecker struct {
	decision *billing.FeatureDecision
	err      error
	called   bool
}

func (m *mockFeatureAccessChecker) CanAccess(ctx context.Context, orgID uuid.UUID, featureName string) (*billing.FeatureDecision, error) {
	m.called = true
	return m.decision, m.err
}

func TestRequireFeatureReturnsPaymentRequiredForDeniedFeature(t *testing.T) {
	checker := &mockFeatureAccessChecker{decision: &billing.FeatureDecision{
		Allowed:                false,
		CurrentPlan:            billingcatalog.TierFree,
		Feature:                billingcatalog.FeatureFullDashboard,
		RecommendedUpgradeTier: billingcatalog.TierGrowth,
	}}
	nextCalled := false
	handler := RequireFeature(checker, billingcatalog.FeatureFullDashboard)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/dashboard/score-distribution", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), uuid.New()))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusPaymentRequired)
	}
	if nextCalled {
		t.Fatal("next handler called for denied feature")
	}
	if !checker.called {
		t.Fatal("feature checker was not called")
	}

	var body map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body["error"] != "feature unavailable on current plan" {
		t.Fatalf("error = %v, want feature unavailable on current plan", body["error"])
	}
	if body["feature"] != billingcatalog.FeatureFullDashboard {
		t.Fatalf("feature = %v, want %s", body["feature"], billingcatalog.FeatureFullDashboard)
	}
	if body["current_plan"] != string(billingcatalog.TierFree) {
		t.Fatalf("current_plan = %v, want %s", body["current_plan"], billingcatalog.TierFree)
	}
	if body["recommended_upgrade_tier"] != string(billingcatalog.TierGrowth) {
		t.Fatalf("recommended_upgrade_tier = %v, want %s", body["recommended_upgrade_tier"], billingcatalog.TierGrowth)
	}
}

func TestRequireFeatureCallsNextForAllowedFeature(t *testing.T) {
	checker := &mockFeatureAccessChecker{decision: &billing.FeatureDecision{
		Allowed:     true,
		CurrentPlan: billingcatalog.TierGrowth,
		Feature:     billingcatalog.FeatureFullDashboard,
	}}
	handler := RequireFeature(checker, billingcatalog.FeatureFullDashboard)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/dashboard/score-distribution", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), uuid.New()))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNoContent)
	}
	if !checker.called {
		t.Fatal("feature checker was not called")
	}
}
