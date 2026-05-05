package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/onnwee/pulse-score/internal/integration"
)

var _ integration.Connector = (*builtInIntegrationConnector)(nil)

type builtInIntegrationConnector struct {
	name          string
	authType      integration.AuthType
	connect       func(context.Context, integration.ConnectConfig) error
	disconnect    func(context.Context, integration.DisconnectConfig) error
	sync          func(context.Context, integration.SyncOptions) (integration.SyncResult, error)
	handleWebhook func(context.Context, integration.WebhookPayload) error
	getStatus     func(context.Context, integration.StatusOptions) integration.Status
}

func (c *builtInIntegrationConnector) Name() string { return c.name }

func (c *builtInIntegrationConnector) AuthType() integration.AuthType { return c.authType }

func (c *builtInIntegrationConnector) Connect(ctx context.Context, config integration.ConnectConfig) error {
	return c.connect(ctx, config)
}

func (c *builtInIntegrationConnector) Disconnect(ctx context.Context, config integration.DisconnectConfig) error {
	return c.disconnect(ctx, config)
}

func (c *builtInIntegrationConnector) Sync(ctx context.Context, opts integration.SyncOptions) (integration.SyncResult, error) {
	return c.sync(ctx, opts)
}

func (c *builtInIntegrationConnector) HandleWebhook(ctx context.Context, payload integration.WebhookPayload) error {
	return c.handleWebhook(ctx, payload)
}

func (c *builtInIntegrationConnector) GetStatus(ctx context.Context, opts integration.StatusOptions) integration.Status {
	return c.getStatus(ctx, opts)
}

// NewBuiltInIntegrationRegistry registers original provider connectors against the issue #195 lifecycle interface.
func NewBuiltInIntegrationRegistry(
	stripeOAuth *StripeOAuthService,
	stripeOrchestrator *SyncOrchestratorService,
	hubspotOAuth *HubSpotOAuthService,
	hubspotOrchestrator *HubSpotSyncOrchestratorService,
	intercomOAuth *IntercomOAuthService,
	intercomOrchestrator *IntercomSyncOrchestratorService,
) (*integration.Registry, error) {
	registry := integration.NewRegistry()
	connectors := []integration.Connector{
		NewStripeIntegrationConnector(stripeOAuth, stripeOrchestrator),
		NewHubSpotIntegrationConnector(hubspotOAuth, hubspotOrchestrator),
		NewIntercomIntegrationConnector(intercomOAuth, intercomOrchestrator),
	}

	for _, connector := range connectors {
		name := connector.Name()
		if err := registry.Register(name, func() integration.Connector { return connector }); err != nil {
			return nil, fmt.Errorf("register integration connector %q: %w", name, err)
		}
	}

	return registry, nil
}

// NewStripeIntegrationConnector adapts Stripe services to the issue #195 lifecycle interface.
func NewStripeIntegrationConnector(oauth *StripeOAuthService, orchestrator *SyncOrchestratorService) integration.Connector {
	return &builtInIntegrationConnector{
		name:     providerStripe,
		authType: integration.AuthTypeOAuth2,
		connect: func(ctx context.Context, config integration.ConnectConfig) error {
			if oauth == nil {
				return &ValidationError{Field: providerStripe, Message: "Stripe OAuth service is not configured"}
			}
			orgID, err := parseConnectorOrgID(config.OrgID)
			if err != nil {
				return err
			}
			return oauth.ExchangeCode(ctx, orgID, config.Code, config.Metadata["state"])
		},
		disconnect: func(ctx context.Context, config integration.DisconnectConfig) error {
			if oauth == nil {
				return &ValidationError{Field: providerStripe, Message: "Stripe OAuth service is not configured"}
			}
			orgID, err := parseConnectorOrgID(config.OrgID)
			if err != nil {
				return err
			}
			return oauth.Disconnect(ctx, orgID)
		},
		sync: func(ctx context.Context, opts integration.SyncOptions) (integration.SyncResult, error) {
			if orchestrator == nil {
				return integration.SyncResult{}, &ValidationError{Field: providerStripe, Message: "Stripe sync orchestrator is not configured"}
			}
			orgID, err := parseConnectorOrgID(opts.OrgID)
			if err != nil {
				return integration.SyncResult{}, err
			}
			var result *SyncResult
			switch opts.Mode {
			case integration.SyncModeFull:
				result = orchestrator.RunFullSync(ctx, orgID)
			case integration.SyncModeIncremental:
				if opts.Since == nil {
					return integration.SyncResult{}, &ValidationError{Field: "since", Message: "incremental sync requires since"}
				}
				result = orchestrator.RunIncrementalSync(ctx, orgID, *opts.Since)
			default:
				return integration.SyncResult{}, &ValidationError{Field: "mode", Message: fmt.Sprintf("unsupported sync mode %q", opts.Mode)}
			}
			return integration.SyncResult{Resources: []integration.ResourceResult{
				integrationResourceResult("customers", result.Customers, result.Error),
				integrationResourceResult("subscriptions", result.Subscriptions, result.Error),
				integrationResourceResult("payments", result.Payments, result.Error),
				{Name: "mrr", Synced: result.MRRUpdated, Error: result.Error},
			}}, nil
		},
		handleWebhook: noOpIntegrationWebhook,
		getStatus: func(ctx context.Context, opts integration.StatusOptions) integration.Status {
			if oauth == nil {
				return integrationErrorStatus("Stripe OAuth service is not configured")
			}
			orgID, err := parseConnectorOrgID(opts.OrgID)
			if err != nil {
				return integrationErrorStatus(err.Error())
			}
			status, err := oauth.GetStatus(ctx, orgID)
			if err != nil {
				return integrationErrorStatus(err.Error())
			}
			return integrationStatus(status.Status, status.ExternalAccountID, status.LastSyncAt, status.LastSyncError, status.ConnectedAt)
		},
	}
}

// NewHubSpotIntegrationConnector adapts HubSpot services to the issue #195 lifecycle interface.
func NewHubSpotIntegrationConnector(oauth *HubSpotOAuthService, orchestrator *HubSpotSyncOrchestratorService) integration.Connector {
	return &builtInIntegrationConnector{
		name:     providerHubSpot,
		authType: integration.AuthTypeOAuth2,
		connect: func(ctx context.Context, config integration.ConnectConfig) error {
			if oauth == nil {
				return &ValidationError{Field: providerHubSpot, Message: "HubSpot OAuth service is not configured"}
			}
			orgID, err := parseConnectorOrgID(config.OrgID)
			if err != nil {
				return err
			}
			return oauth.ExchangeCode(ctx, orgID, config.Code, config.Metadata["state"])
		},
		disconnect: func(ctx context.Context, config integration.DisconnectConfig) error {
			if oauth == nil {
				return &ValidationError{Field: providerHubSpot, Message: "HubSpot OAuth service is not configured"}
			}
			orgID, err := parseConnectorOrgID(config.OrgID)
			if err != nil {
				return err
			}
			return oauth.Disconnect(ctx, orgID)
		},
		sync: func(ctx context.Context, opts integration.SyncOptions) (integration.SyncResult, error) {
			if orchestrator == nil {
				return integration.SyncResult{}, &ValidationError{Field: providerHubSpot, Message: "HubSpot sync orchestrator is not configured"}
			}
			orgID, err := parseConnectorOrgID(opts.OrgID)
			if err != nil {
				return integration.SyncResult{}, err
			}
			var result *HubSpotSyncResult
			switch opts.Mode {
			case integration.SyncModeFull:
				result = orchestrator.RunFullSync(ctx, orgID)
			case integration.SyncModeIncremental:
				if opts.Since == nil {
					return integration.SyncResult{}, &ValidationError{Field: "since", Message: "incremental sync requires since"}
				}
				result = orchestrator.RunIncrementalSync(ctx, orgID, *opts.Since)
			default:
				return integration.SyncResult{}, &ValidationError{Field: "mode", Message: fmt.Sprintf("unsupported sync mode %q", opts.Mode)}
			}
			errText := strings.Join(result.Errors, "; ")
			return integration.SyncResult{Resources: []integration.ResourceResult{
				integrationResourceResult("contacts", result.Contacts, errText),
				integrationResourceResult("deals", result.Deals, errText),
				integrationResourceResult("companies", result.Companies, errText),
			}}, nil
		},
		handleWebhook: noOpIntegrationWebhook,
		getStatus: func(ctx context.Context, opts integration.StatusOptions) integration.Status {
			if oauth == nil {
				return integrationErrorStatus("HubSpot OAuth service is not configured")
			}
			orgID, err := parseConnectorOrgID(opts.OrgID)
			if err != nil {
				return integrationErrorStatus(err.Error())
			}
			status, err := oauth.GetStatus(ctx, orgID)
			if err != nil {
				return integrationErrorStatus(err.Error())
			}
			return integrationStatus(status.Status, status.ExternalAccountID, status.LastSyncAt, status.LastSyncError, status.ConnectedAt)
		},
	}
}

// NewIntercomIntegrationConnector adapts Intercom services to the issue #195 lifecycle interface.
func NewIntercomIntegrationConnector(oauth *IntercomOAuthService, orchestrator *IntercomSyncOrchestratorService) integration.Connector {
	return &builtInIntegrationConnector{
		name:     providerIntercom,
		authType: integration.AuthTypeOAuth2,
		connect: func(ctx context.Context, config integration.ConnectConfig) error {
			if oauth == nil {
				return &ValidationError{Field: providerIntercom, Message: "Intercom OAuth service is not configured"}
			}
			orgID, err := parseConnectorOrgID(config.OrgID)
			if err != nil {
				return err
			}
			return oauth.ExchangeCode(ctx, orgID, config.Code, config.Metadata["state"])
		},
		disconnect: func(ctx context.Context, config integration.DisconnectConfig) error {
			if oauth == nil {
				return &ValidationError{Field: providerIntercom, Message: "Intercom OAuth service is not configured"}
			}
			orgID, err := parseConnectorOrgID(config.OrgID)
			if err != nil {
				return err
			}
			return oauth.Disconnect(ctx, orgID)
		},
		sync: func(ctx context.Context, opts integration.SyncOptions) (integration.SyncResult, error) {
			if orchestrator == nil {
				return integration.SyncResult{}, &ValidationError{Field: providerIntercom, Message: "Intercom sync orchestrator is not configured"}
			}
			orgID, err := parseConnectorOrgID(opts.OrgID)
			if err != nil {
				return integration.SyncResult{}, err
			}
			var result *IntercomSyncResult
			switch opts.Mode {
			case integration.SyncModeFull:
				result = orchestrator.RunFullSync(ctx, orgID)
			case integration.SyncModeIncremental:
				if opts.Since == nil {
					return integration.SyncResult{}, &ValidationError{Field: "since", Message: "incremental sync requires since"}
				}
				result = orchestrator.RunIncrementalSync(ctx, orgID, *opts.Since)
			default:
				return integration.SyncResult{}, &ValidationError{Field: "mode", Message: fmt.Sprintf("unsupported sync mode %q", opts.Mode)}
			}
			errText := strings.Join(result.Errors, "; ")
			return integration.SyncResult{Resources: []integration.ResourceResult{
				integrationResourceResult("contacts", result.Contacts, errText),
				integrationResourceResult("conversations", result.Conversations, errText),
			}}, nil
		},
		handleWebhook: noOpIntegrationWebhook,
		getStatus: func(ctx context.Context, opts integration.StatusOptions) integration.Status {
			if oauth == nil {
				return integrationErrorStatus("Intercom OAuth service is not configured")
			}
			orgID, err := parseConnectorOrgID(opts.OrgID)
			if err != nil {
				return integrationErrorStatus(err.Error())
			}
			status, err := oauth.GetStatus(ctx, orgID)
			if err != nil {
				return integrationErrorStatus(err.Error())
			}
			return integrationStatus(status.Status, status.ExternalAccountID, status.LastSyncAt, status.LastSyncError, status.ConnectedAt)
		},
	}
}

func noOpIntegrationWebhook(context.Context, integration.WebhookPayload) error {
	return nil
}

func integrationResourceResult(name string, progress *SyncProgress, errText string) integration.ResourceResult {
	result := integration.ResourceResult{Name: name, Error: errText}
	if progress == nil {
		return result
	}
	result.Synced = progress.Current
	result.Skipped = progress.Errors
	return result
}

func integrationStatus(status, externalAccountID string, lastSyncAt *time.Time, lastError string, connectedAt time.Time) integration.Status {
	state := integrationStatusState(status, lastError)
	return integration.Status{
		State:             state,
		ExternalAccountID: externalAccountID,
		LastSyncAt:        lastSyncAt,
		LastError:         lastError,
		ConnectedAt:       optionalTime(connectedAt),
	}
}

func integrationStatusState(status, lastError string) integration.StatusState {
	if lastError != "" {
		return integration.StatusError
	}
	switch status {
	case "active", "syncing":
		return integration.StatusConnected
	case "disconnected", "":
		return integration.StatusDisconnected
	default:
		return integration.StatusError
	}
}

func integrationErrorStatus(errText string) integration.Status {
	return integration.Status{State: integration.StatusError, LastError: errText}
}

func optionalTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	return &value
}
