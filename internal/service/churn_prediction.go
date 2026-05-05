package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/ml"
	"github.com/onnwee/pulse-score/internal/repository"
)

const (
	churnModelVersion        = "heuristic-v1"
	defaultPredictionInterval = 24 * time.Hour
)

type churnFeatureSource interface {
	GetByCustomerID(ctx context.Context, customerID, orgID uuid.UUID) (*repository.CustomerFeature, error)
}

type churnPredictionStore interface {
	Upsert(ctx context.Context, prediction *repository.ChurnPrediction) error
	GetByCustomerID(ctx context.Context, customerID, orgID uuid.UUID) (*repository.ChurnPrediction, error)
}

type churnCustomerSource interface {
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*repository.Customer, error)
	GetByIDAndOrg(ctx context.Context, customerID, orgID uuid.UUID) (*repository.Customer, error)
}

type churnConnectionSource interface {
	ListActiveByProvider(ctx context.Context, provider string) ([]*repository.IntegrationConnection, error)
}

// ChurnPredictionDeps contains dependencies for ChurnPredictionService.
type ChurnPredictionDeps struct {
	Customers   churnCustomerSource
	Features    churnFeatureSource
	Store       churnPredictionStore
	Connections churnConnectionSource
	Now         func() time.Time
	Interval    time.Duration
	Providers   []string
}

// ChurnPredictionService scores and stores customer churn predictions.
type ChurnPredictionService struct {
	customers   churnCustomerSource
	features    churnFeatureSource
	store       churnPredictionStore
	connections churnConnectionSource
	now         func() time.Time
	interval    time.Duration
	providers   []string
}

// NewChurnPredictionService creates a new ChurnPredictionService.
func NewChurnPredictionService(deps ChurnPredictionDeps) *ChurnPredictionService {
	now := deps.Now
	if now == nil {
		now = func() time.Time { return time.Now().UTC() }
	}
	interval := deps.Interval
	if interval <= 0 {
		interval = defaultPredictionInterval
	}
	providers := append([]string(nil), deps.Providers...)
	if len(providers) == 0 {
		providers = []string{"stripe", "hubspot", "intercom"}
	}
	return &ChurnPredictionService{
		customers:   deps.Customers,
		features:    deps.Features,
		store:       deps.Store,
		connections: deps.Connections,
		now:         now,
		interval:    interval,
		providers:   providers,
	}
}

// Start runs daily churn predictions until ctx is cancelled.
func (s *ChurnPredictionService) Start(ctx context.Context) {
	s.runBatchAndLog(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.runBatchAndLog(ctx)
		}
	}
}

func (s *ChurnPredictionService) runBatchAndLog(ctx context.Context) {
	if err := s.RunBatch(ctx); err != nil {
		slog.Error("churn prediction batch failed", "error", err)
	}
}

// RunBatch scores all customers for orgs with active data integrations.
func (s *ChurnPredictionService) RunBatch(ctx context.Context) error {
	if err := s.validateBatchDeps(); err != nil {
		return err
	}
	orgIDs, err := s.activeOrgIDs(ctx)
	if err != nil {
		return err
	}
	for _, orgID := range orgIDs {
		if err := s.ScoreOrg(ctx, orgID); err != nil {
			return fmt.Errorf("score org churn %s: %w", orgID, err)
		}
	}
	return nil
}

// ScoreOrg scores and stores churn predictions for every active customer in an org.
func (s *ChurnPredictionService) ScoreOrg(ctx context.Context, orgID uuid.UUID) error {
	if err := s.validateCustomerDeps(); err != nil {
		return err
	}
	customers, err := s.customers.ListByOrg(ctx, orgID)
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
		if err := s.ScoreCustomer(ctx, customer.ID, customer.OrgID); err != nil {
			return fmt.Errorf("score customer churn %s: %w", customer.ID, err)
		}
	}
	return nil
}

// ScoreCustomer scores and stores one customer's churn prediction.
func (s *ChurnPredictionService) ScoreCustomer(ctx context.Context, customerID, orgID uuid.UUID) error {
	if err := s.validateCustomerDeps(); err != nil {
		return err
	}
	feature, err := s.features.GetByCustomerID(ctx, customerID, orgID)
	if err != nil {
		return fmt.Errorf("get customer features: %w", err)
	}
	if feature == nil {
		return nil
	}

	prediction := s.predict(feature)
	if err := s.store.Upsert(ctx, prediction); err != nil {
		return fmt.Errorf("store churn prediction: %w", err)
	}
	return nil
}

// GetCustomerPrediction returns the current churn prediction for a customer.
func (s *ChurnPredictionService) GetCustomerPrediction(ctx context.Context, customerID, orgID uuid.UUID) (*repository.ChurnPrediction, error) {
	if s.customers == nil || s.store == nil {
		return nil, errors.New("churn prediction service dependencies are nil")
	}
	customer, err := s.customers.GetByIDAndOrg(ctx, customerID, orgID)
	if err != nil {
		return nil, fmt.Errorf("get customer: %w", err)
	}
	if customer == nil {
		return nil, &NotFoundError{Resource: "customer", Message: "customer not found"}
	}
	prediction, err := s.store.GetByCustomerID(ctx, customerID, orgID)
	if err != nil {
		return nil, fmt.Errorf("get churn prediction: %w", err)
	}
	if prediction == nil {
		return nil, &NotFoundError{Resource: "churn_prediction", Message: "churn prediction not found"}
	}
	return prediction, nil
}

func (s *ChurnPredictionService) predict(feature *repository.CustomerFeature) *repository.ChurnPrediction {
	contributions := churnFeatureContributions(feature.Features)
	probability := 0.12
	for _, contribution := range contributions {
		probability += contribution.Contribution
	}
	probability = clampProbability(probability)

	confidence := 0.35 + 0.05*float64(len(feature.Features))
	if feature.CalculatedAt.IsZero() {
		confidence -= 0.15
	} else if s.now().UTC().Sub(feature.CalculatedAt) <= 48*time.Hour {
		confidence += 0.1
	}
	confidence = clampProbability(confidence)

	return &repository.ChurnPrediction{
		OrgID:        feature.OrgID,
		CustomerID:   feature.CustomerID,
		Probability:  probability,
		Confidence:   confidence,
		RiskFactors:  topRiskFactors(contributions, 3),
		ModelVersion: churnModelVersion,
		PredictedAt:  s.now().UTC(),
	}
}

func churnFeatureContributions(features map[string]float64) []repository.ChurnRiskFactor {
	weights := map[string]float64{
		ml.FeatureCurrentHealthScore:              -0.28,
		ml.FeaturePaymentFailureFrequency90d:      0.22,
		ml.FeaturePaymentSuccessRate90d:           -0.18,
		ml.FeatureDaysSinceLastActivity:           0.14,
		ml.FeatureUsageFrequencyChange30d:         0.12,
		ml.FeatureUnresolvedTicketRatio:           0.1,
		ml.FeatureScoreTrajectorySlope30d:         0.08,
		ml.FeatureSupportTicketTrend:              0.06,
		ml.FeatureMRRChangeRate:                   0.06,
		ml.FeatureEngagementEventsPerDay:          -0.05,
		ml.FeatureContractTenure:                  -0.04,
	}

	contributions := make([]repository.ChurnRiskFactor, 0, len(weights))
	for _, name := range ml.FeatureNames() {
		weight, ok := weights[name]
		if !ok {
			continue
		}
		value := features[name]
		contribution := value * weight
		if weight < 0 {
			contribution = (1 - value) * -weight
		}
		if contribution <= 0 {
			continue
		}
		contributions = append(contributions, repository.ChurnRiskFactor{Feature: name, Contribution: contribution})
	}
	return contributions
}

func topRiskFactors(contributions []repository.ChurnRiskFactor, limit int) []repository.ChurnRiskFactor {
	sort.Slice(contributions, func(i, j int) bool {
		if contributions[i].Contribution == contributions[j].Contribution {
			return contributions[i].Feature < contributions[j].Feature
		}
		return contributions[i].Contribution > contributions[j].Contribution
	})
	if len(contributions) > limit {
		contributions = contributions[:limit]
	}
	return append([]repository.ChurnRiskFactor(nil), contributions...)
}

func (s *ChurnPredictionService) activeOrgIDs(ctx context.Context) ([]uuid.UUID, error) {
	seen := map[uuid.UUID]struct{}{}
	orgIDs := make([]uuid.UUID, 0)
	for _, provider := range s.providers {
		connections, err := s.connections.ListActiveByProvider(ctx, provider)
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

func (s *ChurnPredictionService) validateBatchDeps() error {
	if s.connections == nil {
		return errors.New("churn prediction connections dependency is nil")
	}
	return s.validateCustomerDeps()
}

func (s *ChurnPredictionService) validateCustomerDeps() error {
	switch {
	case s.customers == nil:
		return errors.New("churn prediction customers dependency is nil")
	case s.features == nil:
		return errors.New("churn prediction features dependency is nil")
	case s.store == nil:
		return errors.New("churn prediction store dependency is nil")
	default:
		return nil
	}
}

func clampProbability(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}
