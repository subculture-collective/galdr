package billing

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	planmodel "github.com/onnwee/pulse-score/internal/billing"
	"github.com/onnwee/pulse-score/internal/repository"
)

const (
	UsageMetricCustomers    = "customer_count"
	UsageMetricIntegrations = "integration_count"
	UsageMetricPlaybooks    = "playbook_count"
	UsageMetricTeamMembers  = "team_member_count"
	UsageMetricAPIRequests  = "api_requests_count"
)

type playbookCounter interface {
	CountByOrg(ctx context.Context, orgID uuid.UUID) (int, error)
}

type usageSnapshotStore interface {
	Record(ctx context.Context, record repository.UsageSnapshotRecord) error
	Increment(ctx context.Context, orgID uuid.UUID, metric string, recordedAt time.Time) error
	CurrentValue(ctx context.Context, orgID uuid.UUID, metric string, recordedAt time.Time) (int, error)
}

// UsageServiceDeps groups usage analytics dependencies.
type UsageServiceDeps struct {
	Subscriptions orgSubscriptionReader
	Organizations organizationReader
	Customers     customerCounter
	Integrations  integrationCounter
	Playbooks     playbookCounter
	Snapshots     usageSnapshotStore
	Catalog       *planmodel.Catalog
	Overrides     featureOverrideReader
	Now           func() time.Time
}

// UsageCounter contains current usage against an optional limit.
type UsageCounter struct {
	Used  int `json:"used"`
	Limit int `json:"limit"`
}

// UsageSummary is returned by GET /api/v1/billing/usage.
type UsageSummary struct {
	RecordedAt       time.Time    `json:"recorded_at"`
	CustomerCount    UsageCounter `json:"customer_count"`
	IntegrationCount UsageCounter `json:"integration_count"`
	PlaybookCount    UsageCounter `json:"playbook_count"`
	TeamMemberCount  UsageCounter `json:"team_member_count"`
	APIRequestsCount UsageCounter `json:"api_requests_count"`
}

// UsageService tracks metered resource usage per organization.
type UsageService struct {
	subscriptions orgSubscriptionReader
	orgs          organizationReader
	customers     customerCounter
	integrations  integrationCounter
	playbooks     playbookCounter
	snapshots     usageSnapshotStore
	catalog       *planmodel.Catalog
	overrides     featureOverrideReader
	now           func() time.Time
}

func NewUsageService(deps UsageServiceDeps) *UsageService {
	now := deps.Now
	if now == nil {
		now = time.Now
	}
	return &UsageService{
		subscriptions: deps.Subscriptions,
		orgs:          deps.Organizations,
		customers:     deps.Customers,
		integrations:  deps.Integrations,
		playbooks:     deps.Playbooks,
		snapshots:     deps.Snapshots,
		catalog:       deps.Catalog,
		overrides:     deps.Overrides,
		now:           now,
	}
}

func (s *UsageService) GetUsage(ctx context.Context, orgID uuid.UUID) (*UsageSummary, error) {
	recordedAt := s.now().UTC()
	limits, err := s.currentLimits(ctx, orgID)
	if err != nil {
		return nil, err
	}

	customers, err := s.customers.CountByOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("count customers: %w", err)
	}
	integrations, err := s.integrations.CountActiveByOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("count integrations: %w", err)
	}
	playbooks, err := s.playbooks.CountByOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("count playbooks: %w", err)
	}
	members, err := s.orgs.CountMembers(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("count members: %w", err)
	}
	apiRequests, err := s.snapshots.CurrentValue(ctx, orgID, UsageMetricAPIRequests, recordedAt)
	if err != nil {
		return nil, fmt.Errorf("count api requests: %w", err)
	}

	for metric, value := range map[string]int{
		UsageMetricCustomers:    customers,
		UsageMetricIntegrations: integrations,
		UsageMetricPlaybooks:    playbooks,
		UsageMetricTeamMembers:  members,
	} {
		if err := s.snapshots.Record(ctx, repository.UsageSnapshotRecord{OrgID: orgID, Metric: metric, Value: value, RecordedAt: recordedAt}); err != nil {
			return nil, fmt.Errorf("record usage snapshot: %w", err)
		}
	}

	return &UsageSummary{
		RecordedAt:       recordedAt,
		CustomerCount:    UsageCounter{Used: customers, Limit: limits.CustomerLimit},
		IntegrationCount: UsageCounter{Used: integrations, Limit: limits.IntegrationLimit},
		PlaybookCount:    UsageCounter{Used: playbooks, Limit: -1},
		TeamMemberCount:  UsageCounter{Used: members, Limit: limits.TeamMemberLimit},
		APIRequestsCount: UsageCounter{Used: apiRequests, Limit: -1},
	}, nil
}

func (s *UsageService) RecordAPIRequest(ctx context.Context, orgID uuid.UUID) error {
	return s.snapshots.Increment(ctx, orgID, UsageMetricAPIRequests, s.now().UTC())
}

func (s *UsageService) RecordDailySnapshot(ctx context.Context, orgID uuid.UUID) error {
	_, err := s.GetUsage(ctx, orgID)
	return err
}

func (s *UsageService) currentLimits(ctx context.Context, orgID uuid.UUID) (planmodel.UsageLimits, error) {
	subSvc := NewSubscriptionService(s.subscriptions, s.orgs, s.customers, s.integrations, s.catalog)
	if s.overrides != nil {
		subSvc.SetFeatureOverrides(s.overrides)
	}
	return subSvc.GetUsageLimits(ctx, orgID)
}
