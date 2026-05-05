package integration

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestRegistryRegistersAndCreatesConnector(t *testing.T) {
	registry := NewRegistry()
	connector := &recordingConnector{name: "mock-crm", authType: AuthTypeAPIKey}

	if err := registry.Register(connector.Name(), func() Connector { return connector }); err != nil {
		t.Fatalf("register connector: %v", err)
	}

	created, ok := registry.Get("mock-crm")
	if !ok {
		t.Fatal("expected registered connector")
	}

	ctx := context.Background()
	if err := created.Connect(ctx, ConnectConfig{OrgID: "org_123", APIKey: "key"}); err != nil {
		t.Fatalf("connect: %v", err)
	}
	result, err := created.Sync(ctx, SyncOptions{OrgID: "org_123", Mode: SyncModeIncremental})
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if err := created.HandleWebhook(ctx, WebhookPayload{OrgID: "org_123", EventType: "customer.updated", ReceivedAt: time.Now()}); err != nil {
		t.Fatalf("handle webhook: %v", err)
	}
	if err := created.Disconnect(ctx, DisconnectConfig{OrgID: "org_123"}); err != nil {
		t.Fatalf("disconnect: %v", err)
	}

	if created.Name() != "mock-crm" || created.AuthType() != AuthTypeAPIKey {
		t.Fatalf("unexpected connector identity: %s/%s", created.Name(), created.AuthType())
	}
	if result.Resources[0].Name != "customers" || result.Resources[0].Synced != 2 {
		t.Fatalf("unexpected sync result: %#v", result.Resources)
	}
	if status := created.GetStatus(ctx, StatusOptions{OrgID: "org_123"}); status.State != StatusDisconnected || status.LastError != "" {
		t.Fatalf("unexpected status after disconnect: %#v", status)
	}
	if connector.connectedOrg != "" || !connector.webhookHandled {
		t.Fatalf("expected lifecycle calls to update connector state: %#v", connector)
	}
}

func TestRegistryRejectsInvalidRegistrations(t *testing.T) {
	registry := NewRegistry()
	connector := &recordingConnector{name: "mock-crm", authType: AuthTypeAPIKey}

	if err := registry.Register("", func() Connector { return connector }); err == nil {
		t.Fatal("expected empty connector name to fail")
	}
	if err := registry.Register("mock-crm", nil); err == nil {
		t.Fatal("expected nil connector factory to fail")
	}
	if err := registry.Register("mock-crm", func() Connector { return connector }); err != nil {
		t.Fatalf("register connector: %v", err)
	}
	if err := registry.Register("mock-crm", func() Connector { return connector }); err == nil {
		t.Fatal("expected duplicate connector registration to fail")
	}
}

func TestNilRegistryIsSafe(t *testing.T) {
	var registry *Registry

	if err := registry.Register("mock-crm", func() Connector {
		return &recordingConnector{name: "mock-crm", authType: AuthTypeAPIKey}
	}); err == nil {
		t.Fatal("expected nil registry registration to fail")
	}
	if connector, ok := registry.Get("mock-crm"); ok || connector != nil {
		t.Fatalf("expected nil registry lookup to miss, got %#v", connector)
	}
	if names := registry.Names(); names != nil {
		t.Fatalf("expected nil names, got %#v", names)
	}
}

func TestRegistryNamesAreSorted(t *testing.T) {
	registry := NewRegistry()
	for _, name := range []string{"stripe", "hubspot", "intercom"} {
		name := name
		if err := registry.Register(name, func() Connector {
			return &recordingConnector{name: name, authType: AuthTypeOAuth2}
		}); err != nil {
			t.Fatalf("register %s: %v", name, err)
		}
	}

	if names := registry.Names(); !reflect.DeepEqual(names, []string{"hubspot", "intercom", "stripe"}) {
		t.Fatalf("unexpected names: %#v", names)
	}
}

type recordingConnector struct {
	name           string
	authType       AuthType
	connectedOrg   string
	webhookHandled bool
}

func (c *recordingConnector) Name() string { return c.name }

func (c *recordingConnector) AuthType() AuthType { return c.authType }

func (c *recordingConnector) Connect(_ context.Context, config ConnectConfig) error {
	c.connectedOrg = config.OrgID
	return nil
}

func (c *recordingConnector) Disconnect(_ context.Context, _ DisconnectConfig) error {
	c.connectedOrg = ""
	return nil
}

func (c *recordingConnector) Sync(context.Context, SyncOptions) (SyncResult, error) {
	return SyncResult{Resources: []ResourceResult{{Name: "customers", Synced: 2}}}, nil
}

func (c *recordingConnector) HandleWebhook(_ context.Context, _ WebhookPayload) error {
	c.webhookHandled = true
	return nil
}

func (c *recordingConnector) GetStatus(context.Context, StatusOptions) Status {
	if c.connectedOrg == "" {
		return Status{State: StatusDisconnected}
	}
	return Status{State: StatusConnected}
}
