package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/auth"
	"github.com/onnwee/pulse-score/internal/repository"
	core "github.com/onnwee/pulse-score/internal/service"
	billing "github.com/onnwee/pulse-score/internal/service/billing"
)

type mockBillingCheckoutService struct {
	createFn func(ctx context.Context, orgID, userID uuid.UUID, req billing.CreateCheckoutSessionRequest) (*billing.CreateCheckoutSessionResponse, error)
}

func (m *mockBillingCheckoutService) CreateCheckoutSession(ctx context.Context, orgID, userID uuid.UUID, req billing.CreateCheckoutSessionRequest) (*billing.CreateCheckoutSessionResponse, error) {
	return m.createFn(ctx, orgID, userID, req)
}

type mockBillingPortalService struct {
	portalFn func(ctx context.Context, orgID uuid.UUID) (*billing.PortalSessionResponse, error)
	cancelFn func(ctx context.Context, orgID uuid.UUID) error
}

func (m *mockBillingPortalService) CreatePortalSession(ctx context.Context, orgID uuid.UUID) (*billing.PortalSessionResponse, error) {
	return m.portalFn(ctx, orgID)
}

func (m *mockBillingPortalService) CancelAtPeriodEnd(ctx context.Context, orgID uuid.UUID) error {
	return m.cancelFn(ctx, orgID)
}

type mockBillingSubscriptionService struct {
	getFn func(ctx context.Context, orgID uuid.UUID) (*billing.SubscriptionSummary, error)
}

func (m *mockBillingSubscriptionService) GetSubscriptionSummary(ctx context.Context, orgID uuid.UUID) (*billing.SubscriptionSummary, error) {
	return m.getFn(ctx, orgID)
}

type mockBillingUsageService struct {
	getFn       func(ctx context.Context, orgID uuid.UUID) (*billing.UsageSummary, error)
	aggregateFn func(ctx context.Context) (*billing.AggregateUsageAnalytics, error)
}

func (m *mockBillingUsageService) GetUsage(ctx context.Context, orgID uuid.UUID) (*billing.UsageSummary, error) {
	return m.getFn(ctx, orgID)
}

func (m *mockBillingUsageService) GetAggregateAnalytics(ctx context.Context) (*billing.AggregateUsageAnalytics, error) {
	return m.aggregateFn(ctx)
}

type mockBillingPlanChangeService struct {
	changeFn func(ctx context.Context, orgID, userID uuid.UUID, req billing.ChangePlanRequest) (*billing.ChangePlanResponse, error)
}

func (m *mockBillingPlanChangeService) ChangePlan(ctx context.Context, orgID, userID uuid.UUID, req billing.ChangePlanRequest) (*billing.ChangePlanResponse, error) {
	return m.changeFn(ctx, orgID, userID, req)
}

type mockBillingWebhookService struct {
	handleFn func(ctx context.Context, payload []byte, sigHeader string) error
}

func (m *mockBillingWebhookService) HandleEvent(ctx context.Context, payload []byte, sigHeader string) error {
	return m.handleFn(ctx, payload, sigHeader)
}

func TestBillingCreateCheckout_Unauthorized(t *testing.T) {
	h := NewBillingHandler(&mockBillingCheckoutService{}, &mockBillingPortalService{}, &mockBillingSubscriptionService{}, &mockBillingUsageService{}, &mockBillingPlanChangeService{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/billing/checkout", strings.NewReader(`{"tier":"growth"}`))
	rr := httptest.NewRecorder()

	h.CreateCheckout(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestBillingCreateCheckout_Success(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	h := NewBillingHandler(
		&mockBillingCheckoutService{createFn: func(ctx context.Context, gotOrgID, gotUserID uuid.UUID, req billing.CreateCheckoutSessionRequest) (*billing.CreateCheckoutSessionResponse, error) {
			if gotOrgID != orgID || gotUserID != userID {
				t.Fatalf("unexpected org/user ids passed")
			}
			return &billing.CreateCheckoutSessionResponse{URL: "https://checkout.stripe.test"}, nil
		}},
		&mockBillingPortalService{},
		&mockBillingSubscriptionService{},
		&mockBillingUsageService{},
		&mockBillingPlanChangeService{},
	)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/billing/checkout", strings.NewReader(`{"tier":"growth","annual":true}`))
	req = req.WithContext(auth.WithUserID(auth.WithOrgID(req.Context(), orgID), userID))
	rr := httptest.NewRecorder()

	h.CreateCheckout(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestBillingGetSubscription_Success(t *testing.T) {
	orgID := uuid.New()
	h := NewBillingHandler(
		&mockBillingCheckoutService{},
		&mockBillingPortalService{},
		&mockBillingSubscriptionService{getFn: func(context.Context, uuid.UUID) (*billing.SubscriptionSummary, error) {
			return &billing.SubscriptionSummary{Tier: "free", Status: "free"}, nil
		}},
		&mockBillingUsageService{},
		&mockBillingPlanChangeService{},
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/billing/subscription", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	rr := httptest.NewRecorder()

	h.GetSubscription(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestBillingGetUsage_Success(t *testing.T) {
	orgID := uuid.New()
	h := NewBillingHandler(
		&mockBillingCheckoutService{},
		&mockBillingPortalService{},
		&mockBillingSubscriptionService{},
		&mockBillingUsageService{getFn: func(ctx context.Context, gotOrgID uuid.UUID) (*billing.UsageSummary, error) {
			if gotOrgID != orgID {
				t.Fatalf("expected org id %s, got %s", orgID, gotOrgID)
			}
			return &billing.UsageSummary{CustomerCount: billing.UsageCounter{Used: 9, Limit: 10}}, nil
		}},
		&mockBillingPlanChangeService{},
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/billing/usage", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	rr := httptest.NewRecorder()

	h.GetUsage(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestBillingGetUsageAnalytics_Success(t *testing.T) {
	h := NewBillingHandler(
		&mockBillingCheckoutService{},
		&mockBillingPortalService{},
		&mockBillingSubscriptionService{},
		&mockBillingUsageService{aggregateFn: func(ctx context.Context) (*billing.AggregateUsageAnalytics, error) {
			return &billing.AggregateUsageAnalytics{Metrics: map[string]repository.UsageMetricAggregate{
				billing.UsageMetricCustomers: {Metric: billing.UsageMetricCustomers, OrgCount: 2, Total: 75, Average: 37.5, Maximum: 50},
			}}, nil
		}},
		&mockBillingPlanChangeService{},
	)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/billing/usage/analytics", nil)
	rr := httptest.NewRecorder()

	h.GetUsageAnalytics(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), `"org_count":2`) {
		t.Fatalf("expected aggregate org count response, got %s", rr.Body.String())
	}
}

func TestBillingCancelSubscription_Success(t *testing.T) {
	orgID := uuid.New()
	h := NewBillingHandler(
		&mockBillingCheckoutService{},
		&mockBillingPortalService{cancelFn: func(context.Context, uuid.UUID) error { return nil }},
		&mockBillingSubscriptionService{},
		&mockBillingUsageService{},
		&mockBillingPlanChangeService{},
	)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/billing/cancel", nil)
	req = req.WithContext(auth.WithOrgID(req.Context(), orgID))
	rr := httptest.NewRecorder()

	h.CancelSubscription(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestBillingChangePlan_Success(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	h := NewBillingHandler(
		&mockBillingCheckoutService{},
		&mockBillingPortalService{},
		&mockBillingSubscriptionService{},
		&mockBillingUsageService{},
		&mockBillingPlanChangeService{changeFn: func(ctx context.Context, gotOrgID, gotUserID uuid.UUID, req billing.ChangePlanRequest) (*billing.ChangePlanResponse, error) {
			if gotOrgID != orgID || gotUserID != userID {
				t.Fatalf("unexpected org/user ids passed")
			}
			if req.Tier != "scale" || req.Cycle != "annual" {
				t.Fatalf("unexpected request: %+v", req)
			}
			return &billing.ChangePlanResponse{Action: "upgrade", Status: "checkout_required", CheckoutURL: "https://checkout.stripe.test", ProrationCents: 1250}, nil
		}},
	)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/billing/change-plan", strings.NewReader(`{"tier":"scale","cycle":"annual"}`))
	req = req.WithContext(auth.WithUserID(auth.WithOrgID(req.Context(), orgID), userID))
	rr := httptest.NewRecorder()

	h.ChangePlan(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestBillingWebhook_MissingSignature(t *testing.T) {
	h := NewWebhookStripeBillingHandler(&mockBillingWebhookService{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe-billing", strings.NewReader("{}"))
	rr := httptest.NewRecorder()

	h.HandleWebhook(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestBillingWebhook_ValidationError(t *testing.T) {
	h := NewWebhookStripeBillingHandler(&mockBillingWebhookService{handleFn: func(context.Context, []byte, string) error {
		return &core.ValidationError{Field: "signature", Message: "invalid webhook signature"}
	}})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/stripe-billing", strings.NewReader("{}"))
	req.Header.Set("Stripe-Signature", "bad-signature")
	rr := httptest.NewRecorder()

	h.HandleWebhook(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}
