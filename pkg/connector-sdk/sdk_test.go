package connectorsdk

import (
	"context"
	"strings"
	"testing"
)

type testConnector struct {
	manifest ConnectorManifest
}

func (c testConnector) Manifest() ConnectorManifest { return c.manifest }

func (c testConnector) Authenticate(ctx context.Context, req AuthRequest) (*AuthResult, error) {
	return &AuthResult{ExternalAccountID: "acct_123", Scopes: []string{"customers:read"}}, nil
}

func (c testConnector) Sync(ctx context.Context, req SyncRequest) (*SyncResult, error) {
	return &SyncResult{Resources: []ResourceResult{{Name: "customers", Synced: 2}}}, nil
}

func (c testConnector) HandleEvent(ctx context.Context, req EventRequest) (*EventResult, error) {
	return &EventResult{Accepted: true, Events: []CustomerEvent{{ExternalCustomerID: "cus_123", Type: "customer.updated"}}}, nil
}

func TestRegistryRegisterValidConnector(t *testing.T) {
	registry := NewRegistry()
	connector := testConnector{manifest: validManifest()}

	registered, err := registry.Register(connector)
	if err != nil {
		t.Fatalf("register connector: %v", err)
	}

	if registered.Manifest.ID != "mock-crm" {
		t.Fatalf("expected manifest ID mock-crm, got %q", registered.Manifest.ID)
	}
	if got := registry.List()[0].Manifest.Version; got != "1.2.3" {
		t.Fatalf("expected version 1.2.3, got %q", got)
	}
}

func TestRegistryGetReturnsRegisteredConnector(t *testing.T) {
	registry := NewRegistry()
	connector := testConnector{manifest: validManifest()}

	if _, err := registry.Register(connector); err != nil {
		t.Fatalf("register connector: %v", err)
	}

	registered, ok := registry.Get("mock-crm")
	if !ok {
		t.Fatal("expected registered connector")
	}
	if registered.Manifest.Name != "Mock CRM" {
		t.Fatalf("expected connector name Mock CRM, got %q", registered.Manifest.Name)
	}

	if _, ok := registry.Get("missing"); ok {
		t.Fatal("expected missing connector lookup to fail")
	}
}

func TestRegistryListOrdersConnectorsByManifestID(t *testing.T) {
	registry := NewRegistry()
	for _, manifest := range []ConnectorManifest{validManifestWithID("zendesk"), validManifestWithID("hubspot")} {
		if _, err := registry.Register(testConnector{manifest: manifest}); err != nil {
			t.Fatalf("register connector %q: %v", manifest.ID, err)
		}
	}

	registered := registry.List()
	if len(registered) != 2 {
		t.Fatalf("expected 2 registered connectors, got %d", len(registered))
	}
	if registered[0].Manifest.ID != "hubspot" || registered[1].Manifest.ID != "zendesk" {
		t.Fatalf("expected sorted connector IDs, got %#v", registered)
	}
}

func TestValidateManifestRejectsInvalidConnectorMetadata(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(*ConnectorManifest)
		wantError string
	}{
		{
			name:      "missing id",
			mutate:    func(m *ConnectorManifest) { m.ID = "" },
			wantError: "id is required",
		},
		{
			name:      "invalid semver",
			mutate:    func(m *ConnectorManifest) { m.Version = "v1" },
			wantError: "semantic version",
		},
		{
			name:      "missing description",
			mutate:    func(m *ConnectorManifest) { m.Description = "" },
			wantError: "description is required",
		},
		{
			name:      "api key without header",
			mutate:    func(m *ConnectorManifest) { m.Auth.APIKey.HeaderName = "" },
			wantError: "header_name",
		},
		{
			name:      "default mode unsupported",
			mutate:    func(m *ConnectorManifest) { m.Sync.DefaultMode = "delta" },
			wantError: "supported_modes",
		},
		{
			name: "supported mode unknown",
			mutate: func(m *ConnectorManifest) {
				m.Sync.SupportedModes = []SyncMode{SyncModeFull, "delta"}
			},
			wantError: "unsupported sync mode",
		},
		{
			name: "duplicate supported mode",
			mutate: func(m *ConnectorManifest) {
				m.Sync.SupportedModes = []SyncMode{SyncModeFull, SyncModeFull}
			},
			wantError: "duplicate sync mode",
		},
		{
			name: "duplicate resource name",
			mutate: func(m *ConnectorManifest) {
				m.Sync.Resources = append(m.Sync.Resources, ResourceConfig{
					Name:        "customers",
					Description: "Duplicate customer feed",
				})
			},
			wantError: "duplicate sync resource",
		},
		{
			name:      "webhook without events",
			mutate:    func(m *ConnectorManifest) { m.Webhooks[0].EventTypes = nil },
			wantError: "event types",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest := validManifest()
			tt.mutate(&manifest)

			err := ValidateManifest(manifest)
			if err == nil {
				t.Fatal("expected validation error")
			}
			assertErrorContains(t, err, tt.wantError)
		})
	}
}

func TestConnectorLifecycle(t *testing.T) {
	ctx := context.Background()
	connector := testConnector{manifest: validManifest()}

	auth, err := connector.Authenticate(ctx, AuthRequest{OrgID: "org_123", APIKey: "test_key"})
	if err != nil {
		t.Fatalf("authenticate: %v", err)
	}
	if auth.ExternalAccountID != "acct_123" {
		t.Fatalf("expected external account acct_123, got %q", auth.ExternalAccountID)
	}

	syncResult, err := connector.Sync(ctx, SyncRequest{OrgID: "org_123", ExternalAccountID: auth.ExternalAccountID, Mode: SyncModeFull})
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if len(syncResult.Resources) != 1 || syncResult.Resources[0].Synced != 2 {
		t.Fatalf("expected one resource with 2 synced records, got %#v", syncResult.Resources)
	}

	eventResult, err := connector.HandleEvent(ctx, EventRequest{OrgID: "org_123", EventType: "customer.updated"})
	if err != nil {
		t.Fatalf("handle event: %v", err)
	}
	if !eventResult.Accepted || len(eventResult.Events) != 1 {
		t.Fatalf("expected accepted event result, got %#v", eventResult)
	}
}

func validManifest() ConnectorManifest {
	return validManifestWithID("mock-crm")
}

func validManifestWithID(id string) ConnectorManifest {
	return ConnectorManifest{
		ID:          id,
		Name:        "Mock CRM",
		Version:     "1.2.3",
		Description: "Reference connector for SDK developers.",
		Auth: AuthConfig{
			Type: AuthTypeAPIKey,
			APIKey: &APIKeyConfig{
				HeaderName: "Authorization",
				Prefix:     "Bearer",
			},
		},
		Sync: SyncConfig{
			SupportedModes: []SyncMode{SyncModeFull, SyncModeIncremental},
			DefaultMode:    SyncModeFull,
			Resources: []ResourceConfig{
				{Name: "customers", Description: "Customer accounts", Required: true},
			},
		},
		Webhooks: []WebhookConfig{
			{Path: "/webhooks/mock-crm", EventTypes: []string{"customer.updated"}, SigningSecretHeader: "X-Mock-Signature"},
		},
	}
}

func assertErrorContains(t *testing.T, err error, want string) {
	t.Helper()
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("expected error containing %q, got %q", want, err.Error())
	}
}
