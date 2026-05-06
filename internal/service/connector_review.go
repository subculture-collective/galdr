package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
	connectorsdk "github.com/onnwee/pulse-score/pkg/connector-sdk"
)

const (
	reviewCheckManifestValid = "manifest_valid"
	reviewCheckHTTPSURLs     = "https_urls"
	reviewCheckNoCredentials = "no_hardcoded_credentials"
	reviewCheckRateLimits    = "rate_limit_compliance"
	reviewCheckSandboxSync   = "sandbox_sync"
)

type connectorReviewRepository interface {
	GetConnector(ctx context.Context, id, version string) (*repository.MarketplaceConnector, error)
	CreateReviewResult(ctx context.Context, result *repository.ConnectorReviewResult) error
	UpdateConnectorStatus(ctx context.Context, id, version, status string) error
}

// ConnectorSecurityChecklist captures the manual admin review gates.
type ConnectorSecurityChecklist struct {
	DataAccessJustified bool `json:"data_access_justified"`
	ErrorHandlingReady  bool `json:"error_handling_ready"`
	DocumentationReady  bool `json:"documentation_ready"`
}

type ConnectorReviewRequest struct {
	Checklist ConnectorSecurityChecklist `json:"checklist"`
}

// ConnectorReviewService reviews submitted marketplace connectors.
type ConnectorReviewService struct {
	repo     connectorReviewRepository
	notifier connectorStatusNotifier
}

func NewConnectorReviewService(repo connectorReviewRepository) *ConnectorReviewService {
	return &ConnectorReviewService{repo: repo}
}

func NewConnectorReviewServiceWithNotifier(repo connectorReviewRepository, notifier connectorStatusNotifier) *ConnectorReviewService {
	return &ConnectorReviewService{repo: repo, notifier: notifier}
}

func (s *ConnectorReviewService) Review(ctx context.Context, reviewerID uuid.UUID, id, version string, req ConnectorReviewRequest) (*repository.ConnectorReviewResult, error) {
	connector, err := s.repo.GetConnector(ctx, id, version)
	if err != nil {
		return nil, err
	}
	if connector == nil {
		return nil, &NotFoundError{Resource: "marketplace_connector", Message: "connector not found"}
	}

	automatedChecks := runAutomatedConnectorChecks(connector.Manifest)
	sandboxChecks := runConnectorSandboxChecks(connector.Manifest)
	status := repository.ConnectorReviewStatusApproved
	connectorStatus := repository.MarketplaceConnectorStatusApproved
	if hasBlockingReviewCheck(automatedChecks) || !req.Checklist.complete() || hasBlockingReviewCheck(sandboxChecks) {
		status = repository.ConnectorReviewStatusBlocked
		connectorStatus = repository.MarketplaceConnectorStatusRejected
	}

	result := &repository.ConnectorReviewResult{
		ConnectorID:       connector.ID,
		ConnectorVersion:  connector.Version,
		ReviewerID:        reviewerID,
		Status:            status,
		AutomatedChecks:   automatedChecks,
		SecurityChecklist: req.Checklist.toMap(),
		SandboxChecks:     sandboxChecks,
	}
	if err := s.repo.CreateReviewResult(ctx, result); err != nil {
		return nil, err
	}
	if err := s.repo.UpdateConnectorStatus(ctx, connector.ID, connector.Version, connectorStatus); err != nil {
		return nil, err
	}
	connector.Status = connectorStatus
	if s.notifier != nil {
		if err := s.notifier.NotifyConnectorStatusChange(ctx, connector, connectorStatus); err != nil {
			return nil, err
		}
	}
	return result, nil
}

func runAutomatedConnectorChecks(manifest connectorsdk.ConnectorManifest) []repository.ConnectorReviewCheck {
	checks := make([]repository.ConnectorReviewCheck, 0, 4)
	if err := connectorsdk.ValidateManifest(manifest); err != nil {
		checks = append(checks, failedReviewCheck(reviewCheckManifestValid, err.Error()))
	} else {
		checks = append(checks, passedReviewCheck(reviewCheckManifestValid))
	}
	checks = append(checks, checkManifestURLs(manifest))
	checks = append(checks, checkManifestCredentials(manifest))
	checks = append(checks, checkManifestRateLimits(manifest))
	return checks
}

func runConnectorSandboxChecks(manifest connectorsdk.ConnectorManifest) []repository.ConnectorReviewCheck {
	if len(manifest.Sync.Resources) == 0 {
		return []repository.ConnectorReviewCheck{failedReviewCheck(reviewCheckSandboxSync, "sandbox requires at least one sync resource")}
	}
	return []repository.ConnectorReviewCheck{passedReviewCheck(reviewCheckSandboxSync)}
}

func checkManifestURLs(manifest connectorsdk.ConnectorManifest) repository.ConnectorReviewCheck {
	if manifest.Auth.Type != connectorsdk.AuthTypeOAuth2 || manifest.Auth.OAuth2 == nil {
		return passedReviewCheck(reviewCheckHTTPSURLs)
	}
	for field, rawURL := range map[string]string{
		"authorize_url": manifest.Auth.OAuth2.AuthorizeURL,
		"token_url":     manifest.Auth.OAuth2.TokenURL,
	} {
		parsed, err := url.Parse(strings.TrimSpace(rawURL))
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
			return failedReviewCheck(reviewCheckHTTPSURLs, fmt.Sprintf("oauth2 %s must be an https URL", field))
		}
	}
	return passedReviewCheck(reviewCheckHTTPSURLs)
}

func checkManifestCredentials(manifest connectorsdk.ConnectorManifest) repository.ConnectorReviewCheck {
	payload, err := json.Marshal(manifest)
	if err != nil {
		return failedReviewCheck(reviewCheckNoCredentials, "manifest could not be scanned")
	}
	var data any
	if err := json.Unmarshal(payload, &data); err != nil {
		return failedReviewCheck(reviewCheckNoCredentials, "manifest could not be scanned")
	}
	if key, value, ok := findCredentialValue(data); ok {
		return failedReviewCheck(reviewCheckNoCredentials, fmt.Sprintf("manifest contains hardcoded credential %s=%q", key, value))
	}
	return passedReviewCheck(reviewCheckNoCredentials)
}

func checkManifestRateLimits(manifest connectorsdk.ConnectorManifest) repository.ConnectorReviewCheck {
	if manifest.Sync.Schedule != nil && manifest.Sync.Schedule.IntervalMinutes < 5 {
		return failedReviewCheck(reviewCheckRateLimits, "sync schedule interval must be at least 5 minutes")
	}
	rawLimit := strings.TrimSpace(manifest.Sync.Options["rate_limit_per_minute"])
	if rawLimit == "" {
		return passedReviewCheck(reviewCheckRateLimits)
	}
	limit, err := strconv.Atoi(rawLimit)
	if err != nil || limit <= 0 || limit > 600 {
		return failedReviewCheck(reviewCheckRateLimits, "rate_limit_per_minute must be between 1 and 600")
	}
	return passedReviewCheck(reviewCheckRateLimits)
}

func findCredentialValue(value any) (string, string, bool) {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			if str, ok := child.(string); ok && isCredentialKey(key) && isCredentialValue(str) {
				return key, str, true
			}
			if key, value, ok := findCredentialValue(child); ok {
				return key, value, true
			}
		}
	case []any:
		for _, child := range typed {
			if key, value, ok := findCredentialValue(child); ok {
				return key, value, true
			}
		}
	}
	return "", "", false
}

func isCredentialKey(key string) bool {
	key = strings.ToLower(key)
	if strings.Contains(key, "header") {
		return false
	}
	credentialMarkers := []string{"secret", "password", "access_token", "refresh_token", "client_secret"}
	for _, marker := range credentialMarkers {
		if strings.Contains(key, marker) {
			return true
		}
	}
	return false
}

func isCredentialValue(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" || strings.Contains(value, "placeholder") || strings.Contains(value, "example") || strings.Contains(value, "${") {
		return false
	}
	return true
}

func (c ConnectorSecurityChecklist) complete() bool {
	return c.DataAccessJustified && c.ErrorHandlingReady && c.DocumentationReady
}

func (c ConnectorSecurityChecklist) toMap() map[string]bool {
	return map[string]bool{
		"data_access_justified": c.DataAccessJustified,
		"error_handling_ready":  c.ErrorHandlingReady,
		"documentation_ready":  c.DocumentationReady,
	}
}

func hasBlockingReviewCheck(checks []repository.ConnectorReviewCheck) bool {
	for _, check := range checks {
		if check.Status == repository.ConnectorReviewCheckFailed {
			return true
		}
	}
	return false
}

func passedReviewCheck(name string) repository.ConnectorReviewCheck {
	return repository.ConnectorReviewCheck{Name: name, Status: repository.ConnectorReviewCheckPassed}
}

func failedReviewCheck(name, message string) repository.ConnectorReviewCheck {
	return repository.ConnectorReviewCheck{Name: name, Status: repository.ConnectorReviewCheckFailed, Message: message}
}
