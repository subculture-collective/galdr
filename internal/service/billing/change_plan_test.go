package billing

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	planmodel "github.com/onnwee/pulse-score/internal/billing"
	"github.com/onnwee/pulse-score/internal/repository"
)

type changePlanSubscriptionStore struct {
	sub       *repository.OrgSubscription
	upserted  *repository.OrgSubscription
}

func (s *changePlanSubscriptionStore) GetByOrg(context.Context, uuid.UUID) (*repository.OrgSubscription, error) {
	return s.sub, nil
}

func (s *changePlanSubscriptionStore) UpsertByOrg(_ context.Context, sub *repository.OrgSubscription) error {
	copy := *sub
	s.upserted = &copy
	return nil
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestPlanChangeImpactUpgradeImmediate(t *testing.T) {
	sub := &repository.OrgSubscription{PlanTier: "growth", BillingCycle: "monthly"}
	resp, err := buildPlanChangeImpact(planmodel.NewCatalog(planmodel.PriceConfig{}), sub, planmodel.TierScale, planmodel.BillingCycleAnnual)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.Action != "upgrade" || resp.Status != "checkout_required" || resp.EffectiveAtPeriodEnd {
		t.Fatalf("unexpected upgrade response: %+v", resp)
	}
	if resp.ProrationCents <= 0 {
		t.Fatalf("expected positive proration estimate, got %d", resp.ProrationCents)
	}
}

func TestPlanChangeImpactDowngradeAtPeriodEnd(t *testing.T) {
	renewal := time.Now().Add(24 * time.Hour).UTC()
	sub := &repository.OrgSubscription{PlanTier: "scale", BillingCycle: "monthly", CurrentPeriodEnd: &renewal}
	resp, err := buildPlanChangeImpact(planmodel.NewCatalog(planmodel.PriceConfig{}), sub, planmodel.TierGrowth, planmodel.BillingCycleMonthly)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.Action != "downgrade" || resp.Status != "scheduled" || !resp.EffectiveAtPeriodEnd {
		t.Fatalf("unexpected downgrade response: %+v", resp)
	}
	if resp.EffectiveAt == nil || !resp.EffectiveAt.Equal(renewal) {
		t.Fatalf("expected downgrade effective at renewal, got %v", resp.EffectiveAt)
	}
}

func TestChangePlanUpgradePaidSubscriptionProratesAndUpdatesImmediately(t *testing.T) {
	orgID := uuid.New()
	userID := uuid.New()
	store := &changePlanSubscriptionStore{sub: &repository.OrgSubscription{
		OrgID:                orgID,
		StripeSubscriptionID: "sub_123",
		StripeCustomerID:     "cus_123",
		PlanTier:             "growth",
		BillingCycle:         "monthly",
		Status:               "active",
	}}
	catalog := planmodel.NewCatalog(planmodel.PriceConfig{
		GrowthMonthly: "price_growth_monthly",
		ScaleMonthly:  "price_scale_monthly",
	})

	var updateForm url.Values
	svc := NewChangePlanService("sk_test", nil, store, catalog)
	svc.httpClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch {
		case req.Method == http.MethodGet && req.URL.Path == "/v1/subscriptions/sub_123":
			return stripeJSON(`{"id":"sub_123","current_period_start":1710000000,"current_period_end":1712592000,"items":{"data":[{"id":"si_123","quantity":1,"price":{"id":"price_growth_monthly"}}]}}`), nil
		case req.Method == http.MethodPost && req.URL.Path == "/v1/subscriptions/sub_123":
			body, err := io.ReadAll(req.Body)
			if err != nil {
				return nil, err
			}
			updateForm, err = url.ParseQuery(string(body))
			if err != nil {
				return nil, err
			}
			return stripeJSON(`{"id":"sub_123","status":"active","current_period_start":1710000000,"current_period_end":1712592000}`), nil
		default:
			t.Fatalf("unexpected Stripe request: %s %s", req.Method, req.URL.Path)
			return nil, nil
		}
	})}

	resp, err := svc.ChangePlan(context.Background(), orgID, userID, ChangePlanRequest{Tier: "scale", Cycle: "monthly"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.Action != "upgrade" || resp.Status != "active" || resp.CheckoutURL != "" || resp.EffectiveAtPeriodEnd {
		t.Fatalf("unexpected upgrade response: %+v", resp)
	}
	if updateForm.Get("proration_behavior") != "create_prorations" {
		t.Fatalf("expected prorations, got form %v", updateForm)
	}
	if updateForm.Get("items[0][id]") != "si_123" || updateForm.Get("items[0][price]") != "price_scale_monthly" {
		t.Fatalf("expected subscription item price update, got form %v", updateForm)
	}
	if store.upserted == nil || store.upserted.PlanTier != "scale" || store.upserted.BillingCycle != "monthly" || store.upserted.Status != "active" {
		t.Fatalf("expected local subscription to grant immediate access, got %+v", store.upserted)
	}
}

func stripeJSON(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}
