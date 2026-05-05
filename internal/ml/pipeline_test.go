package ml

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

func TestFeaturePipelineRecalculatesAndStoresCustomerVector(t *testing.T) {
	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	orgID := uuid.New()
	customerID := uuid.New()
	lastSeen := now.Add(-12 * time.Hour)
	customer := &repository.Customer{
		ID:         customerID,
		OrgID:      orgID,
		LastSeenAt: &lastSeen,
	}

	store := &recordingFeatureStore{}
	pipeline := NewFeaturePipeline(FeaturePipelineDeps{
		Customers: &singleCustomerSource{customer: customer},
		HealthScores: &historySource{scores: []*repository.HealthScore{
			{OrgID: orgID, CustomerID: customerID, OverallScore: 72, CalculatedAt: now.Add(-20 * 24 * time.Hour)},
			{OrgID: orgID, CustomerID: customerID, OverallScore: 82, CalculatedAt: now.Add(-1 * 24 * time.Hour)},
		}},
		Payments: &paymentSource{payments: []*repository.StripePayment{
			payment("succeeded", now.Add(-5*24*time.Hour)),
			payment("failed", now.Add(-15*24*time.Hour)),
		}},
		Events: &eventSource{events: []*repository.CustomerEvent{
			event("login", now.Add(-2*24*time.Hour), nil),
			event("ticket.opened", now.Add(-3*24*time.Hour), nil),
		}},
		Store: store,
		Now:   func() time.Time { return now },
	})

	if err := pipeline.RecalculateCustomer(context.Background(), customerID, orgID); err != nil {
		t.Fatalf("recalculate customer features: %v", err)
	}

	if store.saved == nil {
		t.Fatalf("expected stored feature vector")
	}
	if store.saved.CustomerID != customerID || store.saved.OrgID != orgID {
		t.Fatalf("stored wrong feature vector ids")
	}
	if store.saved.CalculatedAt != now {
		t.Fatalf("stored calculated_at %s, want %s", store.saved.CalculatedAt, now)
	}
	if got := store.saved.Features[FeaturePaymentFailureFrequency90d]; got != 0.5 {
		t.Fatalf("stored payment failure frequency %f, want 0.5", got)
	}
}

func TestFeaturePipelineRecalculatesOrgCustomers(t *testing.T) {
	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	orgID := uuid.New()
	customers := []*repository.Customer{
		{ID: uuid.New(), OrgID: orgID},
		{ID: uuid.New(), OrgID: orgID},
	}
	store := &recordingFeatureStore{}
	events := &eventSource{}
	pipeline := NewFeaturePipeline(FeaturePipelineDeps{
		Customers:    &singleCustomerSource{customers: customers},
		HealthScores: &historySource{},
		Payments:     &paymentSource{},
		Events:       events,
		Store:        store,
		Now:          func() time.Time { return now },
	})

	if err := pipeline.RecalculateOrg(context.Background(), orgID); err != nil {
		t.Fatalf("recalculate org features: %v", err)
	}

	if len(store.savedAll) != len(customers) {
		t.Fatalf("stored %d feature vectors, want %d", len(store.savedAll), len(customers))
	}
	if events.lastSince != now.Add(-90*24*time.Hour) {
		t.Fatalf("event lookback %s, want 90 days", events.lastSince)
	}
}

func TestFeaturePipelineRunBatchDedupesActiveIntegrationOrgs(t *testing.T) {
	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	orgID := uuid.New()
	customer := &repository.Customer{ID: uuid.New(), OrgID: orgID}
	store := &recordingFeatureStore{}
	pipeline := NewFeaturePipeline(FeaturePipelineDeps{
		Customers:    &singleCustomerSource{customer: customer},
		HealthScores: &historySource{},
		Payments:     &paymentSource{},
		Events:       &eventSource{},
		Store:        store,
		Connections: &connectionSource{connections: map[string][]*repository.IntegrationConnection{
			"stripe":  {{OrgID: orgID}},
			"hubspot": {{OrgID: orgID}},
		}},
		Now: func() time.Time { return now },
	})

	if err := pipeline.RunBatch(context.Background()); err != nil {
		t.Fatalf("run batch: %v", err)
	}

	if len(store.savedAll) != 1 {
		t.Fatalf("stored %d feature vectors, want 1 deduped org run", len(store.savedAll))
	}
}

func TestFeaturePipelineStartRunsInitialBatchImmediately(t *testing.T) {
	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	orgID := uuid.New()
	customer := &repository.Customer{ID: uuid.New(), OrgID: orgID}
	store := &recordingFeatureStore{savedCh: make(chan struct{}, 1)}
	pipeline := NewFeaturePipeline(FeaturePipelineDeps{
		Customers:    &singleCustomerSource{customer: customer},
		HealthScores: &historySource{},
		Payments:     &paymentSource{},
		Events:       &eventSource{},
		Store:        store,
		Connections: &connectionSource{connections: map[string][]*repository.IntegrationConnection{
			"stripe": {{OrgID: orgID}},
		}},
		Now:      func() time.Time { return now },
		Interval: time.Hour,
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go pipeline.Start(ctx)

	select {
	case <-store.savedCh:
		cancel()
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("expected initial feature batch before first ticker interval")
	}
}

type singleCustomerSource struct {
	customer  *repository.Customer
	customers []*repository.Customer
}

func (s *singleCustomerSource) GetByIDAndOrg(_ context.Context, customerID, orgID uuid.UUID) (*repository.Customer, error) {
	if s.customer != nil && s.customer.ID == customerID && s.customer.OrgID == orgID {
		return s.customer, nil
	}
	for _, customer := range s.customers {
		if customer.ID == customerID && customer.OrgID == orgID {
			return customer, nil
		}
	}
	return nil, nil
}

func (s *singleCustomerSource) ListByOrg(_ context.Context, orgID uuid.UUID) ([]*repository.Customer, error) {
	if s.customer != nil && s.customer.OrgID == orgID {
		return []*repository.Customer{s.customer}, nil
	}
	var customers []*repository.Customer
	for _, customer := range s.customers {
		if customer.OrgID == orgID {
			customers = append(customers, customer)
		}
	}
	if len(customers) > 0 {
		return customers, nil
	}
	return nil, nil
}

type historySource struct {
	scores []*repository.HealthScore
}

func (s *historySource) GetHistory(_ context.Context, _ uuid.UUID, _ int) ([]*repository.HealthScore, error) {
	return s.scores, nil
}

type paymentSource struct {
	payments []*repository.StripePayment
}

func (s *paymentSource) ListByCustomer(_ context.Context, _ uuid.UUID) ([]*repository.StripePayment, error) {
	return s.payments, nil
}

type eventSource struct {
	events    []*repository.CustomerEvent
	lastSince time.Time
}

func (s *eventSource) ListByCustomerSince(_ context.Context, _ uuid.UUID, since time.Time) ([]*repository.CustomerEvent, error) {
	s.lastSince = since
	return s.events, nil
}

type recordingFeatureStore struct {
	saved    *repository.CustomerFeature
	savedAll []*repository.CustomerFeature
	savedCh  chan struct{}
}

func (s *recordingFeatureStore) Upsert(_ context.Context, feature *repository.CustomerFeature) error {
	s.saved = feature
	s.savedAll = append(s.savedAll, feature)
	if s.savedCh != nil {
		select {
		case s.savedCh <- struct{}{}:
		default:
		}
	}
	return nil
}

type connectionSource struct {
	connections map[string][]*repository.IntegrationConnection
}

func (s *connectionSource) ListActiveByProvider(_ context.Context, provider string) ([]*repository.IntegrationConnection, error) {
	return s.connections[provider], nil
}
