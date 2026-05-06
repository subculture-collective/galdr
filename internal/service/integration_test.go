package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/onnwee/pulse-score/internal/repository"
	connectorsdk "github.com/onnwee/pulse-score/pkg/connector-sdk"
)

func TestIntegrationServiceListIncludesRegistryConnectors(t *testing.T) {
	orgID := uuid.New()
	connectedAt := time.Now().Add(-time.Hour)
	registry := connectorsdk.NewRegistry()
	for _, provider := range []string{"stripe", "zendesk"} {
		if _, err := registry.Register(&recordingConnector{id: provider}); err != nil {
			t.Fatalf("register %s: %v", provider, err)
		}
	}

	store := &fakeIntegrationStore{
		connections: []*repository.IntegrationConnection{
			{
				OrgID:     orgID,
				Provider:  "stripe",
				Status:    "active",
				CreatedAt: connectedAt,
			},
		},
		customerCounts: map[string]int{"stripe": 4},
	}
	svc := NewIntegrationService(store, nil, registry)

	summaries, err := svc.List(context.Background(), orgID)
	if err != nil {
		t.Fatalf("list integrations: %v", err)
	}

	if len(summaries) != 2 {
		t.Fatalf("expected connected and available integrations, got %#v", summaries)
	}
	if summaries[0].Provider != "stripe" || summaries[0].Status != "active" || summaries[0].CustomerCount != 4 {
		t.Fatalf("expected connected stripe summary first, got %#v", summaries[0])
	}
	if summaries[1].Provider != "zendesk" || summaries[1].Status != "disconnected" {
		t.Fatalf("expected disconnected registry connector, got %#v", summaries[1])
	}
}

func TestIntegrationServiceHealthFlagsStaleAndErroringConnections(t *testing.T) {
	orgID := uuid.New()
	now := time.Now().UTC()
	staleSync := now.Add(-25 * time.Hour)
	recentSync := now.Add(-time.Hour)
	registry := connectorsdk.NewRegistry()
	for _, provider := range []string{"stripe", "hubspot", "zendesk"} {
		if _, err := registry.Register(&recordingConnector{id: provider}); err != nil {
			t.Fatalf("register %s: %v", provider, err)
		}
	}

	store := &fakeIntegrationStore{
		connections: []*repository.IntegrationConnection{
			{
				OrgID:      orgID,
				Provider:   "stripe",
				Status:     "active",
				LastSyncAt: &staleSync,
				CreatedAt:  now.Add(-48 * time.Hour),
				Metadata: map[string]any{
					"records_synced":        120,
					"last_sync_duration_ms": 4300,
					"success_count":         8,
				},
			},
			{
				OrgID:          orgID,
				Provider:       "hubspot",
				Status:         "error",
				LastSyncAt:     &recentSync,
				LastSyncError:  "rate limited",
				CreatedAt:      now.Add(-72 * time.Hour),
				Metadata:       map[string]any{"error_count": 5, "success_count": 5},
			},
		},
		customerCounts: map[string]int{"stripe": 120, "hubspot": 80},
	}
	svc := NewIntegrationService(store, nil, registry)

	health, err := svc.GetHealth(context.Background(), orgID)
	if err != nil {
		t.Fatalf("get health: %v", err)
	}

	if len(health.Integrations) != 3 {
		t.Fatalf("expected registry and connected integrations, got %#v", health.Integrations)
	}
	stripe := health.Integrations[0]
	if stripe.Provider != "stripe" || stripe.HealthStatus != "warning" || len(stripe.Alerts) == 0 {
		t.Fatalf("expected stale stripe warning, got %#v", stripe)
	}
	if stripe.RecordsSynced != 120 || stripe.SyncDurationMS != 4300 || len(stripe.SyncHistory) != 1 {
		t.Fatalf("expected stripe sync metrics, got %#v", stripe)
	}
	hubspot := health.Integrations[1]
	if hubspot.Provider != "hubspot" || hubspot.HealthStatus != "down" || hubspot.ErrorRate != 0.5 {
		t.Fatalf("expected hubspot down with error rate, got %#v", hubspot)
	}
	zendesk := health.Integrations[2]
	if zendesk.Provider != "zendesk" || zendesk.HealthStatus != "disconnected" {
		t.Fatalf("expected disconnected registry connector, got %#v", zendesk)
	}
}

type fakeIntegrationStore struct {
	connections    []*repository.IntegrationConnection
	customerCounts map[string]int
}

func (s *fakeIntegrationStore) ListByOrg(context.Context, uuid.UUID) ([]*repository.IntegrationConnection, error) {
	return s.connections, nil
}

func (s *fakeIntegrationStore) GetByOrgAndProvider(_ context.Context, _ uuid.UUID, provider string) (*repository.IntegrationConnection, error) {
	for _, conn := range s.connections {
		if conn.Provider == provider {
			return conn, nil
		}
	}
	return nil, nil
}

func (s *fakeIntegrationStore) GetCustomerCountBySource(_ context.Context, _ uuid.UUID, provider string) (int, error) {
	return s.customerCounts[provider], nil
}

func (s *fakeIntegrationStore) Delete(context.Context, uuid.UUID, string) error { return nil }
