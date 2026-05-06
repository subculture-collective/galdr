package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
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

type mockIntegrationLimitChecker struct {
	decision *billing.LimitDecision
	err      error
	provider string
	called   bool
}

func (m *mockIntegrationLimitChecker) CheckIntegrationLimit(ctx context.Context, orgID uuid.UUID, provider string) (*billing.LimitDecision, error) {
	m.called = true
	m.provider = provider
	return m.decision, m.err
}

func (m *mockFeatureAccessChecker) CanAccess(ctx context.Context, orgID uuid.UUID, featureName string) (*billing.FeatureDecision, error) {
	m.called = true
	return m.decision, m.err
}

func TestRequireFeatureReturnsPaymentRequiredForDeniedFeature(t *testing.T) {
	checker := &mockFeatureAccessChecker{decision: &billing.FeatureDecision{
		Allowed:                false,
		CurrentPlan:            string(billingcatalog.TierFree),
		Feature:                billingcatalog.FeatureFullDashboard,
		RecommendedUpgradeTier: string(billingcatalog.TierGrowth),
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
		CurrentPlan: string(billingcatalog.TierGrowth),
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

func TestRequireIntegrationLimitParamUsesRouteProvider(t *testing.T) {
	checker := &mockIntegrationLimitChecker{decision: &billing.LimitDecision{
		Allowed:                false,
		CurrentPlan:            string(billingcatalog.TierFree),
		LimitType:              billing.LimitIntegration,
		CurrentUsage:           1,
		Limit:                  1,
		RecommendedUpgradeTier: string(billingcatalog.TierGrowth),
	}}
	nextCalled := false
	handler := RequireIntegrationLimitParam(checker, "provider")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	}))

	req := httptest.NewRequest(http.MethodPost, "/integrations/zendesk/connect", nil)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("provider", "zendesk")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	req = req.WithContext(auth.WithOrgID(req.Context(), uuid.New()))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusPaymentRequired {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusPaymentRequired)
	}
	if nextCalled {
		t.Fatal("next handler called for denied integration limit")
	}
	if !checker.called {
		t.Fatal("integration limit checker was not called")
	}
	if checker.provider != "zendesk" {
		t.Fatalf("provider = %q, want zendesk", checker.provider)
	}
}
