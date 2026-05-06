package service

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	connectorsdk "github.com/onnwee/pulse-score/pkg/connector-sdk"

	"github.com/onnwee/pulse-score/internal/repository"
)

const connectorProjectIDMetadataKey = "project_id"

const integrationHealthStaleAfter = 24 * time.Hour

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

// IntegrationHealthResponse reports sync health across all known integrations.
type IntegrationHealthResponse struct {
	GeneratedAt      time.Time                  `json:"generated_at"`
	StaleAfterHours int                        `json:"stale_after_hours"`
	Integrations     []IntegrationHealthSummary `json:"integrations"`
}

// IntegrationHealthSummary holds dashboard-level sync health for one provider.
type IntegrationHealthSummary struct {
	Provider       string                         `json:"provider"`
	Status         string                         `json:"status"`
	HealthStatus   string                         `json:"health_status"`
	LastSyncAt     *time.Time                     `json:"last_sync_at"`
	ConnectedAt    *time.Time                     `json:"connected_at"`
	RecordsSynced  int                            `json:"records_synced"`
	ErrorCount     int                            `json:"error_count"`
	SyncDurationMS int                            `json:"sync_duration_ms"`
	ErrorRate      float64                        `json:"error_rate"`
	CustomerCount  int                            `json:"customer_count"`
	LastSyncError  string                         `json:"last_sync_error,omitempty"`
	Alerts         []IntegrationHealthAlert       `json:"alerts"`
	SyncHistory    []IntegrationSyncHistoryPoint  `json:"sync_history"`
}

// IntegrationHealthAlert describes an actionable integration health warning.
type IntegrationHealthAlert struct {
	Type     string `json:"type"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// IntegrationSyncHistoryPoint is one sync metric point for dashboard charts.
type IntegrationSyncHistoryPoint struct {
	Date          string `json:"date"`
	Status        string `json:"status"`
	RecordsSynced int    `json:"records_synced"`
	DurationMS    int    `json:"duration_ms"`
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
	metadata := make(map[string]string, len(req.Metadata)+1)
	for key, value := range req.Metadata {
		metadata[key] = value
	}
	if req.ProjectID != "" {
		metadata[connectorProjectIDMetadataKey] = req.ProjectID
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

// GetHealth returns integration health metrics for the monitoring dashboard.
func (s *IntegrationService) GetHealth(ctx context.Context, orgID uuid.UUID) (*IntegrationHealthResponse, error) {
	conns, err := s.connStore.ListByOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("list integrations: %w", err)
	}

	connectionsByProvider := make(map[string]*repository.IntegrationConnection, len(conns))
	for _, conn := range conns {
		connectionsByProvider[conn.Provider] = conn
	}

	seen := make(map[string]bool, len(conns))
	integrations := make([]IntegrationHealthSummary, 0, len(conns))
	if s.registry != nil {
		for _, registered := range s.registry.List() {
			provider := registered.Manifest.ID
			seen[provider] = true
			integration, err := s.integrationHealth(ctx, orgID, provider, connectionsByProvider[provider])
			if err != nil {
				return nil, err
			}
			integrations = append(integrations, integration)
		}
	}

	for _, conn := range conns {
		if seen[conn.Provider] {
			continue
		}
		integration, err := s.integrationHealth(ctx, orgID, conn.Provider, conn)
		if err != nil {
			return nil, err
		}
		integrations = append(integrations, integration)
	}

	return &IntegrationHealthResponse{
		GeneratedAt:      time.Now().UTC(),
		StaleAfterHours: int(integrationHealthStaleAfter / time.Hour),
		Integrations:     integrations,
	}, nil
}

func (s *IntegrationService) integrationHealth(ctx context.Context, orgID uuid.UUID, provider string, conn *repository.IntegrationConnection) (IntegrationHealthSummary, error) {
	if conn == nil {
		return IntegrationHealthSummary{
			Provider:     provider,
			Status:       "disconnected",
			HealthStatus: "disconnected",
			Alerts:       []IntegrationHealthAlert{},
			SyncHistory:  []IntegrationSyncHistoryPoint{},
		}, nil
	}

	count, err := s.connStore.GetCustomerCountBySource(ctx, orgID, conn.Provider)
	if err != nil {
		return IntegrationHealthSummary{}, fmt.Errorf("get customer count: %w", err)
	}

	metadata := conn.Metadata
	recordsSynced := metadataInt(metadata, "records_synced", "last_records_synced")
	if recordsSynced == 0 {
		recordsSynced = count
	}
	errorCount := metadataInt(metadata, "error_count", "consecutive_errors")
	successCount := metadataInt(metadata, "success_count", "successful_sync_count")
	syncDurationMS := metadataInt(metadata, "sync_duration_ms", "last_sync_duration_ms")
	alerts := integrationHealthAlerts(conn, errorCount)
	healthStatus := integrationHealthStatus(conn, errorCount, len(alerts) > 0)

	return IntegrationHealthSummary{
		Provider:       conn.Provider,
		Status:         conn.Status,
		HealthStatus:   healthStatus,
		LastSyncAt:     conn.LastSyncAt,
		ConnectedAt:    &conn.CreatedAt,
		RecordsSynced:  recordsSynced,
		ErrorCount:     errorCount,
		SyncDurationMS: syncDurationMS,
		ErrorRate:      integrationErrorRate(errorCount, successCount),
		CustomerCount:  count,
		LastSyncError:  conn.LastSyncError,
		Alerts:         alerts,
		SyncHistory:    integrationSyncHistory(conn, recordsSynced, syncDurationMS),
	}, nil
}

func integrationHealthStatus(conn *repository.IntegrationConnection, errorCount int, hasAlerts bool) string {
	if conn.Status == "disconnected" {
		return "disconnected"
	}
	if conn.Status == "error" || errorCount >= 5 {
		return "down"
	}
	if hasAlerts || conn.LastSyncError != "" || errorCount > 0 {
		return "warning"
	}
	return "healthy"
}

func integrationHealthAlerts(conn *repository.IntegrationConnection, errorCount int) []IntegrationHealthAlert {
	alerts := make([]IntegrationHealthAlert, 0, 3)
	provider := providerDisplayName(conn.Provider)
	if conn.Status == "error" || errorCount >= 5 {
		alerts = append(alerts, IntegrationHealthAlert{Type: "integration_down", Severity: "critical", Message: fmt.Sprintf("%s sync is down.", provider)})
	}
	if errorCount >= 3 {
		alerts = append(alerts, IntegrationHealthAlert{Type: "consecutive_errors", Severity: "warning", Message: fmt.Sprintf("%s has %d consecutive sync errors.", provider, errorCount)})
	}
	if conn.Status == "active" && (conn.LastSyncAt == nil || time.Since(*conn.LastSyncAt) > integrationHealthStaleAfter) {
		alerts = append(alerts, IntegrationHealthAlert{Type: "sync_stale", Severity: "warning", Message: "Last successful sync is stale."})
	}
	return alerts
}

func integrationSyncHistory(conn *repository.IntegrationConnection, recordsSynced, durationMS int) []IntegrationSyncHistoryPoint {
	if history, ok := parseSyncHistory(conn.Metadata); ok {
		return history
	}
	if conn.LastSyncAt == nil {
		return []IntegrationSyncHistoryPoint{}
	}
	status := "success"
	if conn.LastSyncError != "" || conn.Status == "error" {
		status = "error"
	}
	return []IntegrationSyncHistoryPoint{{
		Date:          conn.LastSyncAt.UTC().Format("2006-01-02"),
		Status:        status,
		RecordsSynced: recordsSynced,
		DurationMS:    durationMS,
	}}
}

func parseSyncHistory(metadata map[string]any) ([]IntegrationSyncHistoryPoint, bool) {
	raw, ok := metadata["sync_history"].([]any)
	if !ok || len(raw) == 0 {
		return nil, false
	}
	history := make([]IntegrationSyncHistoryPoint, 0, len(raw))
	for _, item := range raw {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		date, _ := entry["date"].(string)
		status, _ := entry["status"].(string)
		if date == "" || status == "" {
			continue
		}
		history = append(history, IntegrationSyncHistoryPoint{
			Date:          date,
			Status:        status,
			RecordsSynced: metadataInt(entry, "records_synced"),
			DurationMS:    metadataInt(entry, "duration_ms"),
		})
	}
	return history, len(history) > 0
}

func integrationErrorRate(errorCount, successCount int) float64 {
	total := errorCount + successCount
	if total == 0 {
		return 0
	}
	return float64(errorCount) / float64(total)
}

func metadataInt(metadata map[string]any, keys ...string) int {
	for _, key := range keys {
		if value, ok := metadata[key]; ok {
			return anyToInt(value)
		}
	}
	return 0
}

func anyToInt(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case string:
		parsed, _ := strconv.Atoi(v)
		return parsed
	default:
		return 0
	}
}

func providerDisplayName(provider string) string {
	switch provider {
	case "stripe":
		return "Stripe"
	case "hubspot":
		return "HubSpot"
	case "intercom":
		return "Intercom"
	case "zendesk":
		return "Zendesk"
	case "salesforce":
		return "Salesforce"
	case "posthog":
		return "PostHog"
	case "":
		return "Integration"
	default:
		return provider
	}
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
