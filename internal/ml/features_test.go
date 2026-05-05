package ml

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

func TestExtractCustomerFeaturesNormalizesChurnSignals(t *testing.T) {
	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	customerID := uuid.New()
	orgID := uuid.New()
	firstSeen := now.AddDate(0, -6, 0)
	lastSeen := now.Add(-6 * time.Hour)

	features := ExtractCustomerFeatures(FeatureInput{
		Now: now,
		Customer: &repository.Customer{
			ID:          customerID,
			OrgID:       orgID,
			MRRCents:    120_000,
			FirstSeenAt: &firstSeen,
			LastSeenAt:  &lastSeen,
		},
		HealthScoreHistory: []*repository.HealthScore{
			{OverallScore: 80, CalculatedAt: now.Add(-1 * 24 * time.Hour)},
			{OverallScore: 70, CalculatedAt: now.Add(-15 * 24 * time.Hour)},
			{OverallScore: 62, CalculatedAt: now.Add(-30 * 24 * time.Hour)},
		},
		Payments: []*repository.StripePayment{
			payment("succeeded", now.Add(-5*24*time.Hour)),
			payment("failed", now.Add(-10*24*time.Hour)),
			payment("failed", now.Add(-20*24*time.Hour)),
			payment("failed", now.Add(-40*24*time.Hour)),
		},
		Events: []*repository.CustomerEvent{
			event("ticket.opened", now.Add(-6*24*time.Hour), nil),
			event("ticket.opened", now.Add(-10*24*time.Hour), nil),
			event("ticket.resolved", now.Add(-4*24*time.Hour), nil),
			event("login", now.Add(-3*24*time.Hour), nil),
			event("feature_use", now.Add(-4*24*time.Hour), nil),
			event("api_call", now.Add(-9*24*time.Hour), nil),
			event("login", now.Add(-34*24*time.Hour), nil),
			event("mrr.changed", now.Add(-20*24*time.Hour), map[string]any{"old_mrr_cents": float64(100_000), "new_mrr_cents": float64(120_000)}),
		},
	})

	if features.CustomerID != customerID || features.OrgID != orgID {
		t.Fatalf("feature vector ids mismatch")
	}
	if len(features.Values) < 10 {
		t.Fatalf("expected at least 10 features, got %d", len(features.Values))
	}
	for name, value := range features.Values {
		if value < 0 || value > 1 {
			t.Fatalf("feature %s not normalized: %f", name, value)
		}
	}

	assertClose(t, features.Values[FeaturePaymentFailureFrequency90d], 0.75)
	assertClose(t, features.Values[FeaturePaymentSuccessRate90d], 0.25)
	assertClose(t, features.Values[FeatureSupportTicketTrend], 1.0)
	assertClose(t, features.Values[FeatureUsageFrequencyChange30d], 1.0)
	assertClose(t, features.Values[FeatureMRRChangeRate], 0.6)
	assertClose(t, features.Values[FeatureDaysSinceLastActivity], 0.00625)
	assertClose(t, features.Values[FeatureCurrentHealthScore], 0.8)
	assertClose(t, features.Values[FeatureUnresolvedTicketRatio], 0.5)
}

func TestExtractCustomerFeaturesHandlesMissingSignals(t *testing.T) {
	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	features := ExtractCustomerFeatures(FeatureInput{
		Now: now,
		Customer: &repository.Customer{
			ID:       uuid.New(),
			OrgID:    uuid.New(),
			MRRCents: 0,
		},
	})

	if got := features.Values[FeatureSupportTicketTrend]; got != 0.5 {
		t.Fatalf("expected stable support trend for no tickets, got %f", got)
	}
	if got := features.Values[FeatureUsageFrequencyChange30d]; got != 0.5 {
		t.Fatalf("expected neutral usage change for no usage, got %f", got)
	}
	if got := features.Values[FeatureCurrentHealthScore]; got != 0.5 {
		t.Fatalf("expected neutral current score for no score history, got %f", got)
	}
	for name, value := range features.Values {
		if value < 0 || value > 1 {
			t.Fatalf("feature %s not normalized: %f", name, value)
		}
	}
}

func TestExtractCustomerFeaturesCountsIntegrationEngagementEvents(t *testing.T) {
	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	features := ExtractCustomerFeatures(FeatureInput{
		Now: now,
		Customer: &repository.Customer{
			ID:    uuid.New(),
			OrgID: uuid.New(),
		},
		Events: []*repository.CustomerEvent{
			event("mrr.changed", now.Add(-3*24*time.Hour), nil),
			event("conversation.created", now.Add(-4*24*time.Hour), nil),
			event("deal_stage_change", now.Add(-8*24*time.Hour), nil),
			event("payment.failed", now.Add(-40*24*time.Hour), nil),
		},
	})

	assertClose(t, features.Values[FeatureEngagementEventsPerDay], 0.01)
	if got := features.Values[FeatureUsageFrequencyChange30d]; got != 0.5 {
		t.Fatalf("expected usage change to ignore integration events, got %f", got)
	}
}

func TestExtractCustomerFeaturesUsesFullMRRTrajectory(t *testing.T) {
	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	features := ExtractCustomerFeatures(FeatureInput{
		Now: now,
		Customer: &repository.Customer{
			ID:    uuid.New(),
			OrgID: uuid.New(),
		},
		Events: []*repository.CustomerEvent{
			event("mrr.changed", now.Add(-80*24*time.Hour), map[string]any{"old_mrr_cents": float64(100_000), "new_mrr_cents": float64(120_000)}),
			event("mrr.changed", now.Add(-10*24*time.Hour), map[string]any{"old_mrr_cents": float64(120_000), "new_mrr_cents": float64(150_000)}),
		},
	})

	assertClose(t, features.Values[FeatureMRRChangeRate], 0.75)
}

func TestExtractCustomerFeaturesReadsPersistedMRRNumbers(t *testing.T) {
	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	features := ExtractCustomerFeatures(FeatureInput{
		Now: now,
		Customer: &repository.Customer{
			ID:    uuid.New(),
			OrgID: uuid.New(),
		},
		Events: []*repository.CustomerEvent{
			event("mrr.changed", now.Add(-70*24*time.Hour), map[string]any{"old_mrr_cents": json.Number("100000"), "new_mrr_cents": "120000"}),
			event("mrr.changed", now.Add(-5*24*time.Hour), map[string]any{"old_mrr_cents": "120000", "new_mrr_cents": json.Number("150000")}),
		},
	})

	assertClose(t, features.Values[FeatureMRRChangeRate], 0.75)
}

func TestExtractCustomerFeaturesUsesRecentEventsForLastActivity(t *testing.T) {
	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	lastSeen := now.Add(-30 * 24 * time.Hour)
	features := ExtractCustomerFeatures(FeatureInput{
		Now: now,
		Customer: &repository.Customer{
			ID:         uuid.New(),
			OrgID:      uuid.New(),
			LastSeenAt: &lastSeen,
		},
		Events: []*repository.CustomerEvent{
			event("conversation.updated", now.Add(-2*24*time.Hour), nil),
		},
	})

	assertClose(t, features.Values[FeatureDaysSinceLastActivity], 0.05)
}

func TestNormalizeMinMaxClampsToUnitRange(t *testing.T) {
	assertClose(t, NormalizeMinMax(5, 0, 10), 0.5)
	assertClose(t, NormalizeMinMax(-5, 0, 10), 0)
	assertClose(t, NormalizeMinMax(15, 0, 10), 1)
	assertClose(t, NormalizeMinMax(5, 5, 5), 0)
}

func payment(status string, at time.Time) *repository.StripePayment {
	return &repository.StripePayment{Status: status, PaidAt: &at, CreatedAt: at}
}

func event(eventType string, at time.Time, data map[string]any) *repository.CustomerEvent {
	return &repository.CustomerEvent{EventType: eventType, OccurredAt: at, Data: data}
}

func assertClose(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.0001 {
		t.Fatalf("got %f, want %f", got, want)
	}
}
