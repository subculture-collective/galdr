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
