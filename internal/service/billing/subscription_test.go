package billing

import (
	"context"
	"testing"

	"github.com/google/uuid"

	planmodel "github.com/onnwee/pulse-score/internal/billing"
	"github.com/onnwee/pulse-score/internal/repository"
)

type mockOrgSubscriptionReader struct {
	getByOrgFn func(ctx context.Context, orgID uuid.UUID) (*repository.OrgSubscription, error)
}

func (m *mockOrgSubscriptionReader) GetByOrg(ctx context.Context, orgID uuid.UUID) (*repository.OrgSubscription, error) {
	return m.getByOrgFn(ctx, orgID)
}

type mockOrganizationReader struct {
	getByIDFn       func(ctx context.Context, id uuid.UUID) (*repository.Organization, error)
	countMembersFn func(ctx context.Context, orgID uuid.UUID) (int, error)
}

func (m *mockOrganizationReader) GetByID(ctx context.Context, id uuid.UUID) (*repository.Organization, error) {
	return m.getByIDFn(ctx, id)
}

func (m *mockOrganizationReader) CountMembers(ctx context.Context, orgID uuid.UUID) (int, error) {
	if m.countMembersFn == nil {
		return 0, nil
	}
	return m.countMembersFn(ctx, orgID)
}

type mockCustomerCounter struct {
	countByOrgFn func(ctx context.Context, orgID uuid.UUID) (int, error)
}

func (m *mockCustomerCounter) CountByOrg(ctx context.Context, orgID uuid.UUID) (int, error) {
	return m.countByOrgFn(ctx, orgID)
}

type mockIntegrationCounter struct {
	countActiveByOrgFn func(ctx context.Context, orgID uuid.UUID) (int, error)
}

func (m *mockIntegrationCounter) CountActiveByOrg(ctx context.Context, orgID uuid.UUID) (int, error) {
	return m.countActiveByOrgFn(ctx, orgID)
}

func TestGetCurrentPlan_DefaultsToFreeWhenNoSubscriptionRow(t *testing.T) {
	svc := NewSubscriptionService(
		&mockOrgSubscriptionReader{getByOrgFn: func(context.Context, uuid.UUID) (*repository.OrgSubscription, error) {
			return nil, nil
		}},
		&mockOrganizationReader{getByIDFn: func(context.Context, uuid.UUID) (*repository.Organization, error) {
			return nil, nil
		}},
		&mockCustomerCounter{countByOrgFn: func(context.Context, uuid.UUID) (int, error) { return 0, nil }},
		&mockIntegrationCounter{countActiveByOrgFn: func(context.Context, uuid.UUID) (int, error) { return 0, nil }},
		planmodel.NewCatalog(planmodel.PriceConfig{}),
	)

	plan, err := svc.GetCurrentPlan(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if plan != string(planmodel.TierFree) {
		t.Fatalf("expected free plan fallback, got %s", plan)
	}
}

func TestIsActiveStatusTransitions(t *testing.T) {
	orgID := uuid.New()

	statuses := map[string]bool{
		"active":     true,
		"trialing":   true,
		"past_due":   true,
		"canceled":   false,
		"unpaid":     false,
		"incomplete": false,
	}

	for status, expected := range statuses {
		t.Run(status, func(t *testing.T) {
			svc := NewSubscriptionService(
				&mockOrgSubscriptionReader{getByOrgFn: func(context.Context, uuid.UUID) (*repository.OrgSubscription, error) {
					return &repository.OrgSubscription{OrgID: orgID, Status: status, PlanTier: "growth"}, nil
				}},
				&mockOrganizationReader{getByIDFn: func(context.Context, uuid.UUID) (*repository.Organization, error) {
					return &repository.Organization{ID: orgID, Plan: "growth"}, nil
				}},
				&mockCustomerCounter{countByOrgFn: func(context.Context, uuid.UUID) (int, error) { return 0, nil }},
				&mockIntegrationCounter{countActiveByOrgFn: func(context.Context, uuid.UUID) (int, error) { return 0, nil }},
				planmodel.NewCatalog(planmodel.PriceConfig{}),
			)

			active, err := svc.IsActive(context.Background(), orgID)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if active != expected {
				t.Fatalf("expected active=%v for status=%s, got %v", expected, status, active)
			}
		})
	}
}

func TestGetSubscriptionSummaryIncludesTeamMemberUsageLimit(t *testing.T) {
	orgID := uuid.New()
	svc := NewSubscriptionService(
		&mockOrgSubscriptionReader{getByOrgFn: func(context.Context, uuid.UUID) (*repository.OrgSubscription, error) {
			return &repository.OrgSubscription{OrgID: orgID, PlanTier: "growth", Status: "active"}, nil
		}},
		&mockOrganizationReader{
			getByIDFn: func(context.Context, uuid.UUID) (*repository.Organization, error) {
				return &repository.Organization{ID: orgID, Plan: "growth"}, nil
			},
			countMembersFn: func(context.Context, uuid.UUID) (int, error) { return 3, nil },
		},
		&mockCustomerCounter{countByOrgFn: func(context.Context, uuid.UUID) (int, error) { return 42, nil }},
		&mockIntegrationCounter{countActiveByOrgFn: func(context.Context, uuid.UUID) (int, error) { return 2, nil }},
		planmodel.NewCatalog(planmodel.PriceConfig{}),
	)

	summary, err := svc.GetSubscriptionSummary(context.Background(), orgID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if summary.Usage.TeamMembers.Used != 3 || summary.Usage.TeamMembers.Limit != 5 {
		t.Fatalf("expected growth team member usage 3/5, got %+v", summary.Usage.TeamMembers)
	}
}
