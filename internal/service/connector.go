package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	connectorsdk "github.com/onnwee/pulse-score/pkg/connector-sdk"
)

const (
	providerStripe   = "stripe"
	providerHubSpot  = "hubspot"
	providerIntercom = "intercom"
	providerSalesforce = "salesforce"
)

var (
	_ connectorsdk.Connector = (*StripeConnector)(nil)
	_ connectorsdk.Connector = (*HubSpotConnector)(nil)
	_ connectorsdk.Connector = (*IntercomConnector)(nil)
	_ connectorsdk.Connector = (*SalesforceConnector)(nil)
)

// ConnectorSyncer runs a provider sync through the connector registry.
type ConnectorSyncer interface {
	Sync(ctx context.Context, provider string, orgID uuid.UUID, mode connectorsdk.SyncMode, since *time.Time) (*connectorsdk.SyncResult, error)
}

// ConnectorSyncService resolves connector syncs from a registry.
type ConnectorSyncService struct {
	registry *connectorsdk.Registry
}

// NewConnectorSyncService creates a registry-backed sync service.
func NewConnectorSyncService(registry *connectorsdk.Registry) *ConnectorSyncService {
	return &ConnectorSyncService{registry: registry}
}

// Sync looks up a provider connector and runs the requested sync mode.
func (s *ConnectorSyncService) Sync(ctx context.Context, provider string, orgID uuid.UUID, mode connectorsdk.SyncMode, since *time.Time) (*connectorsdk.SyncResult, error) {
	if s == nil || s.registry == nil {
		return nil, &ValidationError{Field: "connector_registry", Message: "connector registry is not configured"}
	}

	registered, ok := s.registry.Get(provider)
	if !ok {
		return nil, &NotFoundError{Resource: "connector", Message: fmt.Sprintf("no %s connector registered", provider)}
	}

	return registered.Connector.Sync(ctx, connectorsdk.SyncRequest{
		OrgID: orgID.String(),
		Mode:  mode,
		Since: since,
	})
}

// NewIntegrationConnectorRegistry registers built-in provider connectors.
func NewIntegrationConnectorRegistry(
	stripeOAuth *StripeOAuthService,
	stripeOrchestrator *SyncOrchestratorService,
	hubspotOAuth *HubSpotOAuthService,
	hubspotOrchestrator *HubSpotSyncOrchestratorService,
	intercomOAuth *IntercomOAuthService,
	intercomOrchestrator *IntercomSyncOrchestratorService,
	salesforceOAuth *SalesforceOAuthService,
	salesforceOrchestrator *SalesforceSyncOrchestratorService,
) (*connectorsdk.Registry, error) {
	registry := connectorsdk.NewRegistry()
	connectors := []connectorsdk.Connector{
		NewStripeConnector(stripeOAuth, stripeOrchestrator),
		NewHubSpotConnector(hubspotOAuth, hubspotOrchestrator),
		NewIntercomConnector(intercomOAuth, intercomOrchestrator),
		NewSalesforceConnector(salesforceOAuth, salesforceOrchestrator),
	}

	for _, connector := range connectors {
		if _, err := registry.Register(connector); err != nil {
			return nil, fmt.Errorf("register connector %q: %w", connector.Manifest().ID, err)
		}
	}

	return registry, nil
}

// StripeConnector adapts the existing Stripe services to the connector SDK.
type StripeConnector struct {
	oauth        *StripeOAuthService
	orchestrator *SyncOrchestratorService
}

// NewStripeConnector creates a Stripe connector adapter.
func NewStripeConnector(oauth *StripeOAuthService, orchestrator *SyncOrchestratorService) *StripeConnector {
	return &StripeConnector{oauth: oauth, orchestrator: orchestrator}
}

// Manifest returns Stripe connector metadata.
func (c *StripeConnector) Manifest() connectorsdk.ConnectorManifest {
	return connectorsdk.ConnectorManifest{
		ID:          providerStripe,
		Name:        "Stripe",
		Version:     "1.0.0",
		Description: "Syncs Stripe customers, subscriptions, payments, and MRR signals.",
		Categories:  []string{"billing", "payments"},
		Auth: connectorsdk.AuthConfig{
			Type: connectorsdk.AuthTypeOAuth2,
			OAuth2: &connectorsdk.OAuth2Config{
				AuthorizeURL: "https://connect.stripe.com/oauth/authorize",
				TokenURL:     "https://connect.stripe.com/oauth/token",
				Scopes:       []string{"read_only"},
			},
		},
		Sync: connectorsdk.SyncConfig{
			SupportedModes: []connectorsdk.SyncMode{connectorsdk.SyncModeFull, connectorsdk.SyncModeIncremental},
			DefaultMode:    connectorsdk.SyncModeFull,
			Resources: []connectorsdk.ResourceConfig{
				{Name: "customers", Description: "Stripe customer records", Required: true},
				{Name: "subscriptions", Description: "Stripe subscription records", Required: true},
				{Name: "payments", Description: "Stripe charge and payment records", Required: true},
				{Name: "mrr", Description: "MRR recalculation after Stripe sync", Required: true},
			},
		},
		Webhooks: []connectorsdk.WebhookConfig{
			{
				Path: "/api/v1/webhooks/stripe",
				EventTypes: []string{
					"customer.created",
					"customer.updated",
					"customer.subscription.created",
					"customer.subscription.updated",
					"customer.subscription.deleted",
					"charge.succeeded",
					"charge.failed",
					"invoice.payment_succeeded",
					"invoice.payment_failed",
				},
				SigningSecretHeader: "Stripe-Signature",
			},
		},
	}
}

// Authenticate exchanges Stripe OAuth credentials using the existing service.
func (c *StripeConnector) Authenticate(ctx context.Context, req connectorsdk.AuthRequest) (*connectorsdk.AuthResult, error) {
	if c.oauth == nil {
		return nil, &ValidationError{Field: providerStripe, Message: "Stripe OAuth service is not configured"}
	}
	orgID, err := parseConnectorOrgID(req.OrgID)
	if err != nil {
		return nil, err
	}
	state := req.Metadata["state"]
	if err := c.oauth.ExchangeCode(ctx, orgID, req.Code, state); err != nil {
		return nil, err
	}
	status, err := c.oauth.GetStatus(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return &connectorsdk.AuthResult{ExternalAccountID: status.ExternalAccountID, Scopes: []string{"read_only"}}, nil
}

// Sync runs a Stripe full or incremental sync.
func (c *StripeConnector) Sync(ctx context.Context, req connectorsdk.SyncRequest) (*connectorsdk.SyncResult, error) {
	if c.orchestrator == nil {
		return nil, &ValidationError{Field: providerStripe, Message: "Stripe sync orchestrator is not configured"}
	}
	orgID, err := parseConnectorOrgID(req.OrgID)
	if err != nil {
		return nil, err
	}

	var result *SyncResult
	switch req.Mode {
	case connectorsdk.SyncModeFull:
		result = c.orchestrator.RunFullSync(ctx, orgID)
	case connectorsdk.SyncModeIncremental:
		if req.Since == nil {
			return nil, &ValidationError{Field: "since", Message: "incremental sync requires since"}
		}
		result = c.orchestrator.RunIncrementalSync(ctx, orgID, *req.Since)
	default:
		return nil, &ValidationError{Field: "mode", Message: fmt.Sprintf("unsupported sync mode %q", req.Mode)}
	}

	resources := []connectorsdk.ResourceResult{
		resourceResult("customers", result.Customers, result.Error),
		resourceResult("subscriptions", result.Subscriptions, result.Error),
		resourceResult("payments", result.Payments, result.Error),
		{Name: "mrr", Synced: result.MRRUpdated, Error: result.Error},
	}
	return &connectorsdk.SyncResult{Resources: resources}, nil
}

// HandleEvent leaves existing Stripe webhook handling unchanged.
func (c *StripeConnector) HandleEvent(context.Context, connectorsdk.EventRequest) (*connectorsdk.EventResult, error) {
	return &connectorsdk.EventResult{Accepted: false}, nil
}

// HubSpotConnector adapts existing HubSpot services to the connector SDK.
type HubSpotConnector struct {
	oauth        *HubSpotOAuthService
	orchestrator *HubSpotSyncOrchestratorService
}

// NewHubSpotConnector creates a HubSpot connector adapter.
func NewHubSpotConnector(oauth *HubSpotOAuthService, orchestrator *HubSpotSyncOrchestratorService) *HubSpotConnector {
	return &HubSpotConnector{oauth: oauth, orchestrator: orchestrator}
}

// Manifest returns HubSpot connector metadata.
func (c *HubSpotConnector) Manifest() connectorsdk.ConnectorManifest {
	return connectorsdk.ConnectorManifest{
		ID:          providerHubSpot,
		Name:        "HubSpot",
		Version:     "1.0.0",
		Description: "Syncs HubSpot contacts, deals, and companies.",
		Categories:  []string{"crm"},
		Auth: connectorsdk.AuthConfig{
			Type: connectorsdk.AuthTypeOAuth2,
			OAuth2: &connectorsdk.OAuth2Config{
				AuthorizeURL: "https://app.hubspot.com/oauth/authorize",
				TokenURL:     "https://api.hubapi.com/oauth/v1/token",
				Scopes:       []string{"crm.objects.contacts.read", "crm.objects.deals.read", "crm.objects.companies.read"},
			},
		},
		Sync: connectorsdk.SyncConfig{
			SupportedModes: []connectorsdk.SyncMode{connectorsdk.SyncModeFull, connectorsdk.SyncModeIncremental},
			DefaultMode:    connectorsdk.SyncModeFull,
			Resources: []connectorsdk.ResourceConfig{
				{Name: "contacts", Description: "HubSpot contact records", Required: true},
				{Name: "deals", Description: "HubSpot deal records", Required: true},
				{Name: "companies", Description: "HubSpot company records", Required: false},
			},
		},
		Webhooks: []connectorsdk.WebhookConfig{
			{
				Path: "/api/v1/webhooks/hubspot",
				EventTypes: []string{
					"contact.creation",
					"contact.propertyChange",
					"contact.deletion",
					"deal.creation",
					"deal.propertyChange",
					"deal.deletion",
					"company.creation",
					"company.propertyChange",
				},
				SigningSecretHeader: "X-HubSpot-Signature-v3",
			},
		},
	}
}

// Authenticate exchanges HubSpot OAuth credentials using the existing service.
func (c *HubSpotConnector) Authenticate(ctx context.Context, req connectorsdk.AuthRequest) (*connectorsdk.AuthResult, error) {
	if c.oauth == nil {
		return nil, &ValidationError{Field: providerHubSpot, Message: "HubSpot OAuth service is not configured"}
	}
	orgID, err := parseConnectorOrgID(req.OrgID)
	if err != nil {
		return nil, err
	}
	state := req.Metadata["state"]
	if err := c.oauth.ExchangeCode(ctx, orgID, req.Code, state); err != nil {
		return nil, err
	}
	status, err := c.oauth.GetStatus(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return &connectorsdk.AuthResult{ExternalAccountID: status.ExternalAccountID, Scopes: []string{"crm.objects.contacts.read", "crm.objects.deals.read", "crm.objects.companies.read"}}, nil
}

// Sync runs a HubSpot full or incremental sync.
func (c *HubSpotConnector) Sync(ctx context.Context, req connectorsdk.SyncRequest) (*connectorsdk.SyncResult, error) {
	if c.orchestrator == nil {
		return nil, &ValidationError{Field: providerHubSpot, Message: "HubSpot sync orchestrator is not configured"}
	}
	orgID, err := parseConnectorOrgID(req.OrgID)
	if err != nil {
		return nil, err
	}

	var result *HubSpotSyncResult
	switch req.Mode {
	case connectorsdk.SyncModeFull:
		result = c.orchestrator.RunFullSync(ctx, orgID)
	case connectorsdk.SyncModeIncremental:
		if req.Since == nil {
			return nil, &ValidationError{Field: "since", Message: "incremental sync requires since"}
		}
		result = c.orchestrator.RunIncrementalSync(ctx, orgID, *req.Since)
	default:
		return nil, &ValidationError{Field: "mode", Message: fmt.Sprintf("unsupported sync mode %q", req.Mode)}
	}

	errorText := strings.Join(result.Errors, "; ")
	resources := []connectorsdk.ResourceResult{
		resourceResult("contacts", result.Contacts, errorText),
		resourceResult("deals", result.Deals, errorText),
		resourceResult("companies", result.Companies, errorText),
	}
	return &connectorsdk.SyncResult{Resources: resources}, nil
}

// HandleEvent leaves existing HubSpot webhook handling unchanged.
func (c *HubSpotConnector) HandleEvent(context.Context, connectorsdk.EventRequest) (*connectorsdk.EventResult, error) {
	return &connectorsdk.EventResult{Accepted: false}, nil
}

// IntercomConnector adapts existing Intercom services to the connector SDK.
type IntercomConnector struct {
	oauth        *IntercomOAuthService
	orchestrator *IntercomSyncOrchestratorService
}

// NewIntercomConnector creates an Intercom connector adapter.
func NewIntercomConnector(oauth *IntercomOAuthService, orchestrator *IntercomSyncOrchestratorService) *IntercomConnector {
	return &IntercomConnector{oauth: oauth, orchestrator: orchestrator}
}

// Manifest returns Intercom connector metadata.
func (c *IntercomConnector) Manifest() connectorsdk.ConnectorManifest {
	return connectorsdk.ConnectorManifest{
		ID:          providerIntercom,
		Name:        "Intercom",
		Version:     "1.0.0",
		Description: "Syncs Intercom contacts and conversations.",
		Categories:  []string{"support"},
		Auth: connectorsdk.AuthConfig{
			Type: connectorsdk.AuthTypeOAuth2,
			OAuth2: &connectorsdk.OAuth2Config{
				AuthorizeURL: "https://app.intercom.com/oauth",
				TokenURL:     "https://api.intercom.io/auth/eagle/token",
				Scopes:       []string{"read_contacts", "read_conversations"},
			},
		},
		Sync: connectorsdk.SyncConfig{
			SupportedModes: []connectorsdk.SyncMode{connectorsdk.SyncModeFull, connectorsdk.SyncModeIncremental},
			DefaultMode:    connectorsdk.SyncModeFull,
			Resources: []connectorsdk.ResourceConfig{
				{Name: "contacts", Description: "Intercom contact records", Required: true},
				{Name: "conversations", Description: "Intercom conversation records", Required: true},
			},
		},
		Webhooks: []connectorsdk.WebhookConfig{
			{
				Path: "/api/v1/webhooks/intercom",
				EventTypes: []string{
					"contact.created",
					"contact.updated",
					"contact.deleted",
					"conversation.created",
					"conversation.updated",
					"conversation.closed",
					"conversation.reopened",
					"conversation.assigned",
				},
				SigningSecretHeader: "X-Hub-Signature",
			},
		},
	}
}

// Authenticate exchanges Intercom OAuth credentials using the existing service.
func (c *IntercomConnector) Authenticate(ctx context.Context, req connectorsdk.AuthRequest) (*connectorsdk.AuthResult, error) {
	if c.oauth == nil {
		return nil, &ValidationError{Field: providerIntercom, Message: "Intercom OAuth service is not configured"}
	}
	orgID, err := parseConnectorOrgID(req.OrgID)
	if err != nil {
		return nil, err
	}
	state := req.Metadata["state"]
	if err := c.oauth.ExchangeCode(ctx, orgID, req.Code, state); err != nil {
		return nil, err
	}
	status, err := c.oauth.GetStatus(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return &connectorsdk.AuthResult{ExternalAccountID: status.ExternalAccountID, Scopes: []string{"read_contacts", "read_conversations"}}, nil
}

// Sync runs an Intercom full or incremental sync.
func (c *IntercomConnector) Sync(ctx context.Context, req connectorsdk.SyncRequest) (*connectorsdk.SyncResult, error) {
	if c.orchestrator == nil {
		return nil, &ValidationError{Field: providerIntercom, Message: "Intercom sync orchestrator is not configured"}
	}
	orgID, err := parseConnectorOrgID(req.OrgID)
	if err != nil {
		return nil, err
	}

	var result *IntercomSyncResult
	switch req.Mode {
	case connectorsdk.SyncModeFull:
		result = c.orchestrator.RunFullSync(ctx, orgID)
	case connectorsdk.SyncModeIncremental:
		if req.Since == nil {
			return nil, &ValidationError{Field: "since", Message: "incremental sync requires since"}
		}
		result = c.orchestrator.RunIncrementalSync(ctx, orgID, *req.Since)
	default:
		return nil, &ValidationError{Field: "mode", Message: fmt.Sprintf("unsupported sync mode %q", req.Mode)}
	}

	errorText := strings.Join(result.Errors, "; ")
	resources := []connectorsdk.ResourceResult{
		resourceResult("contacts", result.Contacts, errorText),
		resourceResult("conversations", result.Conversations, errorText),
	}
	return &connectorsdk.SyncResult{Resources: resources}, nil
}

// HandleEvent leaves existing Intercom webhook handling unchanged.
func (c *IntercomConnector) HandleEvent(context.Context, connectorsdk.EventRequest) (*connectorsdk.EventResult, error) {
	return &connectorsdk.EventResult{Accepted: false}, nil
}

// SalesforceConnector adapts Salesforce services to the connector SDK.
type SalesforceConnector struct {
	oauth        *SalesforceOAuthService
	orchestrator *SalesforceSyncOrchestratorService
}

// NewSalesforceConnector creates a Salesforce connector adapter.
func NewSalesforceConnector(oauth *SalesforceOAuthService, orchestrator *SalesforceSyncOrchestratorService) *SalesforceConnector {
	return &SalesforceConnector{oauth: oauth, orchestrator: orchestrator}
}

// Manifest returns Salesforce connector metadata.
func (c *SalesforceConnector) Manifest() connectorsdk.ConnectorManifest {
	return connectorsdk.ConnectorManifest{
		ID:          providerSalesforce,
		Name:        "Salesforce",
		Version:     "1.0.0",
		Description: "Syncs Salesforce accounts, contacts, and opportunities.",
		Categories:  []string{"crm"},
		Auth: connectorsdk.AuthConfig{
			Type: connectorsdk.AuthTypeOAuth2,
			OAuth2: &connectorsdk.OAuth2Config{
				AuthorizeURL: "https://login.salesforce.com/services/oauth2/authorize",
				TokenURL:     "https://login.salesforce.com/services/oauth2/token",
				Scopes:       []string{"api", "refresh_token"},
			},
		},
		Sync: connectorsdk.SyncConfig{
			SupportedModes: []connectorsdk.SyncMode{connectorsdk.SyncModeFull, connectorsdk.SyncModeIncremental},
			DefaultMode:    connectorsdk.SyncModeFull,
			Resources: []connectorsdk.ResourceConfig{
				{Name: "accounts", Description: "Salesforce account records", Required: true},
				{Name: "contacts", Description: "Salesforce contact records mapped to customers by email", Required: true},
				{Name: "opportunities", Description: "Salesforce opportunities mapped to customer events", Required: true},
			},
		},
	}
}

// Authenticate exchanges Salesforce OAuth credentials using the OAuth service.
func (c *SalesforceConnector) Authenticate(ctx context.Context, req connectorsdk.AuthRequest) (*connectorsdk.AuthResult, error) {
	if c.oauth == nil {
		return nil, &ValidationError{Field: providerSalesforce, Message: "Salesforce OAuth service is not configured"}
	}
	orgID, err := parseConnectorOrgID(req.OrgID)
	if err != nil {
		return nil, err
	}
	state := req.Metadata["state"]
	if err := c.oauth.ExchangeCode(ctx, orgID, req.Code, state); err != nil {
		return nil, err
	}
	status, err := c.oauth.GetStatus(ctx, orgID)
	if err != nil {
		return nil, err
	}
	return &connectorsdk.AuthResult{ExternalAccountID: status.ExternalAccountID, Scopes: []string{"api", "refresh_token"}}, nil
}

// Sync runs a Salesforce full or incremental sync.
func (c *SalesforceConnector) Sync(ctx context.Context, req connectorsdk.SyncRequest) (*connectorsdk.SyncResult, error) {
	if c.orchestrator == nil {
		return nil, &ValidationError{Field: providerSalesforce, Message: "Salesforce sync orchestrator is not configured"}
	}
	orgID, err := parseConnectorOrgID(req.OrgID)
	if err != nil {
		return nil, err
	}

	var result *SalesforceSyncResult
	switch req.Mode {
	case connectorsdk.SyncModeFull:
		result = c.orchestrator.RunFullSync(ctx, orgID)
	case connectorsdk.SyncModeIncremental:
		if req.Since == nil {
			return nil, &ValidationError{Field: "since", Message: "incremental sync requires since"}
		}
		result = c.orchestrator.RunIncrementalSync(ctx, orgID, *req.Since)
	default:
		return nil, &ValidationError{Field: "mode", Message: fmt.Sprintf("unsupported sync mode %q", req.Mode)}
	}

	errorText := strings.Join(result.Errors, "; ")
	resources := []connectorsdk.ResourceResult{
		resourceResult("accounts", result.Accounts, errorText),
		resourceResult("contacts", result.Contacts, errorText),
		resourceResult("opportunities", result.Opportunities, errorText),
	}
	return &connectorsdk.SyncResult{Resources: resources}, nil
}

// HandleEvent leaves Salesforce webhook handling for a later task.
func (c *SalesforceConnector) HandleEvent(context.Context, connectorsdk.EventRequest) (*connectorsdk.EventResult, error) {
	return &connectorsdk.EventResult{Accepted: false}, nil
}

func parseConnectorOrgID(raw string) (uuid.UUID, error) {
	orgID, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, &ValidationError{Field: "org_id", Message: "valid org id is required"}
	}
	return orgID, nil
}

func resourceResult(name string, progress *SyncProgress, errText string) connectorsdk.ResourceResult {
	result := connectorsdk.ResourceResult{Name: name, Error: errText}
	if progress == nil {
		return result
	}
	result.Synced = progress.Current
	result.Skipped = progress.Errors
	return result
}
