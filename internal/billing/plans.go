package billing

import (
	"fmt"
	"strings"
)

const Unlimited = -1

type Tier string

const (
	TierFree   Tier = "free"
	TierGrowth Tier = "growth"
	TierScale  Tier = "scale"
)

type BillingCycle string

const (
	BillingCycleMonthly BillingCycle = "monthly"
	BillingCycleAnnual  BillingCycle = "annual"
)

type UsageLimits struct {
	CustomerLimit    int `json:"customer_limit"`
	IntegrationLimit int `json:"integration_limit"`
	TeamMemberLimit  int `json:"team_member_limit"`
}

type FeatureFlags struct {
	BasicDashboard bool `json:"basic_dashboard"`
	FullDashboard  bool `json:"full_dashboard"`
	EmailAlerts    bool `json:"email_alerts"`
	Playbooks      bool `json:"playbooks"`
	AIInsights     bool `json:"ai_insights"`
	Benchmarks     bool `json:"benchmarks"`
}

const (
	FeatureBasicDashboard = "basic_dashboard"
	FeatureFullDashboard  = "full_dashboard"
	FeatureEmailAlerts    = "email_alerts"
	FeaturePlaybooks      = "playbooks"
	FeatureAIInsights     = "ai_insights"
	FeatureBenchmarks     = "benchmarks"
)

type Plan struct {
	Tier                 Tier         `json:"tier"`
	Name                 string       `json:"name"`
	Description          string       `json:"description"`
	MonthlyPriceCents    int          `json:"monthly_price_cents"`
	AnnualPriceCents     int          `json:"annual_price_cents"`
	StripeMonthlyPriceID string       `json:"stripe_monthly_price_id"`
	StripeAnnualPriceID  string       `json:"stripe_annual_price_id"`
	Limits               UsageLimits  `json:"limits"`
	Features             FeatureFlags `json:"features"`
}

type PriceConfig struct {
	GrowthMonthly string
	GrowthAnnual  string
	ScaleMonthly  string
	ScaleAnnual   string
}

type Catalog struct {
	plans      map[Tier]Plan
	priceIndex map[string]priceRef
}

type priceRef struct {
	Tier  Tier
	Cycle BillingCycle
}

func NewCatalog(cfg PriceConfig) *Catalog {
	plans := map[Tier]Plan{
		TierFree: {
			Tier:              TierFree,
			Name:              "Free",
			Description:       "Best for evaluating PulseScore with a small portfolio.",
			MonthlyPriceCents: 0,
			AnnualPriceCents:  0,
			Limits: UsageLimits{
				CustomerLimit:    10,
				IntegrationLimit: 1,
				TeamMemberLimit:  1,
			},
			Features: FeatureFlags{
				BasicDashboard: true,
				EmailAlerts:    true,
			},
		},
		TierGrowth: {
			Tier:                 TierGrowth,
			Name:                 "Growth",
			Description:          "For fast-moving teams managing churn at scale.",
			MonthlyPriceCents:    4900,
			AnnualPriceCents:     49000,
			StripeMonthlyPriceID: strings.TrimSpace(cfg.GrowthMonthly),
			StripeAnnualPriceID:  strings.TrimSpace(cfg.GrowthAnnual),
			Limits: UsageLimits{
				CustomerLimit:    500,
				IntegrationLimit: 3,
				TeamMemberLimit:  5,
			},
			Features: FeatureFlags{
				BasicDashboard: true,
				FullDashboard:  true,
				EmailAlerts:    true,
				Playbooks:      true,
			},
		},
		TierScale: {
			Tier:                 TierScale,
			Name:                 "Scale",
			Description:          "For mature revenue teams with complex customer motion.",
			MonthlyPriceCents:    14900,
			AnnualPriceCents:     149000,
			StripeMonthlyPriceID: strings.TrimSpace(cfg.ScaleMonthly),
			StripeAnnualPriceID:  strings.TrimSpace(cfg.ScaleAnnual),
			Limits: UsageLimits{
				CustomerLimit:    Unlimited,
				IntegrationLimit: Unlimited,
				TeamMemberLimit:  Unlimited,
			},
			Features: FeatureFlags{
				BasicDashboard: true,
				FullDashboard:  true,
				EmailAlerts:    true,
				Playbooks:      true,
				AIInsights:     true,
				Benchmarks:     true,
			},
		},
	}

	priceIndex := map[string]priceRef{}
	for _, p := range plans {
		if p.StripeMonthlyPriceID != "" {
			priceIndex[p.StripeMonthlyPriceID] = priceRef{Tier: p.Tier, Cycle: BillingCycleMonthly}
		}
		if p.StripeAnnualPriceID != "" {
			priceIndex[p.StripeAnnualPriceID] = priceRef{Tier: p.Tier, Cycle: BillingCycleAnnual}
		}
	}

	return &Catalog{plans: plans, priceIndex: priceIndex}
}

func (c *Catalog) GetPlanByTier(tier string) (Plan, bool) {
	if c == nil {
		return Plan{}, false
	}
	plan, ok := c.plans[NormalizeTier(tier)]
	return plan, ok
}

func (c *Catalog) GetLimits(tier string) (UsageLimits, bool) {
	plan, ok := c.GetPlanByTier(tier)
	if !ok {
		return UsageLimits{}, false
	}
	return plan.Limits, true
}

func (c *Catalog) HasFeature(tier string, featureName string) (bool, bool) {
	plan, ok := c.GetPlanByTier(tier)
	if !ok {
		return false, false
	}

	switch strings.ToLower(strings.TrimSpace(featureName)) {
	case FeatureBasicDashboard:
		return plan.Features.BasicDashboard, true
	case FeatureFullDashboard:
		return plan.Features.FullDashboard, true
	case FeatureEmailAlerts:
		return plan.Features.EmailAlerts, true
	case FeaturePlaybooks:
		return plan.Features.Playbooks, true
	case FeatureAIInsights:
		return plan.Features.AIInsights, true
	case FeatureBenchmarks:
		return plan.Features.Benchmarks, true
	default:
		return false, false
	}
}

func (c *Catalog) GetPriceID(tier string, annual bool) (string, error) {
	plan, ok := c.GetPlanByTier(tier)
	if !ok {
		return "", fmt.Errorf("unknown tier: %s", tier)
	}
	if plan.Tier == TierFree {
		return "", fmt.Errorf("free tier does not have a Stripe price")
	}

	if annual {
		if plan.StripeAnnualPriceID == "" {
			return "", fmt.Errorf("annual price id is not configured for tier %s", plan.Tier)
		}
		return plan.StripeAnnualPriceID, nil
	}

	if plan.StripeMonthlyPriceID == "" {
		return "", fmt.Errorf("monthly price id is not configured for tier %s", plan.Tier)
	}
	return plan.StripeMonthlyPriceID, nil
}

func (c *Catalog) ResolveTierAndCycleByPriceID(priceID string) (Tier, BillingCycle, bool) {
	if c == nil {
		return "", "", false
	}
	ref, ok := c.priceIndex[strings.TrimSpace(priceID)]
	if !ok {
		return "", "", false
	}
	return ref.Tier, ref.Cycle, true
}

func (c *Catalog) RecommendedUpgrade(tier string) Tier {
	switch NormalizeTier(tier) {
	case TierFree:
		return TierGrowth
	case TierGrowth:
		return TierScale
	default:
		return TierScale
	}
}

func NormalizeTier(tier string) Tier {
	switch strings.ToLower(strings.TrimSpace(tier)) {
	case string(TierGrowth):
		return TierGrowth
	case string(TierScale):
		return TierScale
	default:
		return TierFree
	}
}

func IsPaidTier(tier string) bool {
	t := NormalizeTier(tier)
	return t == TierGrowth || t == TierScale
}
