package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	connectorsdk "github.com/onnwee/pulse-score/pkg/connector-sdk"
)

type marketplaceConnectorScanner interface {
	Scan(dest ...any) error
}

const (
	MarketplaceConnectorStatusDraft      = "draft"
	MarketplaceConnectorStatusSubmitted  = "submitted"
	MarketplaceConnectorStatusApproved   = "approved"
	MarketplaceConnectorStatusPublished  = "published"
	MarketplaceConnectorStatusDeprecated = "deprecated"

	ConnectorInstallationStatusActive      = "active"
	ConnectorInstallationStatusDisabled    = "disabled"
	ConnectorInstallationStatusError       = "error"
	ConnectorInstallationStatusUninstalled = "uninstalled"

	ConnectorReviewStatusApproved = "approved"
	ConnectorReviewStatusBlocked  = "blocked"

	ConnectorReviewCheckPassed = "passed"
	ConnectorReviewCheckFailed = "failed"

	MarketplaceSearchSortRelevance  = "relevance"
	MarketplaceSearchSortPopularity = "popularity"
	MarketplaceSearchSortRating     = "rating"
	MarketplaceSearchSortNewest     = "newest"

	connectorErrorRateAlertThreshold = 20.0
)

// MarketplaceConnector represents a versioned marketplace connector manifest.
type MarketplaceConnector struct {
	ID          string                         `json:"id"`
	Version     string                         `json:"version"`
	DeveloperID uuid.UUID                      `json:"developer_id"`
	Name        string                         `json:"name"`
	Description string                         `json:"description"`
	Manifest    connectorsdk.ConnectorManifest `json:"manifest"`
	Status      string                         `json:"status"`
	InstallCount int                           `json:"install_count,omitempty"`
	Rating      float64                        `json:"rating,omitempty"`
	Relevance   float64                        `json:"relevance,omitempty"`
	PublishedAt *time.Time                     `json:"published_at,omitempty"`
	CreatedAt   time.Time                      `json:"created_at"`
	UpdatedAt   time.Time                      `json:"updated_at"`
}

type MarketplaceSearchRequest struct {
	Query    string
	Category string
	Tag      string
	Sort     string
}

// ConnectorInstallation represents a connector installation for an org.
type ConnectorInstallation struct {
	ID               uuid.UUID      `json:"id"`
	ConnectorID      string         `json:"connector_id"`
	ConnectorVersion string         `json:"connector_version"`
	OrgID            uuid.UUID      `json:"org_id"`
	Config           map[string]any `json:"config"`
	Status           string         `json:"status"`
	InstalledAt      time.Time      `json:"installed_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

// ConnectorDailyMetric stores one daily connector analytics aggregate.
type ConnectorDailyMetric struct {
	ConnectorID              string    `json:"connector_id"`
	MetricDate               time.Time `json:"metric_date"`
	InstallCount             int       `json:"install_count"`
	ActiveInstalls           int       `json:"active_installs"`
	SyncSuccessCount         int       `json:"sync_success_count"`
	SyncFailureCount         int       `json:"sync_failure_count"`
	AvgSyncDurationMS        int64     `json:"avg_sync_duration_ms"`
	SyncSuccessRate          float64   `json:"sync_success_rate"`
	ErrorRate                float64   `json:"error_rate"`
	UninstallCount           int       `json:"uninstall_count"`
	UninstallRate            float64   `json:"uninstall_rate"`
	AlertThresholdBreached   bool      `json:"alert_threshold_breached"`
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
}

// ConnectorAnalytics summarizes connector usage and health metrics.
type ConnectorAnalytics struct {
	ConnectorID              string                 `json:"connector_id"`
	InstallCount             int                    `json:"install_count"`
	ActiveInstalls           int                    `json:"active_installs"`
	SyncSuccessRate          float64                `json:"sync_success_rate"`
	AvgSyncDurationMS        int64                  `json:"avg_sync_duration_ms"`
	ErrorRate                float64                `json:"error_rate"`
	UninstallRate            float64                `json:"uninstall_rate"`
	AlertThresholdBreached   bool                   `json:"alert_threshold_breached"`
	Metrics                  []ConnectorDailyMetric `json:"metrics"`
}

// ConnectorReviewCheck records one automated or sandbox review check.
type ConnectorReviewCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// ConnectorReviewResult records the review outcome for a submitted connector.
type ConnectorReviewResult struct {
	ID                uuid.UUID              `json:"id"`
	ConnectorID       string                 `json:"connector_id"`
	ConnectorVersion  string                 `json:"connector_version"`
	ReviewerID        uuid.UUID              `json:"reviewer_id"`
	Status            string                 `json:"status"`
	AutomatedChecks   []ConnectorReviewCheck `json:"automated_checks"`
	SecurityChecklist map[string]bool        `json:"security_checklist"`
	SandboxChecks     []ConnectorReviewCheck `json:"sandbox_checks"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

// MarketplaceRepository handles connector marketplace persistence.
type MarketplaceRepository struct {
	pool *pgxpool.Pool
}

// NewMarketplaceRepository creates a MarketplaceRepository.
func NewMarketplaceRepository(pool *pgxpool.Pool) *MarketplaceRepository {
	return &MarketplaceRepository{pool: pool}
}

// CreateConnector inserts a versioned connector manifest.
func (r *MarketplaceRepository) CreateConnector(ctx context.Context, connector *MarketplaceConnector) error {
	manifest, err := json.Marshal(connector.Manifest)
	if err != nil {
		return fmt.Errorf("marshal connector manifest: %w", err)
	}

	return r.pool.QueryRow(ctx, `
		INSERT INTO marketplace_connectors (id, version, developer_id, name, description, manifest, status, published_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at
	`, connector.ID, connector.Version, connector.DeveloperID, connector.Name, connector.Description, manifest, connector.Status, connector.PublishedAt,
	).Scan(&connector.CreatedAt, &connector.UpdatedAt)
}

// GetConnector returns a connector by versioned identity.
func (r *MarketplaceRepository) GetConnector(ctx context.Context, id, version string) (*MarketplaceConnector, error) {
	connector, err := scanMarketplaceConnector(r.pool.QueryRow(ctx, `
		SELECT id, version, developer_id, name, description, manifest, status, published_at, created_at, updated_at
		FROM marketplace_connectors
		WHERE id = $1 AND version = $2
	`, id, version))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query marketplace connector: %w", err)
	}
	return connector, nil
}

// ListPublishedConnectors returns every published connector version.
func (r *MarketplaceRepository) ListPublishedConnectors(ctx context.Context) ([]*MarketplaceConnector, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, version, developer_id, name, description, manifest, status, published_at, created_at, updated_at
		FROM marketplace_connectors
		WHERE status = 'published'
		ORDER BY id ASC, version DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query published marketplace connectors: %w", err)
	}
	defer rows.Close()

	var connectors []*MarketplaceConnector
	for rows.Next() {
		connector, err := scanMarketplaceConnector(rows)
		if err != nil {
			return nil, err
		}
		connectors = append(connectors, connector)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate marketplace connectors: %w", err)
	}
	return connectors, nil
}

// SearchConnectors searches published connectors by text, category, tag, and sort.
func (r *MarketplaceRepository) SearchConnectors(ctx context.Context, req MarketplaceSearchRequest) ([]*MarketplaceConnector, error) {
	query := strings.TrimSpace(req.Query)
	category := strings.ToLower(strings.TrimSpace(req.Category))
	tag := strings.ToLower(strings.TrimSpace(req.Tag))
	sort := marketplaceSearchSortOrDefault(req.Sort)

	orderBy := "relevance DESC, id ASC, version DESC"
	switch sort {
	case MarketplaceSearchSortPopularity:
		orderBy = "install_count DESC, relevance DESC, id ASC, version DESC"
	case MarketplaceSearchSortRating:
		orderBy = "rating DESC, relevance DESC, id ASC, version DESC"
	case MarketplaceSearchSortNewest:
		orderBy = "published_at DESC NULLS LAST, created_at DESC, id ASC, version DESC"
	}

	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
		WITH searchable AS (
			SELECT c.id, c.version, c.developer_id, c.name, c.description, c.manifest, c.status,
				c.published_at, c.created_at, c.updated_at,
				COALESCE(SUM(m.install_count), 0)::int AS install_count,
				0::float8 AS rating,
				setweight(to_tsvector('english', COALESCE(c.name, '')), 'A') ||
				setweight(to_tsvector('english', COALESCE(c.description, '')), 'B') ||
				setweight(to_tsvector('english', COALESCE(c.manifest->>'description', '')), 'B') ||
				setweight(to_tsvector('english', COALESCE((SELECT string_agg(value, ' ') FROM jsonb_array_elements_text(COALESCE(c.manifest->'categories', '[]'::jsonb))), '')), 'C') ||
				setweight(to_tsvector('english', COALESCE((SELECT string_agg(value, ' ') FROM jsonb_array_elements_text(COALESCE(c.manifest->'tags', '[]'::jsonb))), '')), 'C') AS document
			FROM marketplace_connectors c
			LEFT JOIN connector_metrics m ON m.connector_id = c.id
			WHERE c.status = 'published'
			GROUP BY c.id, c.version, c.developer_id, c.name, c.description, c.manifest, c.status, c.published_at, c.created_at, c.updated_at
		), ranked AS (
			SELECT *, CASE WHEN $1 = '' THEN 0 ELSE ts_rank_cd(document, plainto_tsquery('english', $1)) END AS relevance
			FROM searchable
			WHERE ($1 = '' OR document @@ plainto_tsquery('english', $1))
				AND ($2 = '' OR EXISTS (SELECT 1 FROM jsonb_array_elements_text(COALESCE(manifest->'categories', '[]'::jsonb)) AS category(value) WHERE lower(category.value) = $2))
				AND ($3 = '' OR EXISTS (SELECT 1 FROM jsonb_array_elements_text(COALESCE(manifest->'tags', '[]'::jsonb)) AS tag(value) WHERE lower(tag.value) = $3))
		)
		SELECT id, version, developer_id, name, description, manifest, status, published_at, created_at, updated_at,
			install_count, rating, relevance
		FROM ranked
		ORDER BY %s
	`, orderBy), query, category, tag)
	if err != nil {
		return nil, fmt.Errorf("search marketplace connectors: %w", err)
	}
	defer rows.Close()

	var connectors []*MarketplaceConnector
	for rows.Next() {
		connector, err := scanMarketplaceConnectorWithStats(rows)
		if err != nil {
			return nil, err
		}
		connectors = append(connectors, connector)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate marketplace search results: %w", err)
	}
	return connectors, nil
}

// ListInstalledProviders returns active first-party integration providers for recommendations.
func (r *MarketplaceRepository) ListInstalledProviders(ctx context.Context, orgID uuid.UUID) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT provider
		FROM integration_connections
		WHERE org_id = $1 AND status = 'active'
		ORDER BY provider ASC
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf("query installed integration providers: %w", err)
	}
	defer rows.Close()

	var providers []string
	for rows.Next() {
		var provider string
		if err := rows.Scan(&provider); err != nil {
			return nil, fmt.Errorf("scan installed integration provider: %w", err)
		}
		providers = append(providers, provider)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate installed integration providers: %w", err)
	}
	return providers, nil
}

// CreateInstallation creates an org connector installation.
func (r *MarketplaceRepository) CreateInstallation(ctx context.Context, installation *ConnectorInstallation) error {
	if installation.Config == nil {
		installation.Config = map[string]any{}
	}
	config, err := json.Marshal(installation.Config)
	if err != nil {
		return fmt.Errorf("marshal connector installation config: %w", err)
	}

	return r.pool.QueryRow(ctx, `
		INSERT INTO connector_installations (connector_id, connector_version, org_id, config, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, installed_at, updated_at
	`, installation.ConnectorID, installation.ConnectorVersion, installation.OrgID, config, installation.Status,
	).Scan(&installation.ID, &installation.InstalledAt, &installation.UpdatedAt)
}

// IncrementConnectorInstallMetric increments the daily install aggregate.
func (r *MarketplaceRepository) IncrementConnectorInstallMetric(ctx context.Context, connectorID string, at time.Time) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO connector_metrics (connector_id, metric_date, install_count, active_installs)
		VALUES ($1, $2, 1, (
			SELECT COUNT(*) FROM connector_installations WHERE connector_id = $1 AND status = $3
		))
		ON CONFLICT (connector_id, metric_date)
		DO UPDATE SET
			install_count = connector_metrics.install_count + 1,
			active_installs = EXCLUDED.active_installs,
			updated_at = NOW()
	`, connectorID, metricDate(at), ConnectorInstallationStatusActive)
	if err != nil {
		return fmt.Errorf("increment connector install metric: %w", err)
	}
	return nil
}

// RecordConnectorSyncMetric increments daily sync success/failure and duration aggregates.
func (r *MarketplaceRepository) RecordConnectorSyncMetric(ctx context.Context, connectorID string, duration time.Duration, succeeded bool, at time.Time) error {
	successCount := 0
	failureCount := 0
	if succeeded {
		successCount = 1
	} else {
		failureCount = 1
	}
	durationMS := duration.Milliseconds()
	if durationMS < 0 {
		durationMS = 0
	}

	_, err := r.pool.Exec(ctx, `
		INSERT INTO connector_metrics (connector_id, metric_date, active_installs, sync_success_count, sync_failure_count, total_sync_duration_ms)
		VALUES ($1, $2, (
			SELECT COUNT(*) FROM connector_installations WHERE connector_id = $1 AND status = $6
		), $3, $4, $5)
		ON CONFLICT (connector_id, metric_date)
		DO UPDATE SET
			active_installs = EXCLUDED.active_installs,
			sync_success_count = connector_metrics.sync_success_count + EXCLUDED.sync_success_count,
			sync_failure_count = connector_metrics.sync_failure_count + EXCLUDED.sync_failure_count,
			total_sync_duration_ms = connector_metrics.total_sync_duration_ms + EXCLUDED.total_sync_duration_ms,
			updated_at = NOW()
	`, connectorID, metricDate(at), successCount, failureCount, durationMS, ConnectorInstallationStatusActive)
	if err != nil {
		return fmt.Errorf("record connector sync metric: %w", err)
	}
	return nil
}

// GetConnectorAnalytics returns connector usage and health analytics since a date.
func (r *MarketplaceRepository) GetConnectorAnalytics(ctx context.Context, connectorID string, since time.Time) (*ConnectorAnalytics, error) {
	activeInstalls := 0
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM connector_installations
		WHERE connector_id = $1 AND status = $2
	`, connectorID, ConnectorInstallationStatusActive).Scan(&activeInstalls); err != nil {
		return nil, fmt.Errorf("query connector active installs: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT connector_id, metric_date, install_count, active_installs, sync_success_count,
			sync_failure_count, total_sync_duration_ms, uninstall_count, created_at, updated_at
		FROM connector_metrics
		WHERE connector_id = $1 AND metric_date >= $2
		ORDER BY metric_date ASC
	`, connectorID, metricDate(since))
	if err != nil {
		return nil, fmt.Errorf("query connector metrics: %w", err)
	}
	defer rows.Close()

	analytics := &ConnectorAnalytics{ConnectorID: connectorID, ActiveInstalls: activeInstalls}
	var totalSyncDurationMS int64
	var syncSuccessCount, syncFailureCount, uninstallCount int
	for rows.Next() {
		metric := ConnectorDailyMetric{}
		var totalMetricDurationMS int64
		if err := rows.Scan(
			&metric.ConnectorID,
			&metric.MetricDate,
			&metric.InstallCount,
			&metric.ActiveInstalls,
			&metric.SyncSuccessCount,
			&metric.SyncFailureCount,
			&totalMetricDurationMS,
			&metric.UninstallCount,
			&metric.CreatedAt,
			&metric.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan connector metric: %w", err)
		}
		metric.AvgSyncDurationMS = avgDurationMS(totalMetricDurationMS, metric.SyncSuccessCount+metric.SyncFailureCount)
		metric.SyncSuccessRate = percentage(metric.SyncSuccessCount, metric.SyncSuccessCount+metric.SyncFailureCount)
		metric.ErrorRate = percentage(metric.SyncFailureCount, metric.SyncSuccessCount+metric.SyncFailureCount)
		metric.UninstallRate = percentage(metric.UninstallCount, metric.InstallCount+metric.UninstallCount)
		metric.AlertThresholdBreached = metric.ErrorRate >= connectorErrorRateAlertThreshold

		analytics.InstallCount += metric.InstallCount
		syncSuccessCount += metric.SyncSuccessCount
		syncFailureCount += metric.SyncFailureCount
		uninstallCount += metric.UninstallCount
		totalSyncDurationMS += totalMetricDurationMS
		analytics.Metrics = append(analytics.Metrics, metric)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate connector metrics: %w", err)
	}

	syncTotal := syncSuccessCount + syncFailureCount
	analytics.SyncSuccessRate = percentage(syncSuccessCount, syncTotal)
	analytics.ErrorRate = percentage(syncFailureCount, syncTotal)
	analytics.AvgSyncDurationMS = avgDurationMS(totalSyncDurationMS, syncTotal)
	analytics.UninstallRate = percentage(uninstallCount, analytics.InstallCount+uninstallCount)
	analytics.AlertThresholdBreached = analytics.ErrorRate >= connectorErrorRateAlertThreshold
	return analytics, nil
}

// CreateReviewResult records the connector review outcome for one submission.
func (r *MarketplaceRepository) CreateReviewResult(ctx context.Context, result *ConnectorReviewResult) error {
	automatedChecks, err := json.Marshal(result.AutomatedChecks)
	if err != nil {
		return fmt.Errorf("marshal connector review automated checks: %w", err)
	}
	securityChecklist, err := json.Marshal(result.SecurityChecklist)
	if err != nil {
		return fmt.Errorf("marshal connector review security checklist: %w", err)
	}
	sandboxChecks, err := json.Marshal(result.SandboxChecks)
	if err != nil {
		return fmt.Errorf("marshal connector review sandbox checks: %w", err)
	}

	return r.pool.QueryRow(ctx, `
		INSERT INTO connector_review_results (connector_id, connector_version, reviewer_id, status, automated_checks, security_checklist, sandbox_checks)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`, result.ConnectorID, result.ConnectorVersion, result.ReviewerID, result.Status, automatedChecks, securityChecklist, sandboxChecks,
	).Scan(&result.ID, &result.CreatedAt, &result.UpdatedAt)
}

// UpdateConnectorStatus updates a versioned marketplace connector status.
func (r *MarketplaceRepository) UpdateConnectorStatus(ctx context.Context, id, version, status string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE marketplace_connectors
		SET status = $3, updated_at = NOW()
		WHERE id = $1 AND version = $2
	`, id, version, status)
	if err != nil {
		return fmt.Errorf("update marketplace connector status: %w", err)
	}
	return nil
}

func scanMarketplaceConnector(scanner marketplaceConnectorScanner) (*MarketplaceConnector, error) {
	connector := &MarketplaceConnector{}
	var manifest []byte
	if err := scanner.Scan(
		&connector.ID, &connector.Version, &connector.DeveloperID, &connector.Name, &connector.Description,
		&manifest, &connector.Status, &connector.PublishedAt, &connector.CreatedAt, &connector.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan marketplace connector: %w", err)
	}
	if err := json.Unmarshal(manifest, &connector.Manifest); err != nil {
		return nil, fmt.Errorf("unmarshal connector manifest: %w", err)
	}
	return connector, nil
}

func scanMarketplaceConnectorWithStats(scanner marketplaceConnectorScanner) (*MarketplaceConnector, error) {
	connector := &MarketplaceConnector{}
	var manifest []byte
	if err := scanner.Scan(
		&connector.ID, &connector.Version, &connector.DeveloperID, &connector.Name, &connector.Description,
		&manifest, &connector.Status, &connector.PublishedAt, &connector.CreatedAt, &connector.UpdatedAt,
		&connector.InstallCount, &connector.Rating, &connector.Relevance,
	); err != nil {
		return nil, fmt.Errorf("scan marketplace connector: %w", err)
	}
	if err := json.Unmarshal(manifest, &connector.Manifest); err != nil {
		return nil, fmt.Errorf("unmarshal connector manifest: %w", err)
	}
	return connector, nil
}

func marketplaceSearchSortOrDefault(sort string) string {
	switch sort {
	case MarketplaceSearchSortPopularity, MarketplaceSearchSortRating, MarketplaceSearchSortNewest, MarketplaceSearchSortRelevance:
		return sort
	default:
		return MarketplaceSearchSortRelevance
	}
}

func metricDate(at time.Time) time.Time {
	if at.IsZero() {
		at = time.Now().UTC()
	}
	at = at.UTC()
	return time.Date(at.Year(), at.Month(), at.Day(), 0, 0, 0, 0, time.UTC)
}

func percentage(part, total int) float64 {
	if total <= 0 {
		return 0
	}
	return (float64(part) / float64(total)) * 100
}

func avgDurationMS(total int64, count int) int64 {
	if count <= 0 {
		return 0
	}
	return total / int64(count)
}
