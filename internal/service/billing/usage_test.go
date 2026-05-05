package billing

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	planmodel "github.com/onnwee/pulse-score/internal/billing"
	"github.com/onnwee/pulse-score/internal/repository"
)

type mockPlaybookCounter struct {
	countByOrgFn func(ctx context.Context, orgID uuid.UUID) (int, error)
}

func (m *mockPlaybookCounter) CountByOrg(ctx context.Context, orgID uuid.UUID) (int, error) {
	return m.countByOrgFn(ctx, orgID)
}

type mockUsageSnapshotStore struct {
	recorded []repository.UsageSnapshotRecord
	apiCount int
}

func (m *mockUsageSnapshotStore) Record(ctx context.Context, record repository.UsageSnapshotRecord) error {
	m.recorded = append(m.recorded, record)
	return nil
}

func (m *mockUsageSnapshotStore) Increment(ctx context.Context, orgID uuid.UUID, metric string, recordedAt time.Time) error {
	m.apiCount++
	return nil
}

func (m *mockUsageSnapshotStore) CurrentValue(ctx context.Context, orgID uuid.UUID, metric string, recordedAt time.Time) (int, error) {
	if metric == UsageMetricAPIRequests {
		return m.apiCount, nil
	}
	return 0, nil
}

func TestUsageServiceReturnsCurrentUsageAndRecordsDailySnapshot(t *testing.T) {
	orgID := uuid.New()
	store := &mockUsageSnapshotStore{}
	now := time.Date(2026, 5, 5, 10, 0, 0, 0, time.UTC)
	svc := NewUsageService(UsageServiceDeps{
		Subscriptions: &mockOrgSubscriptionReader{getByOrgFn: func(context.Context, uuid.UUID) (*repository.OrgSubscription, error) {
			return &repository.OrgSubscription{OrgID: orgID, PlanTier: "growth", Status: "active"}, nil
		}},
		Organizations: &mockOrganizationReader{
			getByIDFn:       func(context.Context, uuid.UUID) (*repository.Organization, error) { return &repository.Organization{ID: orgID, Plan: "growth"}, nil },
			countMembersFn: func(context.Context, uuid.UUID) (int, error) { return 4, nil },
		},
		Customers:    &mockCustomerCounter{countByOrgFn: func(context.Context, uuid.UUID) (int, error) { return 42, nil }},
		Integrations: &mockIntegrationCounter{countActiveByOrgFn: func(context.Context, uuid.UUID) (int, error) { return 2, nil }},
		Playbooks:    &mockPlaybookCounter{countByOrgFn: func(context.Context, uuid.UUID) (int, error) { return 7, nil }},
		Snapshots:    store,
		Catalog:      planmodel.NewCatalog(planmodel.PriceConfig{}),
		Now:          func() time.Time { return now },
	})

	if err := svc.RecordAPIRequest(context.Background(), orgID); err != nil {
		t.Fatalf("record api request: %v", err)
	}

	summary, err := svc.GetUsage(context.Background(), orgID)
	if err != nil {
		t.Fatalf("get usage: %v", err)
	}

	if summary.CustomerCount.Used != 42 || summary.CustomerCount.Limit != 500 {
		t.Fatalf("expected customer usage 42/500, got %+v", summary.CustomerCount)
	}
	if summary.IntegrationCount.Used != 2 || summary.IntegrationCount.Limit != 3 {
		t.Fatalf("expected integration usage 2/3, got %+v", summary.IntegrationCount)
	}
	if summary.TeamMemberCount.Used != 4 || summary.TeamMemberCount.Limit != 5 {
		t.Fatalf("expected team member usage 4/5, got %+v", summary.TeamMemberCount)
	}
	if summary.PlaybookCount.Used != 7 || summary.PlaybookCount.Limit != -1 {
		t.Fatalf("expected playbook usage 7/unlimited, got %+v", summary.PlaybookCount)
	}
	if summary.APIRequestsCount.Used != 1 || summary.APIRequestsCount.Limit != -1 {
		t.Fatalf("expected api request usage 1/unlimited, got %+v", summary.APIRequestsCount)
	}

	if len(store.recorded) != 4 {
		t.Fatalf("expected four daily resource snapshots, got %d", len(store.recorded))
	}
	metrics := map[string]int{}
	for _, record := range store.recorded {
		metrics[record.Metric] = record.Value
	}
	if metrics[UsageMetricCustomers] != 42 || metrics[UsageMetricIntegrations] != 2 || metrics[UsageMetricTeamMembers] != 4 || metrics[UsageMetricPlaybooks] != 7 {
		t.Fatalf("unexpected recorded metrics: %+v", metrics)
	}
}
