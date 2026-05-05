package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/onnwee/pulse-score/internal/integration"
	connectorsdk "github.com/onnwee/pulse-score/pkg/connector-sdk"
)

func TestConnectorSyncServiceUsesRegistryConnector(t *testing.T) {
	registry := connectorsdk.NewRegistry()
	connector := &recordingConnector{id: "mock-crm"}
	if _, err := registry.Register(connector); err != nil {
		t.Fatalf("register connector: %v", err)
	}

	svc := NewConnectorSyncService(registry)
	orgID := uuid.New()
	since := time.Now().Add(-time.Hour)

	result, err := svc.Sync(context.Background(), "mock-crm", orgID, connectorsdk.SyncModeIncremental, &since)
	if err != nil {
		t.Fatalf("sync returned error: %v", err)
	}

	if connector.requests != 1 {
		t.Fatalf("expected connector to be called once, got %d", connector.requests)
	}
	if connector.lastRequest.OrgID != orgID.String() {
		t.Fatalf("expected org id %q, got %q", orgID.String(), connector.lastRequest.OrgID)
	}
	if connector.lastRequest.Mode != connectorsdk.SyncModeIncremental {
		t.Fatalf("expected incremental mode, got %q", connector.lastRequest.Mode)
	}
	if connector.lastRequest.Since == nil || !connector.lastRequest.Since.Equal(since) {
		t.Fatalf("expected since %v, got %v", since, connector.lastRequest.Since)
	}
	if len(result.Resources) != 1 || result.Resources[0].Name != "customers" {
		t.Fatalf("unexpected resources: %#v", result.Resources)
	}
}

func TestConnectorSyncServiceMissingProvider(t *testing.T) {
	svc := NewConnectorSyncService(connectorsdk.NewRegistry())

	_, err := svc.Sync(context.Background(), "missing", uuid.New(), connectorsdk.SyncModeFull, nil)
	if err == nil {
		t.Fatal("expected missing connector error")
	}
}

func TestNewIntegrationConnectorRegistryContainsBuiltIns(t *testing.T) {
	registry, err := NewIntegrationConnectorRegistry(nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}

	for _, provider := range []string{"hubspot", "intercom", "posthog", "salesforce", "stripe", "zendesk"} {
		if _, ok := registry.Get(provider); !ok {
			t.Fatalf("expected %s connector to be registered", provider)
		}
	}
}

func TestNewBuiltInIntegrationRegistryContainsOriginalProviders(t *testing.T) {
	registry, err := NewBuiltInIntegrationRegistry(nil, nil, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("create registry: %v", err)
	}

	providers := map[string]integration.AuthType{
		"hubspot":  integration.AuthTypeOAuth2,
		"intercom": integration.AuthTypeOAuth2,
		"stripe":   integration.AuthTypeOAuth2,
	}
	for provider, authType := range providers {
		connector, ok := registry.Get(provider)
		if !ok {
			t.Fatalf("expected %s connector to be registered", provider)
		}
		if connector.Name() != provider || connector.AuthType() != authType {
			t.Fatalf("unexpected %s connector identity: %s/%s", provider, connector.Name(), connector.AuthType())
		}
	}
}

type recordingConnector struct {
	id          string
	requests    int
	lastRequest connectorsdk.SyncRequest
}

func (c *recordingConnector) Manifest() connectorsdk.ConnectorManifest {
	return connectorsdk.ConnectorManifest{
		ID:          c.id,
		Name:        "Mock CRM",
		Version:     "1.0.0",
		Description: "Test connector.",
		Auth: connectorsdk.AuthConfig{
			Type: connectorsdk.AuthTypeNone,
		},
		Sync: connectorsdk.SyncConfig{
			SupportedModes: []connectorsdk.SyncMode{connectorsdk.SyncModeFull, connectorsdk.SyncModeIncremental},
			DefaultMode:    connectorsdk.SyncModeFull,
			Resources: []connectorsdk.ResourceConfig{
				{Name: "customers", Description: "Customer records", Required: true},
			},
		},
	}
}

func (c *recordingConnector) Authenticate(context.Context, connectorsdk.AuthRequest) (*connectorsdk.AuthResult, error) {
	return &connectorsdk.AuthResult{}, nil
}

func (c *recordingConnector) Sync(_ context.Context, req connectorsdk.SyncRequest) (*connectorsdk.SyncResult, error) {
	c.requests++
	c.lastRequest = req
	return &connectorsdk.SyncResult{
		Resources: []connectorsdk.ResourceResult{{Name: "customers", Synced: 3}},
	}, nil
}

func (c *recordingConnector) HandleEvent(context.Context, connectorsdk.EventRequest) (*connectorsdk.EventResult, error) {
	return &connectorsdk.EventResult{Accepted: false}, nil
}
