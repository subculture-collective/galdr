package service

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
	connectorsdk "github.com/onnwee/pulse-score/pkg/connector-sdk"
)

type mockConnectorReviewRepository struct {
	getConnectorFn          func(ctx context.Context, id, version string) (*repository.MarketplaceConnector, error)
	createReviewResultFn    func(ctx context.Context, result *repository.ConnectorReviewResult) error
	updateConnectorStatusFn func(ctx context.Context, id, version, status string) error
}

func (m *mockConnectorReviewRepository) GetConnector(ctx context.Context, id, version string) (*repository.MarketplaceConnector, error) {
	return m.getConnectorFn(ctx, id, version)
}

func (m *mockConnectorReviewRepository) CreateReviewResult(ctx context.Context, result *repository.ConnectorReviewResult) error {
	return m.createReviewResultFn(ctx, result)
}

func (m *mockConnectorReviewRepository) UpdateConnectorStatus(ctx context.Context, id, version, status string) error {
	return m.updateConnectorStatusFn(ctx, id, version, status)
}

func TestConnectorReviewRejectsInsecureOAuthURLAndRecordsResult(t *testing.T) {
	connector := marketplaceConnector("mock-crm", "1.0.0", repository.MarketplaceConnectorStatusSubmitted)
	connector.Manifest.Auth = connectorsdk.AuthConfig{
		Type: connectorsdk.AuthTypeOAuth2,
		OAuth2: &connectorsdk.OAuth2Config{
			AuthorizeURL: "http://provider.example/oauth/authorize",
			TokenURL:     "https://provider.example/oauth/token",
		},
	}
	var recorded *repository.ConnectorReviewResult
	repo := &mockConnectorReviewRepository{
		getConnectorFn: func(ctx context.Context, id, version string) (*repository.MarketplaceConnector, error) {
			return connector, nil
		},
		createReviewResultFn: func(ctx context.Context, result *repository.ConnectorReviewResult) error {
			recorded = result
			return nil
		},
		updateConnectorStatusFn: func(ctx context.Context, id, version, status string) error {
			if status != repository.MarketplaceConnectorStatusSubmitted {
				t.Fatalf("expected connector to remain submitted, got %q", status)
			}
			return nil
		},
	}

	result, err := NewConnectorReviewService(repo).Review(context.Background(), uuid.New(), connector.ID, connector.Version, ConnectorReviewRequest{
		Checklist: ConnectorSecurityChecklist{
			DataAccessJustified: true,
			ErrorHandlingReady:  true,
			DocumentationReady:  true,
		},
	})
	if err != nil {
		t.Fatalf("review failed: %v", err)
	}

	if recorded == nil {
		t.Fatal("expected review result to be recorded")
	}
	if result.Status != repository.ConnectorReviewStatusBlocked {
		t.Fatalf("expected blocked review, got %q", result.Status)
	}
	if !hasFailedReviewCheck(result.AutomatedChecks, "https_urls") {
		t.Fatalf("expected failed https_urls check, got %+v", result.AutomatedChecks)
	}
}

func TestConnectorReviewApprovesCompleteSecureSubmission(t *testing.T) {
	connector := marketplaceConnector("mock-crm", "1.0.0", repository.MarketplaceConnectorStatusSubmitted)
	var updatedStatus string
	repo := &mockConnectorReviewRepository{
		getConnectorFn: func(ctx context.Context, id, version string) (*repository.MarketplaceConnector, error) {
			return connector, nil
		},
		createReviewResultFn: func(ctx context.Context, result *repository.ConnectorReviewResult) error {
			return nil
		},
		updateConnectorStatusFn: func(ctx context.Context, id, version, status string) error {
			updatedStatus = status
			return nil
		},
	}

	result, err := NewConnectorReviewService(repo).Review(context.Background(), uuid.New(), connector.ID, connector.Version, completeConnectorReviewRequest())
	if err != nil {
		t.Fatalf("review failed: %v", err)
	}

	if result.Status != repository.ConnectorReviewStatusApproved {
		t.Fatalf("expected approved review, got %q", result.Status)
	}
	if updatedStatus != repository.MarketplaceConnectorStatusApproved {
		t.Fatalf("expected connector approved, got %q", updatedStatus)
	}
	if hasFailedReviewCheck(result.AutomatedChecks, reviewCheckNoCredentials) || hasFailedReviewCheck(result.SandboxChecks, reviewCheckSandboxSync) {
		t.Fatalf("expected passing checks, got automated=%+v sandbox=%+v", result.AutomatedChecks, result.SandboxChecks)
	}
}

func TestConnectorReviewBlocksIncompleteSecurityChecklist(t *testing.T) {
	connector := marketplaceConnector("mock-crm", "1.0.0", repository.MarketplaceConnectorStatusSubmitted)
	repo := reviewRepoForConnector(connector, func(ctx context.Context, id, version, status string) error {
		if status != repository.MarketplaceConnectorStatusSubmitted {
			t.Fatalf("expected connector to remain submitted, got %q", status)
		}
		return nil
	})

	result, err := NewConnectorReviewService(repo).Review(context.Background(), uuid.New(), connector.ID, connector.Version, ConnectorReviewRequest{
		Checklist: ConnectorSecurityChecklist{DataAccessJustified: true, ErrorHandlingReady: true},
	})
	if err != nil {
		t.Fatalf("review failed: %v", err)
	}

	if result.Status != repository.ConnectorReviewStatusBlocked {
		t.Fatalf("expected blocked review, got %q", result.Status)
	}
	if result.SecurityChecklist["documentation_ready"] {
		t.Fatal("expected missing documentation checklist gate to be recorded")
	}
}

func TestConnectorReviewBlocksHardcodedCredentials(t *testing.T) {
	connector := marketplaceConnector("mock-crm", "1.0.0", repository.MarketplaceConnectorStatusSubmitted)
	connector.Manifest.Sync.Options = map[string]string{"client_secret": "sk_live_123"}
	repo := reviewRepoForConnector(connector, nil)

	result, err := NewConnectorReviewService(repo).Review(context.Background(), uuid.New(), connector.ID, connector.Version, completeConnectorReviewRequest())
	if err != nil {
		t.Fatalf("review failed: %v", err)
	}

	if result.Status != repository.ConnectorReviewStatusBlocked || !hasFailedReviewCheck(result.AutomatedChecks, reviewCheckNoCredentials) {
		t.Fatalf("expected credential block, got status=%q checks=%+v", result.Status, result.AutomatedChecks)
	}
}

func TestConnectorReviewAllowsWebhookSigningHeaderName(t *testing.T) {
	connector := marketplaceConnector("mock-crm", "1.0.0", repository.MarketplaceConnectorStatusSubmitted)
	connector.Manifest.Webhooks = []connectorsdk.WebhookConfig{
		{Path: "/api/v1/webhooks/connectors/mock-crm", EventTypes: []string{"customer.updated"}, SigningSecretHeader: "X-Mock-Signature"},
	}
	repo := reviewRepoForConnector(connector, nil)

	result, err := NewConnectorReviewService(repo).Review(context.Background(), uuid.New(), connector.ID, connector.Version, completeConnectorReviewRequest())
	if err != nil {
		t.Fatalf("review failed: %v", err)
	}

	if result.Status != repository.ConnectorReviewStatusApproved || hasFailedReviewCheck(result.AutomatedChecks, reviewCheckNoCredentials) {
		t.Fatalf("expected signing header to pass credential scan, got status=%q checks=%+v", result.Status, result.AutomatedChecks)
	}
}

func TestConnectorReviewBlocksInvalidRateLimit(t *testing.T) {
	connector := marketplaceConnector("mock-crm", "1.0.0", repository.MarketplaceConnectorStatusSubmitted)
	connector.Manifest.Sync.Options = map[string]string{"rate_limit_per_minute": "900"}
	repo := reviewRepoForConnector(connector, nil)

	result, err := NewConnectorReviewService(repo).Review(context.Background(), uuid.New(), connector.ID, connector.Version, completeConnectorReviewRequest())
	if err != nil {
		t.Fatalf("review failed: %v", err)
	}

	if result.Status != repository.ConnectorReviewStatusBlocked || !hasFailedReviewCheck(result.AutomatedChecks, reviewCheckRateLimits) {
		t.Fatalf("expected rate limit block, got status=%q checks=%+v", result.Status, result.AutomatedChecks)
	}
}

func TestConnectorReviewBlocksSandboxSyncFailure(t *testing.T) {
	connector := marketplaceConnector("mock-crm", "1.0.0", repository.MarketplaceConnectorStatusSubmitted)
	connector.Manifest.Sync.Resources = nil
	repo := reviewRepoForConnector(connector, nil)

	result, err := NewConnectorReviewService(repo).Review(context.Background(), uuid.New(), connector.ID, connector.Version, completeConnectorReviewRequest())
	if err != nil {
		t.Fatalf("review failed: %v", err)
	}

	if result.Status != repository.ConnectorReviewStatusBlocked || !hasFailedReviewCheck(result.SandboxChecks, reviewCheckSandboxSync) {
		t.Fatalf("expected sandbox block, got status=%q checks=%+v", result.Status, result.SandboxChecks)
	}
}

func completeConnectorReviewRequest() ConnectorReviewRequest {
	return ConnectorReviewRequest{
		Checklist: ConnectorSecurityChecklist{
			DataAccessJustified: true,
			ErrorHandlingReady:  true,
			DocumentationReady:  true,
		},
	}
}

func reviewRepoForConnector(connector *repository.MarketplaceConnector, updateFn func(ctx context.Context, id, version, status string) error) *mockConnectorReviewRepository {
	if updateFn == nil {
		updateFn = func(ctx context.Context, id, version, status string) error { return nil }
	}
	return &mockConnectorReviewRepository{
		getConnectorFn: func(ctx context.Context, id, version string) (*repository.MarketplaceConnector, error) {
			return connector, nil
		},
		createReviewResultFn: func(ctx context.Context, result *repository.ConnectorReviewResult) error {
			return nil
		},
		updateConnectorStatusFn: updateFn,
	}
}

func hasFailedReviewCheck(checks []repository.ConnectorReviewCheck, name string) bool {
	for _, check := range checks {
		if check.Name == name && check.Status == repository.ConnectorReviewCheckFailed {
			return true
		}
	}
	return false
}
