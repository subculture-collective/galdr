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

type rowScanner interface {
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

func scanMarketplaceConnector(scanner rowScanner) (*MarketplaceConnector, error) {
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
