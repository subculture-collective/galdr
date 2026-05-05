package repository

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	connectorsdk "github.com/onnwee/pulse-score/pkg/connector-sdk"
)

type noRowsScanner struct{}

func (noRowsScanner) Scan(dest ...any) error {
	return pgx.ErrNoRows
}

func TestMarketplaceConnectorStatusConstants(t *testing.T) {
	cases := map[string]string{
		MarketplaceConnectorStatusDraft:      "draft",
		MarketplaceConnectorStatusSubmitted:  "submitted",
		MarketplaceConnectorStatusApproved:   "approved",
		MarketplaceConnectorStatusPublished:  "published",
		MarketplaceConnectorStatusDeprecated: "deprecated",
	}
	for got, want := range cases {
		if got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	}
}

func TestMarketplaceConnectorModel(t *testing.T) {
	developerID := uuid.New()
	now := time.Now()
	connector := &MarketplaceConnector{
		ID:          "mock-crm",
		Version:     "1.0.0",
		DeveloperID: developerID,
		Name:        "Mock CRM",
		Description: "Mock connector",
		Manifest: connectorsdk.ConnectorManifest{
			ID:      "mock-crm",
			Name:    "Mock CRM",
			Version: "1.0.0",
		},
		Status:      MarketplaceConnectorStatusPublished,
		PublishedAt: &now,
	}

	if connector.DeveloperID != developerID {
		t.Fatalf("expected developer id %s, got %s", developerID, connector.DeveloperID)
	}
	if connector.Manifest.ID != connector.ID || connector.Manifest.Version != connector.Version {
		t.Fatal("expected manifest identity to match connector identity")
	}
}

func TestConnectorInstallationModel(t *testing.T) {
	orgID := uuid.New()
	installation := &ConnectorInstallation{
		ID:               uuid.New(),
		OrgID:            orgID,
		ConnectorID:      "mock-crm",
		ConnectorVersion: "1.0.0",
		Config:           map[string]any{"region": "us"},
		Status:           ConnectorInstallationStatusActive,
	}

	if installation.OrgID != orgID {
		t.Fatalf("expected org id %s, got %s", orgID, installation.OrgID)
	}
	if installation.Config["region"] != "us" {
		t.Fatalf("expected config to be retained")
	}
}

func TestConnectorReviewResultModel(t *testing.T) {
	reviewerID := uuid.New()
	result := &ConnectorReviewResult{
		ConnectorID:      "mock-crm",
		ConnectorVersion: "1.0.0",
		ReviewerID:       reviewerID,
		Status:           ConnectorReviewStatusBlocked,
		AutomatedChecks: []ConnectorReviewCheck{
			{Name: "https_urls", Status: ConnectorReviewCheckFailed, Message: "oauth2 authorize_url must be an https URL"},
		},
		SecurityChecklist: map[string]bool{"data_access_justified": true},
		SandboxChecks:     []ConnectorReviewCheck{{Name: "sandbox_sync", Status: ConnectorReviewCheckPassed}},
	}

	if result.ReviewerID != reviewerID {
		t.Fatalf("expected reviewer id %s, got %s", reviewerID, result.ReviewerID)
	}
	if result.AutomatedChecks[0].Status != ConnectorReviewCheckFailed || !result.SecurityChecklist["data_access_justified"] {
		t.Fatal("expected review result checks to be retained")
	}
}

func TestScanMarketplaceConnectorPreservesNoRowsError(t *testing.T) {
	_, err := scanMarketplaceConnector(noRowsScanner{})
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("expected pgx.ErrNoRows, got %v", err)
	}
}

func TestMarketplaceMigrationUpFileContainsTables(t *testing.T) {
	data, err := os.ReadFile("../../migrations/000027_create_marketplace_connectors.up.sql")
	if err != nil {
		t.Fatalf("failed to read migration file: %v", err)
	}
	sql := string(data)
	for _, table := range []string{"marketplace_connectors", "connector_installations"} {
		if !strings.Contains(sql, "CREATE TABLE "+table) {
			t.Fatalf("migration up file missing CREATE TABLE %s", table)
		}
	}
	if !strings.Contains(sql, "PRIMARY KEY (id, version)") {
		t.Fatal("marketplace_connectors must support versioned identity")
	}
	if !strings.Contains(sql, "UNIQUE (org_id, connector_id)") {
		t.Fatal("connector_installations must prevent duplicate org installs")
	}
}

func TestMarketplaceMigrationDownFileDropsTables(t *testing.T) {
	data, err := os.ReadFile("../../migrations/000027_create_marketplace_connectors.down.sql")
	if err != nil {
		t.Fatalf("failed to read migration file: %v", err)
	}
	sql := string(data)
	for _, stmt := range []string{"DROP TABLE IF EXISTS connector_installations", "DROP TABLE IF EXISTS marketplace_connectors"} {
		if !strings.Contains(sql, stmt) {
			t.Fatalf("migration down file missing statement: %s", stmt)
		}
	}
}

func TestConnectorReviewMigrationContainsReviewResults(t *testing.T) {
	data, err := os.ReadFile("../../migrations/000029_create_connector_review_results.up.sql")
	if err != nil {
		t.Fatalf("failed to read migration file: %v", err)
	}
	sql := string(data)
	for _, fragment := range []string{
		"CREATE TABLE connector_review_results",
		"FOREIGN KEY (connector_id, connector_version)",
		"automated_checks JSONB NOT NULL",
		"security_checklist JSONB NOT NULL",
		"sandbox_checks JSONB NOT NULL",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("review migration missing fragment: %s", fragment)
		}
	}

	down, err := os.ReadFile("../../migrations/000029_create_connector_review_results.down.sql")
	if err != nil {
		t.Fatalf("failed to read migration file: %v", err)
	}
	if !strings.Contains(string(down), "DROP TABLE IF EXISTS connector_review_results") {
		t.Fatal("review migration down file must drop connector_review_results")
	}
}
