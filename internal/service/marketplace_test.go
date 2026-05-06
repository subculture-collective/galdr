package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
	connectorsdk "github.com/onnwee/pulse-score/pkg/connector-sdk"
)

type mockMarketplaceRepository struct {
	createConnectorFn         func(ctx context.Context, connector *repository.MarketplaceConnector) error
	getConnectorFn            func(ctx context.Context, id, version string) (*repository.MarketplaceConnector, error)
	listPublishedConnectorsFn func(ctx context.Context) ([]*repository.MarketplaceConnector, error)
	createInstallationFn      func(ctx context.Context, installation *repository.ConnectorInstallation) error
	incrementInstallMetricFn  func(ctx context.Context, connectorID string, at time.Time) error
	getAnalyticsFn            func(ctx context.Context, connectorID string, since time.Time) (*repository.ConnectorAnalytics, error)
	createReviewResultFn      func(ctx context.Context, result *repository.ConnectorReviewResult) error
	updateConnectorStatusFn   func(ctx context.Context, id, version, status string) error
}

func (m *mockMarketplaceRepository) CreateConnector(ctx context.Context, connector *repository.MarketplaceConnector) error {
	return m.createConnectorFn(ctx, connector)
}

func (m *mockMarketplaceRepository) GetConnector(ctx context.Context, id, version string) (*repository.MarketplaceConnector, error) {
	return m.getConnectorFn(ctx, id, version)
}

func (m *mockMarketplaceRepository) ListPublishedConnectors(ctx context.Context) ([]*repository.MarketplaceConnector, error) {
	return m.listPublishedConnectorsFn(ctx)
}

func (m *mockMarketplaceRepository) CreateInstallation(ctx context.Context, installation *repository.ConnectorInstallation) error {
	return m.createInstallationFn(ctx, installation)
}

func (m *mockMarketplaceRepository) IncrementConnectorInstallMetric(ctx context.Context, connectorID string, at time.Time) error {
	return m.incrementInstallMetricFn(ctx, connectorID, at)
}

func (m *mockMarketplaceRepository) GetConnectorAnalytics(ctx context.Context, connectorID string, since time.Time) (*repository.ConnectorAnalytics, error) {
	return m.getAnalyticsFn(ctx, connectorID, since)
}

func (m *mockMarketplaceRepository) CreateReviewResult(ctx context.Context, result *repository.ConnectorReviewResult) error {
	return m.createReviewResultFn(ctx, result)
}

func (m *mockMarketplaceRepository) UpdateConnectorStatus(ctx context.Context, id, version, status string) error {
	return m.updateConnectorStatusFn(ctx, id, version, status)
}

func TestMarketplaceRegisterValidatesManifestAndDefaultsDraft(t *testing.T) {
	developerID := uuid.New()
	manifest := validMarketplaceManifest("mock-crm", "1.0.0")
	repo := &mockMarketplaceRepository{
		getConnectorFn: func(ctx context.Context, id, version string) (*repository.MarketplaceConnector, error) {
			if id != manifest.ID || version != manifest.Version {
				t.Fatalf("unexpected connector lookup %s %s", id, version)
			}
			return nil, nil
		},
		createConnectorFn: func(ctx context.Context, connector *repository.MarketplaceConnector) error {
			if connector.ID != manifest.ID || connector.Version != manifest.Version {
				t.Fatalf("connector id/version not copied from manifest")
			}
			if connector.DeveloperID != developerID {
				t.Fatalf("expected developer id %s, got %s", developerID, connector.DeveloperID)
			}
			if connector.Status != repository.MarketplaceConnectorStatusDraft {
				t.Fatalf("expected draft status, got %q", connector.Status)
			}
			if connector.PublishedAt != nil {
				t.Fatal("draft connector should not have published_at")
			}
			return nil
		},
	}

	connector, err := NewMarketplaceService(repo).Register(context.Background(), developerID, RegisterConnectorRequest{Manifest: manifest})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if connector.Status != repository.MarketplaceConnectorStatusDraft {
		t.Fatalf("expected draft status, got %q", connector.Status)
	}
}

func TestMarketplaceRegisterRejectsInvalidManifest(t *testing.T) {
	_, err := NewMarketplaceService(&mockMarketplaceRepository{}).Register(context.Background(), uuid.New(), RegisterConnectorRequest{
		Manifest: connectorsdk.ConnectorManifest{ID: "bad id"},
	})

	if err == nil {
		t.Fatal("expected validation error")
	}
	if _, ok := err.(*ValidationError); !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
}

func TestMarketplaceListPublishedReturnsLatestVersionPerConnector(t *testing.T) {
	oldConnector := marketplaceConnector("mock-crm", "1.2.0", repository.MarketplaceConnectorStatusPublished)
	newConnector := marketplaceConnector("mock-crm", "1.10.0", repository.MarketplaceConnectorStatusPublished)
	otherConnector := marketplaceConnector("analytics", "1.0.0", repository.MarketplaceConnectorStatusPublished)
	repo := &mockMarketplaceRepository{
		listPublishedConnectorsFn: func(ctx context.Context) ([]*repository.MarketplaceConnector, error) {
			return []*repository.MarketplaceConnector{oldConnector, otherConnector, newConnector}, nil
		},
	}

	connectors, err := NewMarketplaceService(repo).ListPublished(context.Background())
	if err != nil {
		t.Fatalf("list published failed: %v", err)
	}
	if len(connectors) != 2 {
		t.Fatalf("expected 2 latest connectors, got %d", len(connectors))
	}
	if connectors[0].ID != "analytics" || connectors[1].ID != "mock-crm" {
		t.Fatalf("expected connectors sorted by id, got %s %s", connectors[0].ID, connectors[1].ID)
	}
	if connectors[1].Version != "1.10.0" {
		t.Fatalf("expected latest semver 1.10.0, got %q", connectors[1].Version)
	}
}

func TestMarketplaceListPublishedPrefersStableOverPrerelease(t *testing.T) {
	repo := &mockMarketplaceRepository{
		listPublishedConnectorsFn: func(ctx context.Context) ([]*repository.MarketplaceConnector, error) {
			return []*repository.MarketplaceConnector{
				marketplaceConnector("mock-crm", "1.0.0-beta.1", repository.MarketplaceConnectorStatusPublished),
				marketplaceConnector("mock-crm", "1.0.0", repository.MarketplaceConnectorStatusPublished),
			}, nil
		},
	}

	connectors, err := NewMarketplaceService(repo).ListPublished(context.Background())
	if err != nil {
		t.Fatalf("list published failed: %v", err)
	}
	if len(connectors) != 1 {
		t.Fatalf("expected 1 latest connector, got %d", len(connectors))
	}
	if connectors[0].Version != "1.0.0" {
		t.Fatalf("expected stable 1.0.0, got %q", connectors[0].Version)
	}
}

func TestMarketplaceInstallUsesLatestPublishedVersion(t *testing.T) {
	orgID := uuid.New()
	latest := marketplaceConnector("mock-crm", "2.0.0", repository.MarketplaceConnectorStatusPublished)
	metricRecorded := false
	repo := &mockMarketplaceRepository{
		listPublishedConnectorsFn: func(ctx context.Context) ([]*repository.MarketplaceConnector, error) {
			return []*repository.MarketplaceConnector{marketplaceConnector("mock-crm", "1.0.0", repository.MarketplaceConnectorStatusPublished), latest}, nil
		},
		createInstallationFn: func(ctx context.Context, installation *repository.ConnectorInstallation) error {
			if installation.OrgID != orgID {
				t.Fatalf("expected org id %s, got %s", orgID, installation.OrgID)
			}
			if installation.ConnectorID != "mock-crm" || installation.ConnectorVersion != "2.0.0" {
				t.Fatalf("expected latest connector version, got %s@%s", installation.ConnectorID, installation.ConnectorVersion)
			}
			if installation.Status != repository.ConnectorInstallationStatusActive {
				t.Fatalf("expected active installation, got %q", installation.Status)
			}
			return nil
		},
		incrementInstallMetricFn: func(ctx context.Context, connectorID string, at time.Time) error {
			if connectorID != "mock-crm" {
				t.Fatalf("expected install metric for mock-crm, got %s", connectorID)
			}
			if at.IsZero() {
				t.Fatal("expected install metric timestamp")
			}
			metricRecorded = true
			return nil
		},
	}

	installation, err := NewMarketplaceService(repo).Install(context.Background(), orgID, "mock-crm", InstallConnectorRequest{Config: map[string]any{"region": "us"}})
	if err != nil {
		t.Fatalf("install failed: %v", err)
	}
	if installation.ConnectorVersion != latest.Version {
		t.Fatalf("expected connector version %s, got %s", latest.Version, installation.ConnectorVersion)
	}
	if !metricRecorded {
		t.Fatal("expected install metric to be recorded")
	}
}

func TestMarketplaceAnalyticsReturnsConnectorMetrics(t *testing.T) {
	repo := &mockMarketplaceRepository{
		getAnalyticsFn: func(ctx context.Context, connectorID string, since time.Time) (*repository.ConnectorAnalytics, error) {
			if connectorID != "mock-crm" {
				t.Fatalf("expected analytics for mock-crm, got %s", connectorID)
			}
			if since.IsZero() {
				t.Fatal("expected analytics since timestamp")
			}
			return &repository.ConnectorAnalytics{
				ConnectorID:     connectorID,
				InstallCount:    10,
				ActiveInstalls:  8,
				SyncSuccessRate: 90,
				ErrorRate:       10,
				Metrics: []repository.ConnectorDailyMetric{
					{ConnectorID: connectorID, InstallCount: 2},
				},
			}, nil
		},
	}

	analytics, err := NewMarketplaceService(repo).Analytics(context.Background(), "mock-crm")
	if err != nil {
		t.Fatalf("analytics failed: %v", err)
	}
	if analytics.ConnectorID != "mock-crm" || analytics.InstallCount != 10 || len(analytics.Metrics) != 1 {
		t.Fatalf("unexpected analytics response: %+v", analytics)
	}
}

func validMarketplaceManifest(id, version string) connectorsdk.ConnectorManifest {
	return connectorsdk.ConnectorManifest{
		ID:          id,
		Name:        "Mock CRM",
		Version:     version,
		Description: "Mock CRM connector",
		Auth: connectorsdk.AuthConfig{
			Type: connectorsdk.AuthTypeAPIKey,
			APIKey: &connectorsdk.APIKeyConfig{
				HeaderName: "Authorization",
				Prefix:     "Bearer",
			},
		},
		Sync: connectorsdk.SyncConfig{
			SupportedModes: []connectorsdk.SyncMode{connectorsdk.SyncModeFull, connectorsdk.SyncModeIncremental},
			DefaultMode:    connectorsdk.SyncModeFull,
			Resources: []connectorsdk.ResourceConfig{
				{Name: "customers", Description: "CRM customers", Required: true},
			},
		},
	}
}

func marketplaceConnector(id, version, status string) *repository.MarketplaceConnector {
	return &repository.MarketplaceConnector{
		ID:          id,
		Version:     version,
		DeveloperID: uuid.New(),
		Name:        "Mock CRM",
		Description: "Mock CRM connector",
		Manifest:    validMarketplaceManifest(id, version),
		Status:      status,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}
