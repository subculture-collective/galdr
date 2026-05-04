// Package main demonstrates a minimal PulseScore marketplace connector.
package main

import (
	"context"
	"fmt"
	"time"

	connectorsdk "github.com/onnwee/pulse-score/pkg/connector-sdk"
)

type mockConnector struct{}

func (mockConnector) Manifest() connectorsdk.ConnectorManifest {
	return connectorsdk.ConnectorManifest{
		ID:          "mock-crm",
		Name:        "Mock CRM",
		Version:     "1.0.0",
		Description: "Example CRM connector for PulseScore marketplace developers.",
		Categories:  []string{"crm", "example"},
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
				{Name: "customers", Description: "CRM account records", Required: true},
				{Name: "events", Description: "CRM activity timeline", Required: false},
			},
			Schedule: &connectorsdk.SyncSchedule{IntervalMinutes: 360},
		},
		Webhooks: []connectorsdk.WebhookConfig{
			{Path: "/api/v1/webhooks/connectors/mock-crm", EventTypes: []string{"customer.updated"}, SigningSecretHeader: "X-Mock-CRM-Signature"},
		},
	}
}

func (mockConnector) Authenticate(ctx context.Context, req connectorsdk.AuthRequest) (*connectorsdk.AuthResult, error) {
	if req.APIKey == "" {
		return nil, fmt.Errorf("api key is required")
	}
	return &connectorsdk.AuthResult{
		ExternalAccountID: "mock-account-123",
		Scopes:            []string{"customers:read", "events:read"},
	}, nil
}

func (mockConnector) Sync(ctx context.Context, req connectorsdk.SyncRequest) (*connectorsdk.SyncResult, error) {
	now := time.Now().UTC()
	return &connectorsdk.SyncResult{
		Resources: []connectorsdk.ResourceResult{
			{Name: "customers", Synced: 1},
			{Name: "events", Synced: 1},
		},
		Customers: []connectorsdk.CustomerRecord{
			{ExternalID: "mock-customer-1", Email: "owner@example.com", Name: "Example Customer", CompanyName: "Example Co", LastSeenAt: &now},
		},
		Events: []connectorsdk.CustomerEvent{
			{ExternalCustomerID: "mock-customer-1", Type: "crm.activity", OccurredAt: &now, Data: map[string]any{"source": "mock-crm"}},
		},
	}, nil
}

func (mockConnector) HandleEvent(ctx context.Context, req connectorsdk.EventRequest) (*connectorsdk.EventResult, error) {
	return &connectorsdk.EventResult{
		Accepted: true,
		Events: []connectorsdk.CustomerEvent{
			{ExternalCustomerID: "mock-customer-1", Type: req.EventType, OccurredAt: &req.ReceivedAt},
		},
	}, nil
}

func main() {
	registry := connectorsdk.NewRegistry()
	registered, err := registry.Register(mockConnector{})
	if err != nil {
		panic(err)
	}
	fmt.Printf("registered %s connector with SDK %s\n", registered.Manifest.ID, connectorsdk.SDKVersion)
}
