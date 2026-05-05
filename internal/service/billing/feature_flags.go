package billing

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	planmodel "github.com/onnwee/pulse-score/internal/billing"
	"github.com/onnwee/pulse-score/internal/repository"
)

const (
	LimitCustomer    = "customer_limit"
	LimitIntegration = "integration_limit"
	LimitTeamMember  = "team_member_limit"
)

type planResolver interface {
	GetCurrentPlan(ctx context.Context, orgID uuid.UUID) (string, error)
}

type featureOverrideReader interface {
	GetByOrgAndFeature(ctx context.Context, orgID uuid.UUID, featureName string) (*repository.FeatureOverride, error)
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]repository.FeatureOverride, error)
}

// FeatureFlagService resolves effective feature and limit access for an org.
type FeatureFlagService struct {
	plans     planResolver
	overrides featureOverrideReader
	catalog   *planmodel.Catalog
}

func NewFeatureFlagService(plans planResolver, overrides featureOverrideReader, catalog *planmodel.Catalog) *FeatureFlagService {
	return &FeatureFlagService{plans: plans, overrides: overrides, catalog: catalog}
}

func (s *FeatureFlagService) CanAccess(ctx context.Context, orgID uuid.UUID, featureName string) (bool, error) {
	featureName = normalizeFlagName(featureName)
	tier, err := s.plans.GetCurrentPlan(ctx, orgID)
	if err != nil {
		return false, err
	}

	allowed, ok := s.catalog.HasFeature(tier, featureName)
	if !ok {
		return false, fmt.Errorf("unknown feature %s", featureName)
	}

	override, err := s.getOverride(ctx, orgID, featureName)
	if err != nil {
		return false, err
	}
	if override != nil && override.Enabled != nil {
		return *override.Enabled, nil
	}

	return allowed, nil
}

func (s *FeatureFlagService) GetLimit(ctx context.Context, orgID uuid.UUID, limitName string) (int, error) {
	limitName = normalizeFlagName(limitName)
	tier, err := s.plans.GetCurrentPlan(ctx, orgID)
	if err != nil {
		return 0, err
	}

	limits, ok := s.catalog.GetLimits(tier)
	if !ok {
		return 0, fmt.Errorf("no limits configured for tier %s", tier)
	}

	limit, ok := defaultLimit(limits, limitName)
	if !ok {
		return 0, fmt.Errorf("unknown limit %s", limitName)
	}

	override, err := s.getOverride(ctx, orgID, limitName)
	if err != nil {
		return 0, err
	}
	if override != nil && override.LimitOverride != nil {
		return *override.LimitOverride, nil
	}

	return limit, nil
}

func (s *FeatureFlagService) EffectiveFeatures(ctx context.Context, orgID uuid.UUID) (map[string]bool, error) {
	tier, err := s.plans.GetCurrentPlan(ctx, orgID)
	if err != nil {
		return nil, err
	}

	plan, ok := s.catalog.GetPlanByTier(tier)
	if !ok {
		return nil, fmt.Errorf("no plan configured for tier %s", tier)
	}

	features := subscriptionFeatureMap(plan.Features)
	overrides, err := s.listOverrides(ctx, orgID)
	if err != nil {
		return nil, err
	}
	for _, override := range overrides {
		featureName := normalizeFlagName(override.FeatureName)
		if _, ok := features[featureName]; ok && override.Enabled != nil {
			features[featureName] = *override.Enabled
		}
	}

	return features, nil
}

func (s *FeatureFlagService) EffectiveLimits(ctx context.Context, orgID uuid.UUID) (planmodel.UsageLimits, error) {
	tier, err := s.plans.GetCurrentPlan(ctx, orgID)
	if err != nil {
		return planmodel.UsageLimits{}, err
	}

	limits, ok := s.catalog.GetLimits(tier)
	if !ok {
		return planmodel.UsageLimits{}, fmt.Errorf("no limits configured for tier %s", tier)
	}

	overrides, err := s.listOverrides(ctx, orgID)
	if err != nil {
		return planmodel.UsageLimits{}, err
	}
	for _, override := range overrides {
		if override.LimitOverride == nil {
			continue
		}
		switch normalizeFlagName(override.FeatureName) {
		case LimitCustomer:
			limits.CustomerLimit = *override.LimitOverride
		case LimitIntegration:
			limits.IntegrationLimit = *override.LimitOverride
		case LimitTeamMember:
			limits.TeamMemberLimit = *override.LimitOverride
		}
	}

	return limits, nil
}

func (s *FeatureFlagService) getOverride(ctx context.Context, orgID uuid.UUID, name string) (*repository.FeatureOverride, error) {
	if s.overrides == nil {
		return nil, nil
	}
	override, err := s.overrides.GetByOrgAndFeature(ctx, orgID, normalizeFlagName(name))
	if err != nil {
		return nil, fmt.Errorf("get feature override: %w", err)
	}
	return override, nil
}

func (s *FeatureFlagService) listOverrides(ctx context.Context, orgID uuid.UUID) ([]repository.FeatureOverride, error) {
	if s.overrides == nil {
		return nil, nil
	}
	overrides, err := s.overrides.ListByOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("list feature overrides: %w", err)
	}
	return overrides, nil
}

func defaultLimit(limits planmodel.UsageLimits, limitName string) (int, bool) {
	switch limitName {
	case LimitCustomer:
		return limits.CustomerLimit, true
	case LimitIntegration:
		return limits.IntegrationLimit, true
	case LimitTeamMember:
		return limits.TeamMemberLimit, true
	default:
		return 0, false
	}
}

func normalizeFlagName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
