package ml

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

const (
	featureHistoryLimit   = 50
	featureEventLookback  = 90 * 24 * time.Hour
	defaultFeatureInterval = 24 * time.Hour
)

var defaultFeatureProviders = []string{"stripe", "hubspot", "intercom"}

// FeaturePipeline recalculates and stores churn-model feature vectors.
type FeaturePipeline struct {
	customers    CustomerSource
	healthScores HealthScoreSource
	payments     PaymentSource
	events       EventSource
	store        FeatureStore
	connections  ConnectionSource
	now          func() time.Time
	interval     time.Duration
	providers    []string
}

// CustomerSource loads customers for feature extraction.
type CustomerSource interface {
	GetByIDAndOrg(ctx context.Context, customerID, orgID uuid.UUID) (*repository.Customer, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*repository.Customer, error)
}

// HealthScoreSource loads historical scores for trajectory features.
type HealthScoreSource interface {
	GetHistory(ctx context.Context, customerID uuid.UUID, limit int) ([]*repository.HealthScore, error)
}

// PaymentSource loads Stripe payments for billing-risk features.
type PaymentSource interface {
	ListByCustomer(ctx context.Context, customerID uuid.UUID) ([]*repository.StripePayment, error)
}

// EventSource loads customer events for usage, support, and MRR features.
type EventSource interface {
	ListByCustomerSince(ctx context.Context, customerID uuid.UUID, since time.Time) ([]*repository.CustomerEvent, error)
}

// FeatureStore persists the latest feature vector for a customer.
type FeatureStore interface {
	Upsert(ctx context.Context, feature *repository.CustomerFeature) error
}

// ConnectionSource identifies orgs with active integrations for scheduled runs.
type ConnectionSource interface {
	ListActiveByProvider(ctx context.Context, provider string) ([]*repository.IntegrationConnection, error)
}

// FeaturePipelineDeps contains dependencies for FeaturePipeline.
type FeaturePipelineDeps struct {
	Customers    CustomerSource
	HealthScores HealthScoreSource
	Payments     PaymentSource
	Events       EventSource
	Store        FeatureStore
	Connections  ConnectionSource
	Now          func() time.Time
	Interval     time.Duration
	Providers    []string
}

// NewFeaturePipeline creates a feature engineering pipeline.
func NewFeaturePipeline(deps FeaturePipelineDeps) *FeaturePipeline {
	now := deps.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}

	interval := deps.Interval
	if interval <= 0 {
		interval = defaultFeatureInterval
	}

	providers := append([]string(nil), deps.Providers...)
	if len(providers) == 0 {
		providers = append([]string(nil), defaultFeatureProviders...)
	}

	return &FeaturePipeline{
		customers:    deps.Customers,
		healthScores: deps.HealthScores,
		payments:     deps.Payments,
		events:       deps.Events,
		store:        deps.Store,
		connections:  deps.Connections,
		now:          now,
		interval:     interval,
		providers:    providers,
	}
}

// Start runs daily feature-vector recalculation until ctx is cancelled.
func (p *FeaturePipeline) Start(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := p.RunBatch(ctx); err != nil {
				slog.Error("feature pipeline batch failed", "error", err)
			}
		}
	}
}

// RunBatch recalculates features for orgs with active data integrations.
func (p *FeaturePipeline) RunBatch(ctx context.Context) error {
	if err := p.validateBatchDeps(); err != nil {
		return err
	}

	orgIDs, err := p.activeOrgIDs(ctx)
	if err != nil {
		return err
	}
	for _, orgID := range orgIDs {
		if err := p.RecalculateOrg(ctx, orgID); err != nil {
			return fmt.Errorf("recalculate org features %s: %w", orgID, err)
		}
	}
	return nil
}

// RecalculateOrg recalculates feature vectors for every active customer in an org.
func (p *FeaturePipeline) RecalculateOrg(ctx context.Context, orgID uuid.UUID) error {
	if err := p.validateCustomerDeps(); err != nil {
		return err
	}

	customers, err := p.customers.ListByOrg(ctx, orgID)
	if err != nil {
		return fmt.Errorf("list org customers: %w", err)
	}
	for _, customer := range customers {
		if err := ctx.Err(); err != nil {
			return err
		}
		if customer == nil {
			continue
		}
		if err := p.recalculateLoadedCustomer(ctx, customer); err != nil {
			return fmt.Errorf("recalculate customer features %s: %w", customer.ID, err)
		}
	}
	return nil
}

// RecalculateCustomer recalculates one customer's feature vector.
func (p *FeaturePipeline) RecalculateCustomer(ctx context.Context, customerID, orgID uuid.UUID) error {
	if err := p.validateCustomerDeps(); err != nil {
		return err
	}

	customer, err := p.customers.GetByIDAndOrg(ctx, customerID, orgID)
	if err != nil {
		return fmt.Errorf("get customer: %w", err)
	}
	if customer == nil {
		return fmt.Errorf("customer %s not found in org %s", customerID, orgID)
	}
	return p.recalculateLoadedCustomer(ctx, customer)
}

func (p *FeaturePipeline) recalculateLoadedCustomer(ctx context.Context, customer *repository.Customer) error {
	now := p.now().UTC()
	history, err := p.healthScores.GetHistory(ctx, customer.ID, featureHistoryLimit)
	if err != nil {
		return fmt.Errorf("load health score history: %w", err)
	}
	payments, err := p.payments.ListByCustomer(ctx, customer.ID)
	if err != nil {
		return fmt.Errorf("load payments: %w", err)
	}
	events, err := p.events.ListByCustomerSince(ctx, customer.ID, now.Add(-featureEventLookback))
	if err != nil {
		return fmt.Errorf("load customer events: %w", err)
	}

	vector := ExtractCustomerFeatures(FeatureInput{
		Now:                now,
		Customer:           customer,
		HealthScoreHistory: history,
		Payments:           payments,
		Events:             events,
	})

	feature := &repository.CustomerFeature{
		OrgID:        vector.OrgID,
		CustomerID:   vector.CustomerID,
		Features:     vector.Values,
		CalculatedAt: vector.CalculatedAt,
	}
	if err := p.store.Upsert(ctx, feature); err != nil {
		return fmt.Errorf("store customer features: %w", err)
	}
	return nil
}

func (p *FeaturePipeline) activeOrgIDs(ctx context.Context) ([]uuid.UUID, error) {
	seen := map[uuid.UUID]struct{}{}
	orgIDs := make([]uuid.UUID, 0)
	for _, provider := range p.providers {
		connections, err := p.connections.ListActiveByProvider(ctx, provider)
		if err != nil {
			return nil, fmt.Errorf("list active %s connections: %w", provider, err)
		}
		for _, conn := range connections {
			if conn == nil {
				continue
			}
			if _, ok := seen[conn.OrgID]; ok {
				continue
			}
			seen[conn.OrgID] = struct{}{}
			orgIDs = append(orgIDs, conn.OrgID)
		}
	}
	return orgIDs, nil
}

func (p *FeaturePipeline) validateBatchDeps() error {
	if p.connections == nil {
		return errors.New("feature pipeline connections dependency is nil")
	}
	return p.validateCustomerDeps()
}

func (p *FeaturePipeline) validateCustomerDeps() error {
	switch {
	case p.customers == nil:
		return errors.New("feature pipeline customers dependency is nil")
	case p.healthScores == nil:
		return errors.New("feature pipeline health scores dependency is nil")
	case p.payments == nil:
		return errors.New("feature pipeline payments dependency is nil")
	case p.events == nil:
		return errors.New("feature pipeline events dependency is nil")
	case p.store == nil:
		return errors.New("feature pipeline store dependency is nil")
	default:
		return nil
	}
}
