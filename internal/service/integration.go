package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	connectorsdk "github.com/onnwee/pulse-score/pkg/connector-sdk"

	"github.com/onnwee/pulse-score/internal/repository"
)

// IntegrationService handles integration management business logic.
type IntegrationService struct {
	connStore integrationConnectionStore
	syncer    ConnectorSyncer
	registry  *connectorsdk.Registry
}

type integrationConnectionStore interface {
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*repository.IntegrationConnection, error)
	GetByOrgAndProvider(ctx context.Context, orgID uuid.UUID, provider string) (*repository.IntegrationConnection, error)
	GetCustomerCountBySource(ctx context.Context, orgID uuid.UUID, provider string) (int, error)
	Delete(ctx context.Context, orgID uuid.UUID, provider string) error
}

// NewIntegrationService creates a new IntegrationService.
func NewIntegrationService(
	connStore integrationConnectionStore,
	syncer ConnectorSyncer,
	registry *connectorsdk.Registry,
) *IntegrationService {
	return &IntegrationService{
		connStore: connStore,
		syncer:    syncer,
		registry:  registry,
	}
}

// IntegrationSummary holds summary info for an integration connection.
type IntegrationSummary struct {
	Provider      string     `json:"provider"`
	Status        string     `json:"status"`
	LastSyncAt    *time.Time `json:"last_sync_at"`
	LastSyncError string     `json:"last_sync_error,omitempty"`
	CustomerCount int        `json:"customer_count"`
	ConnectedAt   time.Time  `json:"connected_at"`
}

// IntegrationStatus holds detailed status for an integration.
type IntegrationStatus struct {
	IntegrationSummary
	ExternalAccountID string   `json:"external_account_id"`
	Scopes            []string `json:"scopes"`
}

// ConnectIntegrationRequest holds generic connector credentials.
type ConnectIntegrationRequest struct {
	APIKey    string            `json:"api_key"`
	ProjectID string            `json:"project_id"`
	Metadata  map[string]string `json:"metadata"`
}

// Connect validates and stores credentials for connectors that support direct auth.
func (s *IntegrationService) Connect(ctx context.Context, orgID uuid.UUID, provider string, req ConnectIntegrationRequest) (*connectorsdk.AuthResult, error) {
	if s.registry == nil {
		return nil, &ValidationError{Field: "connector_registry", Message: "connector registry is not configured"}
	}
	registered, ok := s.registry.Get(provider)
	if !ok {
		return nil, &NotFoundError{Resource: "connector", Message: fmt.Sprintf("no %s connector registered", provider)}
	}
	if registered.Manifest.Auth.Type != connectorsdk.AuthTypeAPIKey {
		return nil, &ValidationError{Field: provider, Message: "direct API-key connection is not supported for this provider"}
	}
	metadata := req.Metadata
	if metadata == nil {
		metadata = map[string]string{}
	}
	if req.ProjectID != "" {
		metadata["project_id"] = req.ProjectID
	}
	return registered.Connector.Authenticate(ctx, connectorsdk.AuthRequest{
		OrgID:    orgID.String(),
		APIKey:   req.APIKey,
		Metadata: metadata,
	})
}

// List returns all integration connections for an org.
func (s *IntegrationService) List(ctx context.Context, orgID uuid.UUID) ([]IntegrationSummary, error) {
	conns, err := s.connStore.ListByOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("list integrations: %w", err)
	}

	connectionsByProvider := make(map[string]*repository.IntegrationConnection, len(conns))
	for _, conn := range conns {
		connectionsByProvider[conn.Provider] = conn
	}

	seen := make(map[string]bool, len(conns))
	summaries := make([]IntegrationSummary, 0, len(conns))
	if s.registry != nil {
		for _, registered := range s.registry.List() {
			provider := registered.Manifest.ID
			seen[provider] = true
			conn := connectionsByProvider[provider]
			if conn == nil {
				summaries = append(summaries, IntegrationSummary{Provider: provider, Status: "disconnected"})
				continue
			}
			summary, err := s.integrationSummary(ctx, orgID, conn)
			if err != nil {
				return nil, err
			}
			summaries = append(summaries, summary)
		}
	}

	for _, conn := range conns {
		if seen[conn.Provider] {
			continue
		}
		summary, err := s.integrationSummary(ctx, orgID, conn)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}

	return summaries, nil
}

func (s *IntegrationService) integrationSummary(ctx context.Context, orgID uuid.UUID, conn *repository.IntegrationConnection) (IntegrationSummary, error) {
	count, err := s.connStore.GetCustomerCountBySource(ctx, orgID, conn.Provider)
	if err != nil {
		return IntegrationSummary{}, fmt.Errorf("get customer count: %w", err)
	}
	return IntegrationSummary{
		Provider:      conn.Provider,
		Status:        conn.Status,
		LastSyncAt:    conn.LastSyncAt,
		LastSyncError: conn.LastSyncError,
		CustomerCount: count,
		ConnectedAt:   conn.CreatedAt,
	}, nil
}

// GetStatus returns detailed status for a specific integration provider.
func (s *IntegrationService) GetStatus(ctx context.Context, orgID uuid.UUID, provider string) (*IntegrationStatus, error) {
	conn, err := s.connStore.GetByOrgAndProvider(ctx, orgID, provider)
	if err != nil {
		return nil, fmt.Errorf("get integration status: %w", err)
	}
	if conn == nil {
		return nil, &NotFoundError{Resource: "integration", Message: fmt.Sprintf("no %s integration found", provider)}
	}

	count, err := s.connStore.GetCustomerCountBySource(ctx, orgID, provider)
	if err != nil {
		return nil, fmt.Errorf("get customer count: %w", err)
	}

	return &IntegrationStatus{
		IntegrationSummary: IntegrationSummary{
			Provider:      conn.Provider,
			Status:        conn.Status,
			LastSyncAt:    conn.LastSyncAt,
			LastSyncError: conn.LastSyncError,
			CustomerCount: count,
			ConnectedAt:   conn.CreatedAt,
		},
		ExternalAccountID: conn.ExternalAccountID,
		Scopes:            conn.Scopes,
	}, nil
}

// TriggerSync triggers a sync for a specific integration provider.
func (s *IntegrationService) TriggerSync(ctx context.Context, orgID uuid.UUID, provider string) error {
	conn, err := s.connStore.GetByOrgAndProvider(ctx, orgID, provider)
	if err != nil {
		return fmt.Errorf("get integration: %w", err)
	}
	if conn == nil {
		return &NotFoundError{Resource: "integration", Message: fmt.Sprintf("no %s integration found", provider)}
	}
	if conn.Status != "active" {
		return &ValidationError{Field: "status", Message: "integration is not active"}
	}
	if s.syncer == nil {
		return &ValidationError{Field: "syncer", Message: "connector syncer is not configured"}
	}

	go func() {
		_, _ = s.syncer.Sync(context.Background(), provider, orgID, connectorsdk.SyncModeFull, nil)
	}()

	return nil
}

// Disconnect removes an integration connection.
func (s *IntegrationService) Disconnect(ctx context.Context, orgID uuid.UUID, provider string) error {
	conn, err := s.connStore.GetByOrgAndProvider(ctx, orgID, provider)
	if err != nil {
		return fmt.Errorf("get integration: %w", err)
	}
	if conn == nil {
		return &NotFoundError{Resource: "integration", Message: fmt.Sprintf("no %s integration found", provider)}
	}

	if err := s.connStore.Delete(ctx, orgID, provider); err != nil {
		return fmt.Errorf("delete integration: %w", err)
	}

	return nil
}
