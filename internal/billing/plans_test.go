package billing

import "testing"

func TestGetPlanByTier(t *testing.T) {
	catalog := NewCatalog(PriceConfig{})

	plan, ok := catalog.GetPlanByTier("growth")
	if !ok {
		t.Fatal("expected growth plan to exist")
	}
	if plan.Tier != TierGrowth {
		t.Fatalf("expected growth tier, got %s", plan.Tier)
	}

	free, ok := catalog.GetPlanByTier("FREE")
	if !ok {
		t.Fatal("expected free plan to exist")
	}
	if free.Tier != TierFree {
		t.Fatalf("expected free tier, got %s", free.Tier)
	}
}

func TestGetLimits(t *testing.T) {
	catalog := NewCatalog(PriceConfig{})

	freeLimits, ok := catalog.GetLimits("free")
	if !ok {
		t.Fatal("expected free limits")
	}
	if freeLimits.CustomerLimit != 10 {
		t.Fatalf("expected free customer limit 10, got %d", freeLimits.CustomerLimit)
	}
	if freeLimits.IntegrationLimit != 1 {
		t.Fatalf("expected free integration limit 1, got %d", freeLimits.IntegrationLimit)
	}

	scaleLimits, ok := catalog.GetLimits("scale")
	if !ok {
		t.Fatal("expected scale limits")
	}
	if scaleLimits.CustomerLimit != Unlimited {
		t.Fatalf("expected scale customer limit unlimited, got %d", scaleLimits.CustomerLimit)
	}
}

func TestTierSpecifications(t *testing.T) {
	catalog := NewCatalog(PriceConfig{})

	tests := []struct {
		name             string
		tier             Tier
		customerLimit    int
		integrationLimit int
		teamMemberLimit  int
		features         map[string]bool
	}{
		{
			name:             "free",
			tier:             TierFree,
			customerLimit:    10,
			integrationLimit: 1,
			teamMemberLimit:  1,
			features: map[string]bool{
				FeatureBasicDashboard: true,
				FeatureFullDashboard:  false,
				FeatureEmailAlerts:    true,
				FeaturePlaybooks:      false,
				FeatureAIInsights:     false,
				FeatureBenchmarks:     false,
			},
		},
		{
			name:             "growth",
			tier:             TierGrowth,
			customerLimit:    500,
			integrationLimit: 3,
			teamMemberLimit:  5,
			features: map[string]bool{
				FeatureBasicDashboard: true,
				FeatureFullDashboard:  true,
				FeatureEmailAlerts:    true,
				FeaturePlaybooks:      true,
				FeatureAIInsights:     false,
				FeatureBenchmarks:     false,
			},
		},
		{
			name:             "scale",
			tier:             TierScale,
			customerLimit:    Unlimited,
			integrationLimit: Unlimited,
			teamMemberLimit:  Unlimited,
			features: map[string]bool{
				FeatureBasicDashboard: true,
				FeatureFullDashboard:  true,
				FeatureEmailAlerts:    true,
				FeaturePlaybooks:      true,
				FeatureAIInsights:     true,
				FeatureBenchmarks:     true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, ok := catalog.GetPlanByTier(string(tt.tier))
			if !ok {
				t.Fatalf("expected %s plan", tt.tier)
			}

			if plan.Limits.CustomerLimit != tt.customerLimit || plan.Limits.IntegrationLimit != tt.integrationLimit || plan.Limits.TeamMemberLimit != tt.teamMemberLimit {
				t.Fatalf("unexpected limits for %s: %+v", tt.tier, plan.Limits)
			}

			for feature, want := range tt.features {
				got, ok := catalog.HasFeature(string(tt.tier), feature)
				if !ok {
					t.Fatalf("expected feature lookup for %s", feature)
				}
				if got != want {
					t.Fatalf("expected %s access for %s to be %v, got %v", feature, tt.tier, want, got)
				}
			}
		})
	}
}

func TestPriceMappingMonthlyAnnual(t *testing.T) {
	catalog := NewCatalog(PriceConfig{
		GrowthMonthly: "price_growth_monthly",
		GrowthAnnual:  "price_growth_annual",
		ScaleMonthly:  "price_scale_monthly",
		ScaleAnnual:   "price_scale_annual",
	})

	monthly, err := catalog.GetPriceID("growth", false)
	if err != nil {
		t.Fatalf("expected growth monthly price id, got error: %v", err)
	}
	if monthly != "price_growth_monthly" {
		t.Fatalf("expected growth monthly id, got %s", monthly)
	}

	annual, err := catalog.GetPriceID("growth", true)
	if err != nil {
		t.Fatalf("expected growth annual price id, got error: %v", err)
	}
	if annual != "price_growth_annual" {
		t.Fatalf("expected growth annual id, got %s", annual)
	}

	tier, cycle, ok := catalog.ResolveTierAndCycleByPriceID("price_scale_annual")
	if !ok {
		t.Fatal("expected price id mapping for scale annual")
	}
	if tier != TierScale {
		t.Fatalf("expected scale tier, got %s", tier)
	}
	if cycle != BillingCycleAnnual {
		t.Fatalf("expected annual cycle, got %s", cycle)
	}
}
