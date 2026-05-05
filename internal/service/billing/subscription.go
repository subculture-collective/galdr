package billing

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	planmodel "github.com/onnwee/pulse-score/internal/billing"
	"github.com/onnwee/pulse-score/internal/repository"
)

type orgSubscriptionReader interface {
	GetByOrg(ctx context.Context, orgID uuid.UUID) (*repository.OrgSubscription, error)
}

type organizationReader interface {
	GetByID(ctx context.Context, id uuid.UUID) (*repository.Organization, error)
	CountMembers(ctx context.Context, orgID uuid.UUID) (int, error)
}

type customerCounter interface {
	CountByOrg(ctx context.Context, orgID uuid.UUID) (int, error)
}

type integrationCounter interface {
	CountActiveByOrg(ctx context.Context, orgID uuid.UUID) (int, error)
}

// SubscriptionService exposes org-level subscription state and limit information.
type SubscriptionService struct {
	subscriptions orgSubscriptionReader
	orgs          organizationReader
	customers     customerCounter
	integrations  integrationCounter
	catalog       *planmodel.Catalog
	featureFlags  *FeatureFlagService
}

// UsageSnapshot contains current usage against plan limits.
type UsageSnapshot struct {
	Customers struct {
		Used  int `json:"used"`
		Limit int `json:"limit"`
	} `json:"customers"`
	Integrations struct {
		Used  int `json:"used"`
		Limit int `json:"limit"`
	} `json:"integrations"`
	TeamMembers struct {
		Used  int `json:"used"`
		Limit int `json:"limit"`
	} `json:"team_members"`
}

func (s *SubscriptionService) SetFeatureOverrides(overrides featureOverrideReader) {
	s.featureFlags = NewFeatureFlagService(s, overrides, s.catalog)
}

// SubscriptionSummary is returned by GET /api/v1/billing/subscription.
type SubscriptionSummary struct {
	Tier              string         `json:"tier"`
	Status            string         `json:"status"`
	BillingCycle      string         `json:"billing_cycle"`
	RenewalDate       *time.Time     `json:"renewal_date"`
	CancelAtPeriodEnd bool           `json:"cancel_at_period_end"`
	Usage             UsageSnapshot  `json:"usage"`
	Features          map[string]bool `json:"features"`
}

func NewSubscriptionService(
	subscriptions orgSubscriptionReader,
	orgs organizationReader,
	customers customerCounter,
	integrations integrationCounter,
	catalog *planmodel.Catalog,
) *SubscriptionService {
	return &SubscriptionService{
		subscriptions: subscriptions,
		orgs:          orgs,
		customers:     customers,
		integrations:  integrations,
		catalog:       catalog,
	}
}

// GetCurrentPlan resolves the current tier, falling back to org.plan then free.
func (s *SubscriptionService) GetCurrentPlan(ctx context.Context, orgID uuid.UUID) (string, error) {
	sub, err := s.subscriptions.GetByOrg(ctx, orgID)
	if err != nil {
		return "", fmt.Errorf("get org subscription: %w", err)
	}
	if sub != nil && strings.TrimSpace(sub.PlanTier) != "" {
		return string(planmodel.NormalizeTier(sub.PlanTier)), nil
	}

	org, err := s.orgs.GetByID(ctx, orgID)
	if err != nil {
		return "", fmt.Errorf("get organization: %w", err)
	}
	if org != nil && strings.TrimSpace(org.Plan) != "" {
		return string(planmodel.NormalizeTier(org.Plan)), nil
	}

	return string(planmodel.TierFree), nil
}

// IsActive reports whether the org subscription status is currently active.
func (s *SubscriptionService) IsActive(ctx context.Context, orgID uuid.UUID) (bool, error) {
	sub, err := s.subscriptions.GetByOrg(ctx, orgID)
	if err != nil {
		return false, fmt.Errorf("get org subscription: %w", err)
	}
	if sub == nil {
		return true, nil // Free tier fallback should remain usable.
	}

	status := strings.ToLower(strings.TrimSpace(sub.Status))
	switch status {
	case "active", "trialing", "past_due":
		return true, nil
	default:
		return false, nil
	}
}

// GetUsageLimits resolves the current usage limits from the plan catalog.
func (s *SubscriptionService) GetUsageLimits(ctx context.Context, orgID uuid.UUID) (planmodel.UsageLimits, error) {
	tier, err := s.GetCurrentPlan(ctx, orgID)
	if err != nil {
		return planmodel.UsageLimits{}, err
	}

	if s.featureFlags != nil {
		return s.featureFlags.EffectiveLimits(ctx, orgID)
	}

	limits, ok := s.catalog.GetLimits(tier)
	if !ok {
		return planmodel.UsageLimits{}, fmt.Errorf("no limits configured for tier %s", tier)
	}
	return limits, nil
}

// GetSubscriptionSummary returns the current subscription state and usage counters.
func (s *SubscriptionService) GetSubscriptionSummary(ctx context.Context, orgID uuid.UUID) (*SubscriptionSummary, error) {
	tier, err := s.GetCurrentPlan(ctx, orgID)
	if err != nil {
		return nil, err
	}

	limits, err := s.GetUsageLimits(ctx, orgID)
	if err != nil {
		return nil, err
	}

	customerCount, err := s.customers.CountByOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("count customers: %w", err)
	}
	integrationCount, err := s.integrations.CountActiveByOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("count integrations: %w", err)
	}
	memberCount, err := s.orgs.CountMembers(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("count members: %w", err)
	}

	summary := &SubscriptionSummary{
		Tier:         tier,
		Status:       "free",
		BillingCycle: string(planmodel.BillingCycleMonthly),
		Features:     map[string]bool{},
	}

	summary.Usage.Customers.Used = customerCount
	summary.Usage.Customers.Limit = limits.CustomerLimit
	summary.Usage.Integrations.Used = integrationCount
	summary.Usage.Integrations.Limit = limits.IntegrationLimit
	summary.Usage.TeamMembers.Used = memberCount
	summary.Usage.TeamMembers.Limit = limits.TeamMemberLimit

	if s.featureFlags != nil {
		features, err := s.featureFlags.EffectiveFeatures(ctx, orgID)
		if err != nil {
			return nil, err
		}
		summary.Features = features
	} else if plan, ok := s.catalog.GetPlanByTier(tier); ok {
		summary.Features = subscriptionFeatureMap(plan.Features)
	}

	sub, err := s.subscriptions.GetByOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("get org subscription: %w", err)
	}
	if sub != nil {
		summary.Status = sub.Status
		if strings.TrimSpace(sub.BillingCycle) != "" {
			summary.BillingCycle = sub.BillingCycle
		}
		summary.RenewalDate = sub.CurrentPeriodEnd
		summary.CancelAtPeriodEnd = sub.CancelAtPeriodEnd
	}

	return summary, nil
}

func subscriptionFeatureMap(features planmodel.FeatureFlags) map[string]bool {
	return map[string]bool{
		planmodel.FeatureBasicDashboard: features.BasicDashboard,
		planmodel.FeatureFullDashboard:  features.FullDashboard,
		planmodel.FeatureEmailAlerts:    features.EmailAlerts,
		planmodel.FeaturePlaybooks:      features.Playbooks,
		planmodel.FeatureAIInsights:     features.AIInsights,
		planmodel.FeatureBenchmarks:     features.Benchmarks,
	}
}
