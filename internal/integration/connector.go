package integration

import (
	"context"
	"time"
)

// AuthType identifies how a connector is authorized.
type AuthType string

const (
	// AuthTypeOAuth2 uses a provider OAuth2 authorization flow.
	AuthTypeOAuth2 AuthType = "oauth2"
	// AuthTypeAPIKey uses direct API-key credentials.
	AuthTypeAPIKey AuthType = "api_key"
	// AuthTypeWebhook uses inbound signed webhooks only.
	AuthTypeWebhook AuthType = "webhook"
	// AuthTypeNone requires no provider authentication.
	AuthTypeNone AuthType = "none"
)

// SyncMode controls full versus incremental connector imports.
type SyncMode string

const (
	// SyncModeFull imports the provider's full supported dataset.
	SyncModeFull SyncMode = "full"
	// SyncModeIncremental imports provider changes since a checkpoint.
	SyncModeIncremental SyncMode = "incremental"
)

// StatusState describes current connector state for one organization.
type StatusState string

const (
	// StatusConnected means credentials exist and syncs may run.
	StatusConnected StatusState = "connected"
	// StatusDisconnected means the organization has no active connection.
	StatusDisconnected StatusState = "disconnected"
	// StatusError means the connector has credentials but last health check failed.
	StatusError StatusState = "error"
)

// ConnectConfig carries credentials and provider metadata for connection setup.
type ConnectConfig struct {
	OrgID    string
	Code     string
	APIKey   string
	Metadata map[string]string
}

// DisconnectConfig identifies the org/provider connection to remove.
type DisconnectConfig struct {
	OrgID string
}

// SyncOptions selects org, mode, and checkpoint for a connector sync.
type SyncOptions struct {
	OrgID string
	Mode  SyncMode
	Since *time.Time
}

// WebhookPayload carries a normalized inbound provider webhook.
type WebhookPayload struct {
	OrgID      string
	EventType  string
	Headers    map[string]string
	Body       []byte
	ReceivedAt time.Time
}

// StatusOptions identifies the org/provider status to fetch.
type StatusOptions struct {
	OrgID string
}

// SyncResult summarizes resources imported by a connector.
type SyncResult struct {
	Resources []ResourceResult
	Cursor    string
}

// ResourceResult summarizes one provider resource processed during sync.
type ResourceResult struct {
	Name    string
	Synced  int
	Skipped int
	Error   string
}

// Status is an org-scoped connector health/status response.
type Status struct {
	State             StatusState
	ExternalAccountID string
	LastSyncAt        *time.Time
	LastError         string
	ConnectedAt       *time.Time
	Metadata          map[string]string
}

// Connector is the internal lifecycle contract all integration providers expose.
type Connector interface {
	Name() string
	AuthType() AuthType
	Connect(context.Context, ConnectConfig) error
	Disconnect(context.Context, DisconnectConfig) error
	Sync(context.Context, SyncOptions) (SyncResult, error)
	HandleWebhook(context.Context, WebhookPayload) error
	GetStatus(context.Context, StatusOptions) Status
}
