// Package connectorsdk defines the public contract third-party connectors use
// to integrate external systems with the PulseScore integration marketplace.
package connectorsdk

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
)

const (
	// SDKVersion follows semantic versioning and changes when public connector
	// contracts change.
	SDKVersion = "0.1.0"

	connectorIDPattern     = `^[a-z0-9]+(?:-[a-z0-9]+)*$`
	semanticVersionPattern = `^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`
)

var (
	connectorIDPatternRE = regexp.MustCompile(connectorIDPattern)
	semverPattern        = regexp.MustCompile(semanticVersionPattern)
)

// Connector is the public lifecycle contract for marketplace integrations.
type Connector interface {
	Manifest() ConnectorManifest
	Authenticate(context.Context, AuthRequest) (*AuthResult, error)
	Sync(context.Context, SyncRequest) (*SyncResult, error)
	HandleEvent(context.Context, EventRequest) (*EventResult, error)
}

// ConnectorManifest describes connector metadata, auth, sync, and webhook shape.
type ConnectorManifest struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Version     string          `json:"version"`
	Description string          `json:"description"`
	Categories  []string        `json:"categories,omitempty"`
	Auth        AuthConfig      `json:"auth"`
	Sync        SyncConfig      `json:"sync"`
	Webhooks    []WebhookConfig `json:"webhooks,omitempty"`
}

// AuthType identifies the connector authentication strategy.
type AuthType string

const (
	// AuthTypeOAuth2 uses an OAuth 2 authorization-code flow.
	AuthTypeOAuth2 AuthType = "oauth2"
	// AuthTypeAPIKey uses a static API key supplied by the installing org.
	AuthTypeAPIKey AuthType = "api_key"
	// AuthTypeNone is for public data sources that need no credentials.
	AuthTypeNone AuthType = "none"
)

// AuthConfig describes how PulseScore authenticates with the external provider.
type AuthConfig struct {
	Type   AuthType      `json:"type"`
	OAuth2 *OAuth2Config `json:"oauth2,omitempty"`
	APIKey *APIKeyConfig `json:"api_key,omitempty"`
}

// OAuth2Config describes OAuth endpoints and requested scopes.
type OAuth2Config struct {
	AuthorizeURL string   `json:"authorize_url"`
	TokenURL     string   `json:"token_url"`
	Scopes       []string `json:"scopes,omitempty"`
}

// APIKeyConfig describes where an API key should be sent.
type APIKeyConfig struct {
	HeaderName string `json:"header_name"`
	Prefix     string `json:"prefix,omitempty"`
}

// SyncMode identifies the sync strategy requested by PulseScore.
type SyncMode string

const (
	// SyncModeFull imports the provider's full supported dataset.
	SyncModeFull SyncMode = "full"
	// SyncModeIncremental imports records modified since SyncRequest.Since.
	SyncModeIncremental SyncMode = "incremental"
)

// SyncConfig declares supported sync modes and resources.
type SyncConfig struct {
	SupportedModes []SyncMode        `json:"supported_modes"`
	DefaultMode    SyncMode          `json:"default_mode"`
	Resources      []ResourceConfig  `json:"resources"`
	Schedule       *SyncSchedule     `json:"schedule,omitempty"`
	Options        map[string]string `json:"options,omitempty"`
}

// SyncSchedule describes recommended recurring sync cadence.
type SyncSchedule struct {
	IntervalMinutes int `json:"interval_minutes"`
}

// ResourceConfig describes a provider object a connector can sync.
type ResourceConfig struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Required    bool   `json:"required"`
}

// WebhookConfig declares webhook route metadata for provider events.
type WebhookConfig struct {
	Path                string   `json:"path"`
	EventTypes          []string `json:"event_types"`
	SigningSecretHeader string   `json:"signing_secret_header,omitempty"`
}

// AuthRequest contains installation credentials for a connector.
type AuthRequest struct {
	OrgID       string            `json:"org_id"`
	RedirectURI string            `json:"redirect_uri,omitempty"`
	Code        string            `json:"code,omitempty"`
	APIKey      string            `json:"api_key,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// AuthResult contains authenticated external account metadata.
type AuthResult struct {
	ExternalAccountID string            `json:"external_account_id"`
	AccessToken       string            `json:"access_token,omitempty"`
	RefreshToken      string            `json:"refresh_token,omitempty"`
	ExpiresAt         *time.Time        `json:"expires_at,omitempty"`
	Scopes            []string          `json:"scopes,omitempty"`
	Metadata          map[string]string `json:"metadata,omitempty"`
}

// SyncRequest asks a connector to import provider resources.
type SyncRequest struct {
	OrgID             string            `json:"org_id"`
	ExternalAccountID string            `json:"external_account_id"`
	Mode              SyncMode          `json:"mode"`
	Since             *time.Time        `json:"since,omitempty"`
	Resources         []string          `json:"resources,omitempty"`
	Metadata          map[string]string `json:"metadata,omitempty"`
}

// SyncResult reports synced resources and normalized customer payloads.
type SyncResult struct {
	Resources []ResourceResult `json:"resources"`
	Customers []CustomerRecord `json:"customers,omitempty"`
	Events    []CustomerEvent  `json:"events,omitempty"`
	Cursor    string           `json:"cursor,omitempty"`
}

// ResourceResult reports import counts for one provider resource.
type ResourceResult struct {
	Name    string `json:"name"`
	Synced  int    `json:"synced"`
	Skipped int    `json:"skipped,omitempty"`
	Error   string `json:"error,omitempty"`
}

// CustomerRecord is the normalized customer shape accepted by PulseScore.
type CustomerRecord struct {
	ExternalID  string         `json:"external_id"`
	Email       string         `json:"email,omitempty"`
	Name        string         `json:"name,omitempty"`
	CompanyName string         `json:"company_name,omitempty"`
	MRRCents    int64          `json:"mrr_cents,omitempty"`
	Currency    string         `json:"currency,omitempty"`
	LastSeenAt  *time.Time     `json:"last_seen_at,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// EventRequest wraps a provider webhook event.
type EventRequest struct {
	OrgID       string            `json:"org_id"`
	EventType   string            `json:"event_type"`
	Headers     map[string]string `json:"headers,omitempty"`
	RawPayload  []byte            `json:"-"`
	ReceivedAt  time.Time         `json:"received_at"`
	WebhookPath string            `json:"webhook_path,omitempty"`
}

// EventResult reports whether a webhook was accepted and normalized.
type EventResult struct {
	Accepted  bool             `json:"accepted"`
	Customers []CustomerRecord `json:"customers,omitempty"`
	Events    []CustomerEvent  `json:"events,omitempty"`
}

// CustomerEvent is a normalized customer timeline event emitted by connectors.
type CustomerEvent struct {
	ExternalCustomerID string         `json:"external_customer_id"`
	Type               string         `json:"type"`
	OccurredAt         *time.Time     `json:"occurred_at,omitempty"`
	Data               map[string]any `json:"data,omitempty"`
}

// RegisteredConnector is the validated connector stored by a Registry.
type RegisteredConnector struct {
	Manifest  ConnectorManifest
	Connector Connector
}

// Registry stores connector registrations for a process.
type Registry struct {
	mu         sync.RWMutex
	connectors map[string]RegisteredConnector
}

// NewRegistry creates an empty connector registry.
func NewRegistry() *Registry {
	return &Registry{connectors: make(map[string]RegisteredConnector)}
}

// Register validates and stores a connector by manifest ID.
func (r *Registry) Register(connector Connector) (*RegisteredConnector, error) {
	if connector == nil {
		return nil, errors.New("connector is nil")
	}

	manifest := connector.Manifest()
	if err := ValidateManifest(manifest); err != nil {
		return nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.connectors[manifest.ID]; exists {
		return nil, fmt.Errorf("connector %q already registered", manifest.ID)
	}

	registered := RegisteredConnector{Manifest: manifest, Connector: connector}
	r.connectors[manifest.ID] = registered
	return &registered, nil
}

// Get returns a registered connector by manifest ID.
func (r *Registry) Get(id string) (*RegisteredConnector, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	registered, ok := r.connectors[id]
	if !ok {
		return nil, false
	}
	return &registered, true
}

// List returns registered connectors ordered by manifest ID.
func (r *Registry) List() []RegisteredConnector {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.connectors))
	for id := range r.connectors {
		ids = append(ids, id)
	}
	slices.Sort(ids)

	result := make([]RegisteredConnector, 0, len(ids))
	for _, id := range ids {
		result = append(result, r.connectors[id])
	}
	return result
}

// ValidateManifest verifies the connector manifest is complete and versioned.
func ValidateManifest(manifest ConnectorManifest) error {
	if err := validateConnectorMetadata(manifest); err != nil {
		return err
	}
	if err := validateAuth(manifest.Auth); err != nil {
		return err
	}
	if err := validateSync(manifest.Sync); err != nil {
		return err
	}
	return validateWebhooks(manifest.Webhooks)
}

func validateConnectorMetadata(manifest ConnectorManifest) error {
	if strings.TrimSpace(manifest.ID) == "" {
		return errors.New("manifest id is required")
	}
	if !connectorIDPatternRE.MatchString(manifest.ID) {
		return errors.New("manifest id must use lowercase letters, numbers, and single hyphens")
	}
	if strings.TrimSpace(manifest.Name) == "" {
		return errors.New("manifest name is required")
	}
	if !semverPattern.MatchString(manifest.Version) {
		return fmt.Errorf("manifest version %q must be semantic version", manifest.Version)
	}
	if strings.TrimSpace(manifest.Description) == "" {
		return errors.New("manifest description is required")
	}
	return nil
}

func validateWebhooks(webhooks []WebhookConfig) error {
	for _, webhook := range webhooks {
		if strings.TrimSpace(webhook.Path) == "" {
			return errors.New("webhook path is required")
		}
		if len(webhook.EventTypes) == 0 {
			return fmt.Errorf("webhook %q must declare event types", webhook.Path)
		}
		if err := validateWebhookEventTypes(webhook); err != nil {
			return err
		}
	}
	return nil
}

func validateWebhookEventTypes(webhook WebhookConfig) error {
	seen := make(map[string]struct{}, len(webhook.EventTypes))
	for _, eventType := range webhook.EventTypes {
		eventType = strings.TrimSpace(eventType)
		if eventType == "" {
			return fmt.Errorf("webhook %q event type is required", webhook.Path)
		}
		if _, exists := seen[eventType]; exists {
			return fmt.Errorf("duplicate webhook event type %q", eventType)
		}
		seen[eventType] = struct{}{}
	}
	return nil
}

func validateAuth(auth AuthConfig) error {
	switch auth.Type {
	case AuthTypeNone:
		return nil
	case AuthTypeAPIKey:
		if auth.APIKey == nil || strings.TrimSpace(auth.APIKey.HeaderName) == "" {
			return errors.New("api_key auth requires header_name")
		}
		return nil
	case AuthTypeOAuth2:
		if auth.OAuth2 == nil {
			return errors.New("oauth2 auth config is required")
		}
		if strings.TrimSpace(auth.OAuth2.AuthorizeURL) == "" || strings.TrimSpace(auth.OAuth2.TokenURL) == "" {
			return errors.New("oauth2 auth requires authorize_url and token_url")
		}
		return nil
	default:
		return fmt.Errorf("unsupported auth type %q", auth.Type)
	}
}

func validateSync(syncConfig SyncConfig) error {
	if err := validateSupportedSyncModes(syncConfig.SupportedModes); err != nil {
		return err
	}
	if syncConfig.DefaultMode == "" {
		return errors.New("sync default_mode is required")
	}
	if !slices.Contains(syncConfig.SupportedModes, syncConfig.DefaultMode) {
		return fmt.Errorf("sync default_mode %q must be in supported_modes", syncConfig.DefaultMode)
	}
	if err := validateSyncResources(syncConfig.Resources); err != nil {
		return err
	}
	if syncConfig.Schedule != nil && syncConfig.Schedule.IntervalMinutes <= 0 {
		return errors.New("sync schedule interval_minutes must be positive")
	}
	return nil
}

func validateSupportedSyncModes(modes []SyncMode) error {
	if len(modes) == 0 {
		return errors.New("sync supported_modes is required")
	}
	seen := make(map[SyncMode]struct{}, len(modes))
	for _, mode := range modes {
		if !isSupportedSyncMode(mode) {
			return fmt.Errorf("unsupported sync mode %q", mode)
		}
		if _, exists := seen[mode]; exists {
			return fmt.Errorf("duplicate sync mode %q", mode)
		}
		seen[mode] = struct{}{}
	}
	return nil
}

func validateSyncResources(resources []ResourceConfig) error {
	if len(resources) == 0 {
		return errors.New("sync resources is required")
	}
	seen := make(map[string]struct{}, len(resources))
	for _, resource := range resources {
		name := strings.TrimSpace(resource.Name)
		if name == "" {
			return errors.New("sync resource name is required")
		}
		if _, exists := seen[name]; exists {
			return fmt.Errorf("duplicate sync resource %q", name)
		}
		seen[name] = struct{}{}
	}
	return nil
}

func isSupportedSyncMode(mode SyncMode) bool {
	return mode == SyncModeFull || mode == SyncModeIncremental
}
