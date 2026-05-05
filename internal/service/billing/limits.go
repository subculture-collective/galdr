package billing

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	planmodel "github.com/onnwee/pulse-score/internal/billing"
	"github.com/onnwee/pulse-score/internal/repository"
)

type integrationLookup interface {
	GetByOrgAndProvider(ctx context.Context, orgID uuid.UUID, provider string) (*repository.IntegrationConnection, error)
}

// LimitDecision is used for feature-gating responses and middleware decisions.
type LimitDecision struct {
	Allowed                bool   `json:"allowed"`
	CurrentPlan            string `json:"current_plan"`
	LimitType              string `json:"limit_type"`
	CurrentUsage           int    `json:"current_usage"`
	Limit                  int    `json:"limit"`
	RecommendedUpgradeTier string `json:"recommended_upgrade_tier"`
}

// FeatureDecision is used for plan-based feature access checks.
type FeatureDecision struct {
	Allowed                bool   `json:"allowed"`
	CurrentPlan            string `json:"current_plan"`
	Feature                string `json:"feature"`
	RecommendedUpgradeTier string `json:"recommended_upgrade_tier"`
}

// LimitsService handles server-side billing limits and feature access checks.
type LimitsService struct {
	subscriptions      *SubscriptionService
	customers          customerCounter
	integrationCounter integrationCounter
	integrationLookup  integrationLookup
	catalog            *planmodel.Catalog
	featureFlags       *FeatureFlagService
}

func NewLimitsService(
	subscriptions *SubscriptionService,
	customers customerCounter,
	integrationCounter integrationCounter,
	integrationLookup integrationLookup,
	overrides featureOverrideReader,
	catalog *planmodel.Catalog,
) *LimitsService {
	return &LimitsService{
		subscriptions:      subscriptions,
		customers:          customers,
		integrationCounter: integrationCounter,
		integrationLookup:  integrationLookup,
		catalog:            catalog,
		featureFlags:       NewFeatureFlagService(subscriptions, overrides, catalog),
	}
}

func (s *LimitsService) CheckCustomerLimit(ctx context.Context, orgID uuid.UUID) (*LimitDecision, error) {
	tier, err := s.subscriptions.GetCurrentPlan(ctx, orgID)
	if err != nil {
		return nil, err
	}

	limit, err := s.featureFlags.GetLimit(ctx, orgID, LimitCustomer)
	if err != nil {
		return nil, err
	}

	used, err := s.customers.CountByOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("count customers: %w", err)
	}

	return s.buildLimitDecision(tier, LimitCustomer, used, limit), nil
}

func (s *LimitsService) CheckIntegrationLimit(ctx context.Context, orgID uuid.UUID, provider string) (*LimitDecision, error) {
	if provider != "" {
		conn, err := s.integrationLookup.GetByOrgAndProvider(ctx, orgID, provider)
		if err != nil {
			return nil, fmt.Errorf("get integration by provider: %w", err)
		}
		if conn != nil && conn.Status == "active" {
			tier, err := s.subscriptions.GetCurrentPlan(ctx, orgID)
			if err != nil {
				return nil, err
			}
			return &LimitDecision{Allowed: true, CurrentPlan: tier}, nil
		}
	}

	tier, err := s.subscriptions.GetCurrentPlan(ctx, orgID)
	if err != nil {
		return nil, err
	}

	limit, err := s.featureFlags.GetLimit(ctx, orgID, LimitIntegration)
	if err != nil {
		return nil, err
	}

	used, err := s.integrationCounter.CountActiveByOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("count active integrations: %w", err)
	}

	return s.buildLimitDecision(tier, LimitIntegration, used, limit), nil
}

func (s *LimitsService) CanAccess(ctx context.Context, orgID uuid.UUID, featureName string) (*FeatureDecision, error) {
	tier, err := s.subscriptions.GetCurrentPlan(ctx, orgID)
	if err != nil {
		return nil, err
	}

	if _, ok := s.catalog.GetPlanByTier(tier); !ok {
		return nil, fmt.Errorf("no plan configured for tier %s", tier)
	}

	allowed, err := s.featureFlags.CanAccess(ctx, orgID, featureName)
	if err != nil {
		return nil, err
	}

	decision := &FeatureDecision{
		Allowed:     allowed,
		CurrentPlan: tier,
		Feature:     featureName,
	}
	if !allowed {
		decision.RecommendedUpgradeTier = string(s.catalog.RecommendedUpgrade(tier))
	}

	return decision, nil
}

func (s *LimitsService) buildLimitDecision(tier, limitType string, used, limit int) *LimitDecision {
	decision := &LimitDecision{
		Allowed:      true,
		CurrentPlan:  tier,
		LimitType:    limitType,
		CurrentUsage: used,
		Limit:        limit,
	}

	if limit != planmodel.Unlimited && used >= limit {
		decision.Allowed = false
		decision.RecommendedUpgradeTier = string(s.catalog.RecommendedUpgrade(tier))
	}

	return decision
}
