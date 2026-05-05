package billing

import (
	"context"
	"testing"

	"github.com/google/uuid"

	planmodel "github.com/onnwee/pulse-score/internal/billing"
	"github.com/onnwee/pulse-score/internal/repository"
)

type mockFeatureOverrideReader struct {
	getByOrgAndFeatureFn func(ctx context.Context, orgID uuid.UUID, featureName string) (*repository.FeatureOverride, error)
	listByOrgFn          func(ctx context.Context, orgID uuid.UUID) ([]repository.FeatureOverride, error)
}

func (m *mockFeatureOverrideReader) GetByOrgAndFeature(ctx context.Context, orgID uuid.UUID, featureName string) (*repository.FeatureOverride, error) {
	if m.getByOrgAndFeatureFn == nil {
		return nil, nil
	}
	return m.getByOrgAndFeatureFn(ctx, orgID, featureName)
}

func (m *mockFeatureOverrideReader) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]repository.FeatureOverride, error) {
	if m.listByOrgFn == nil {
		return nil, nil
	}
	return m.listByOrgFn(ctx, orgID)
}

func TestFeatureFlagServiceCanAccessUsesTierDefaults(t *testing.T) {
	orgID := uuid.New()
	svc := newFeatureFlagServiceForTest("growth", &mockFeatureOverrideReader{})

	allowed, err := svc.CanAccess(context.Background(), orgID, planmodel.FeaturePlaybooks)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !allowed {
		t.Fatal("expected growth tier to access playbooks")
	}

	allowed, err = svc.CanAccess(context.Background(), orgID, planmodel.FeatureAIInsights)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if allowed {
		t.Fatal("expected growth tier to deny ai insights by default")
	}
}

func TestFeatureFlagServiceCanAccessAppliesPerOrgOverride(t *testing.T) {
	orgID := uuid.New()
	enabled := true
	svc := newFeatureFlagServiceForTest("free", &mockFeatureOverrideReader{
		getByOrgAndFeatureFn: func(ctx context.Context, gotOrgID uuid.UUID, featureName string) (*repository.FeatureOverride, error) {
			if gotOrgID != orgID {
				t.Fatalf("expected org %s, got %s", orgID, gotOrgID)
			}
			if featureName != planmodel.FeatureAIInsights {
				t.Fatalf("expected feature %s, got %s", planmodel.FeatureAIInsights, featureName)
			}
			return &repository.FeatureOverride{OrgID: orgID, FeatureName: featureName, Enabled: &enabled}, nil
		},
	})

	allowed, err := svc.CanAccess(context.Background(), orgID, planmodel.FeatureAIInsights)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !allowed {
		t.Fatal("expected override to allow ai insights on free tier")
	}
}

func TestFeatureFlagServiceGetLimitAppliesPerOrgOverride(t *testing.T) {
	orgID := uuid.New()
	override := 25
	svc := newFeatureFlagServiceForTest("free", &mockFeatureOverrideReader{
		getByOrgAndFeatureFn: func(ctx context.Context, gotOrgID uuid.UUID, featureName string) (*repository.FeatureOverride, error) {
			if featureName != LimitCustomer {
				t.Fatalf("expected limit key %s, got %s", LimitCustomer, featureName)
			}
			return &repository.FeatureOverride{OrgID: gotOrgID, FeatureName: featureName, LimitOverride: &override}, nil
		},
	})

	limit, err := svc.GetLimit(context.Background(), orgID, LimitCustomer)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if limit != override {
		t.Fatalf("expected override limit %d, got %d", override, limit)
	}
}

func newFeatureFlagServiceForTest(tier string, overrides featureOverrideReader) *FeatureFlagService {
	return NewFeatureFlagService(
		&mockPlanResolver{tier: tier},
		overrides,
		planmodel.NewCatalog(planmodel.PriceConfig{}),
	)
}

type mockPlanResolver struct {
	tier string
}

func (m *mockPlanResolver) GetCurrentPlan(context.Context, uuid.UUID) (string, error) {
	return m.tier, nil
}
