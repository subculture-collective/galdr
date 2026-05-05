package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	PublishedAt *time.Time                     `json:"published_at,omitempty"`
	CreatedAt   time.Time                      `json:"created_at"`
	UpdatedAt   time.Time                      `json:"updated_at"`
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
