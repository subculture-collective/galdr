package service

import (
	"context"
	"log/slog"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
	connectorsdk "github.com/onnwee/pulse-score/pkg/connector-sdk"
)

type marketplaceRepository interface {
	CreateConnector(ctx context.Context, connector *repository.MarketplaceConnector) error
	GetConnector(ctx context.Context, id, version string) (*repository.MarketplaceConnector, error)
	ListPublishedConnectors(ctx context.Context) ([]*repository.MarketplaceConnector, error)
	ListConnectorReviewQueue(ctx context.Context) ([]*repository.MarketplaceConnector, error)
	SearchConnectors(ctx context.Context, req repository.MarketplaceSearchRequest) ([]*repository.MarketplaceConnector, error)
	ListInstalledProviders(ctx context.Context, orgID uuid.UUID) ([]string, error)
	CreateInstallation(ctx context.Context, installation *repository.ConnectorInstallation) error
	IncrementConnectorInstallMetric(ctx context.Context, connectorID string, at time.Time) error
	GetConnectorAnalytics(ctx context.Context, connectorID string, since time.Time) (*repository.ConnectorAnalytics, error)
	CreateReviewResult(ctx context.Context, result *repository.ConnectorReviewResult) error
	UpdateConnectorStatus(ctx context.Context, id, version, status string) error
}

type marketplaceConnectionStore interface {
	Upsert(ctx context.Context, conn *repository.IntegrationConnection) error
}

type connectorStatusNotifier interface {
	NotifyConnectorStatusChange(ctx context.Context, connector *repository.MarketplaceConnector, status string) error
}

type RegisterConnectorRequest struct {
	Manifest connectorsdk.ConnectorManifest `json:"manifest"`
	Status   string                         `json:"status,omitempty"`
}

type InstallConnectorRequest struct {
	Auth           InstallConnectorAuth `json:"auth,omitempty"`
	Config         map[string]any       `json:"config,omitempty"`
	TestConnection bool                 `json:"test_connection,omitempty"`
}

type InstallConnectorAuth struct {
	Type              string `json:"type,omitempty"`
	APIKey            string `json:"api_key,omitempty"`
	OAuthAuthorizeURL string `json:"oauth_authorize_url,omitempty"`
}

type MarketplaceSearchRequest struct {
	Query    string `json:"query,omitempty"`
	Category string `json:"category,omitempty"`
	Tag      string `json:"tag,omitempty"`
	Sort     string `json:"sort,omitempty"`
}

type MarketplaceSearchResponse struct {
	Connectors       []*repository.MarketplaceConnector `json:"connectors"`
	Recommendations  []*repository.MarketplaceConnector `json:"recommendations"`
	Query            string                             `json:"query,omitempty"`
	Category         string                             `json:"category,omitempty"`
	Tag              string                             `json:"tag,omitempty"`
	Sort             string                             `json:"sort"`
}

// MarketplaceService handles connector registration and discovery.
type MarketplaceService struct {
	repo      marketplaceRepository
	connStore marketplaceConnectionStore
	notifier  connectorStatusNotifier
}

func NewMarketplaceService(repo marketplaceRepository, connStore ...marketplaceConnectionStore) *MarketplaceService {
	service := &MarketplaceService{repo: repo}
	if len(connStore) > 0 {
		service.connStore = connStore[0]
	}
	return service
}

func NewMarketplaceServiceWithNotifier(repo marketplaceRepository, notifier connectorStatusNotifier, connStore ...marketplaceConnectionStore) *MarketplaceService {
	service := NewMarketplaceService(repo, connStore...)
	service.notifier = notifier
	return service
}

func (s *MarketplaceService) Register(ctx context.Context, developerID uuid.UUID, req RegisterConnectorRequest) (*repository.MarketplaceConnector, error) {
	if err := connectorsdk.ValidateManifest(req.Manifest); err != nil {
		return nil, &ValidationError{Field: "manifest", Message: err.Error()}
	}

	status, err := marketplaceStatusOrDefault(req.Status)
	if err != nil {
		return nil, err
	}

	existing, err := s.repo.GetConnector(ctx, req.Manifest.ID, req.Manifest.Version)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, &ConflictError{Resource: "marketplace_connector", Message: "connector version already exists"}
	}

	connector := newMarketplaceConnector(developerID, req.Manifest, status)

	if err := s.repo.CreateConnector(ctx, connector); err != nil {
		return nil, err
	}
	return connector, nil
}

func marketplaceStatusOrDefault(status string) (string, error) {
	status = strings.TrimSpace(status)
	if status == "" {
		return repository.MarketplaceConnectorStatusDraft, nil
	}
	if !isValidMarketplaceStatus(status) {
		return "", &ValidationError{Field: "status", Message: "invalid connector status"}
	}
	return status, nil
}

func newMarketplaceConnector(developerID uuid.UUID, manifest connectorsdk.ConnectorManifest, status string) *repository.MarketplaceConnector {
	connector := &repository.MarketplaceConnector{
		ID:          manifest.ID,
		Version:     manifest.Version,
		DeveloperID: developerID,
		Name:        manifest.Name,
		Description: manifest.Description,
		Manifest:    manifest,
		Status:      status,
	}
	if status == repository.MarketplaceConnectorStatusPublished {
		now := time.Now().UTC()
		connector.PublishedAt = &now
	}
	return connector
}

func (s *MarketplaceService) ListPublished(ctx context.Context) ([]*repository.MarketplaceConnector, error) {
	connectors, err := s.repo.ListPublishedConnectors(ctx)
	if err != nil {
		return nil, err
	}
	return latestConnectorVersions(connectors), nil
}

func (s *MarketplaceService) GetPublished(ctx context.Context, id string) (*repository.MarketplaceConnector, error) {
	connectors, err := s.ListPublished(ctx)
	if err != nil {
		return nil, err
	}
	for _, connector := range connectors {
		if connector.ID == id {
			return connector, nil
		}
	}
	return nil, &NotFoundError{Resource: "marketplace_connector", Message: "connector not found"}
}

func (s *MarketplaceService) Search(ctx context.Context, orgID uuid.UUID, req MarketplaceSearchRequest) (*MarketplaceSearchResponse, error) {
	searchReq := repository.MarketplaceSearchRequest{
		Query:    strings.TrimSpace(req.Query),
		Category: strings.ToLower(strings.TrimSpace(req.Category)),
		Tag:      strings.ToLower(strings.TrimSpace(req.Tag)),
		Sort:     marketplaceSearchSortOrDefault(req.Sort),
	}

	connectors, err := s.repo.SearchConnectors(ctx, searchReq)
	if err != nil {
		return nil, err
	}
	connectors = latestConnectorVersions(connectors)
	applyMarketplaceSearchSort(connectors, searchReq.Sort)

	installedProviders, err := s.repo.ListInstalledProviders(ctx, orgID)
	if err != nil {
		return nil, err
	}
	recommendations, err := s.recommendConnectors(ctx, installedProviders)
	if err != nil {
		return nil, err
	}

	return &MarketplaceSearchResponse{
		Connectors:      connectors,
		Recommendations: recommendations,
		Query:           searchReq.Query,
		Category:        searchReq.Category,
		Tag:             searchReq.Tag,
		Sort:            searchReq.Sort,
	}, nil
}

func (s *MarketplaceService) Install(ctx context.Context, orgID uuid.UUID, id string, req InstallConnectorRequest) (*repository.ConnectorInstallation, error) {
	connector, err := s.GetPublished(ctx, id)
	if err != nil {
		return nil, err
	}
	if req.Auth.Type != "" && !req.TestConnection {
		return nil, &ValidationError{Field: "test_connection", Message: "connection test is required before activation"}
	}
	installation := &repository.ConnectorInstallation{
		ConnectorID:      connector.ID,
		ConnectorVersion: connector.Version,
		OrgID:            orgID,
		Config:           req.Config,
		Status:           repository.ConnectorInstallationStatusActive,
	}
	if err := s.repo.CreateInstallation(ctx, installation); err != nil {
		return nil, err
	}
	if s.connStore != nil {
		metadata := map[string]any{
			"connector_version": connector.Version,
			"marketplace":       true,
		}
		if req.Auth.Type != "" {
			metadata["auth_type"] = req.Auth.Type
		}
		if err := s.connStore.Upsert(ctx, &repository.IntegrationConnection{
			OrgID:             orgID,
			Provider:          connector.ID,
			Status:            integrationStatusActive,
			ExternalAccountID: connector.ID,
			Metadata:          metadata,
		}); err != nil {
			return nil, err
		}
	}
	if err := s.repo.IncrementConnectorInstallMetric(ctx, connector.ID, time.Now().UTC()); err != nil {
		slog.Warn("failed to record connector install metric", "connector", connector.ID, "error", err)
	}
	return installation, nil
}

func (s *MarketplaceService) Analytics(ctx context.Context, id string) (*repository.ConnectorAnalytics, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil, &ValidationError{Field: "connector_id", Message: "connector id is required"}
	}
	since := time.Now().UTC().AddDate(0, 0, -30)
	return s.repo.GetConnectorAnalytics(ctx, id, since)
}

func (s *MarketplaceService) ListReviewQueue(ctx context.Context) ([]*repository.MarketplaceConnector, error) {
	return s.repo.ListConnectorReviewQueue(ctx)
}

func (s *MarketplaceService) recommendConnectors(ctx context.Context, installedProviders []string) ([]*repository.MarketplaceConnector, error) {
	connectors, err := s.repo.ListPublishedConnectors(ctx)
	if err != nil {
		return nil, err
	}
	connectors = latestConnectorVersions(connectors)

	installed := make(map[string]struct{}, len(installedProviders))
	wantedCategories := make(map[string]struct{})
	for _, provider := range installedProviders {
		provider = strings.ToLower(strings.TrimSpace(provider))
		if provider == "" {
			continue
		}
		installed[provider] = struct{}{}
		for _, category := range recommendedCategoriesForProvider(provider) {
			wantedCategories[category] = struct{}{}
		}
	}
	if len(wantedCategories) == 0 {
		wantedCategories["crm"] = struct{}{}
		wantedCategories["support"] = struct{}{}
		wantedCategories["analytics"] = struct{}{}
	}

	type scoredConnector struct {
		connector *repository.MarketplaceConnector
		score     int
	}
	scored := make([]scoredConnector, 0, len(connectors))
	for _, connector := range connectors {
		if connector == nil || connectorMatchesInstalled(connector, installed) {
			continue
		}
		score := connector.InstallCount
		for _, category := range connector.Manifest.Categories {
			if _, ok := wantedCategories[strings.ToLower(strings.TrimSpace(category))]; ok {
				score += 1000
			}
		}
		if score > 0 {
			scored = append(scored, scoredConnector{connector: connector, score: score})
		}
	}

	slices.SortFunc(scored, func(a, b scoredConnector) int {
		if a.score != b.score {
			return compareInts(b.score, a.score)
		}
		return strings.Compare(a.connector.ID, b.connector.ID)
	})

	limit := 3
	recommendations := make([]*repository.MarketplaceConnector, 0, limit)
	for _, item := range scored {
		if len(recommendations) == limit {
			break
		}
		recommendations = append(recommendations, item.connector)
	}
	return recommendations, nil
}

func (s *MarketplaceService) Review(ctx context.Context, reviewerID uuid.UUID, id, version string, req ConnectorReviewRequest) (*repository.ConnectorReviewResult, error) {
	return NewConnectorReviewServiceWithNotifier(s.repo, s.notifier).Review(ctx, reviewerID, id, version, req)
}

func (s *MarketplaceService) Reject(ctx context.Context, id, version string) (*repository.MarketplaceConnector, error) {
	connector, err := s.repo.GetConnector(ctx, id, version)
	if err != nil {
		return nil, err
	}
	if connector == nil {
		return nil, &NotFoundError{Resource: "marketplace_connector", Message: "connector not found"}
	}
	if err := s.repo.UpdateConnectorStatus(ctx, id, version, repository.MarketplaceConnectorStatusRejected); err != nil {
		return nil, err
	}
	connector.Status = repository.MarketplaceConnectorStatusRejected
	if err := s.notifyStatusChange(ctx, connector, repository.MarketplaceConnectorStatusRejected); err != nil {
		return nil, err
	}
	return connector, nil
}

func (s *MarketplaceService) Publish(ctx context.Context, id, version string) (*repository.MarketplaceConnector, error) {
	connector, err := s.repo.GetConnector(ctx, id, version)
	if err != nil {
		return nil, err
	}
	if connector == nil {
		return nil, &NotFoundError{Resource: "marketplace_connector", Message: "connector not found"}
	}
	if connector.Status != repository.MarketplaceConnectorStatusApproved {
		return nil, &ValidationError{Field: "status", Message: "connector must be approved before publishing"}
	}
	if err := s.repo.UpdateConnectorStatus(ctx, id, version, repository.MarketplaceConnectorStatusPublished); err != nil {
		return nil, err
	}
	connector.Status = repository.MarketplaceConnectorStatusPublished
	if connector.PublishedAt == nil {
		now := time.Now().UTC()
		connector.PublishedAt = &now
	}
	if err := s.notifyStatusChange(ctx, connector, repository.MarketplaceConnectorStatusPublished); err != nil {
		return nil, err
	}
	return connector, nil
}

func (s *MarketplaceService) notifyStatusChange(ctx context.Context, connector *repository.MarketplaceConnector, status string) error {
	if s.notifier == nil {
		return nil
	}
	return s.notifier.NotifyConnectorStatusChange(ctx, connector, status)
}

func latestConnectorVersions(connectors []*repository.MarketplaceConnector) []*repository.MarketplaceConnector {
	latestByID := make(map[string]*repository.MarketplaceConnector, len(connectors))
	for _, connector := range connectors {
		if connector == nil {
			continue
		}
		current, ok := latestByID[connector.ID]
		if !ok || compareSemver(connector.Version, current.Version) > 0 {
			latestByID[connector.ID] = connector
		}
	}

	ids := make([]string, 0, len(latestByID))
	for id := range latestByID {
		ids = append(ids, id)
	}
	slices.Sort(ids)

	result := make([]*repository.MarketplaceConnector, 0, len(ids))
	for _, id := range ids {
		result = append(result, latestByID[id])
	}
	return result
}

func compareSemver(a, b string) int {
	aParts := semverCoreParts(a)
	bParts := semverCoreParts(b)
	for i := 0; i < 3; i++ {
		if cmp := compareInts(aParts[i], bParts[i]); cmp != 0 {
			return cmp
		}
	}
	return comparePrerelease(semverPrerelease(a), semverPrerelease(b))
}

func semverCoreParts(version string) [3]int {
	core := strings.SplitN(strings.SplitN(version, "+", 2)[0], "-", 2)[0]
	parts := strings.Split(core, ".")
	var nums [3]int
	for i := 0; i < len(parts) && i < 3; i++ {
		n, err := strconv.Atoi(parts[i])
		if err != nil {
			return [3]int{}
		}
		nums[i] = n
	}
	return nums
}

func semverPrerelease(version string) string {
	withoutBuild := strings.SplitN(version, "+", 2)[0]
	parts := strings.SplitN(withoutBuild, "-", 2)
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}

func comparePrerelease(a, b string) int {
	if a == "" && b == "" {
		return 0
	}
	if a == "" {
		return 1
	}
	if b == "" {
		return -1
	}
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")
	for i := 0; i < len(aParts) && i < len(bParts); i++ {
		if cmp := comparePrereleaseIdentifier(aParts[i], bParts[i]); cmp != 0 {
			return cmp
		}
	}
	return compareInts(len(aParts), len(bParts))
}

func comparePrereleaseIdentifier(a, b string) int {
	aNum, aErr := strconv.Atoi(a)
	bNum, bErr := strconv.Atoi(b)
	aIsNum := aErr == nil
	bIsNum := bErr == nil

	if aIsNum && bIsNum {
		return compareInts(aNum, bNum)
	}
	if aIsNum {
		return -1
	}
	if bIsNum {
		return 1
	}
	return strings.Compare(a, b)
}

func compareInts(a, b int) int {
	if a > b {
		return 1
	}
	if a < b {
		return -1
	}
	return 0
}

func marketplaceSearchSortOrDefault(sort string) string {
	switch strings.ToLower(strings.TrimSpace(sort)) {
	case repository.MarketplaceSearchSortPopularity:
		return repository.MarketplaceSearchSortPopularity
	case repository.MarketplaceSearchSortRating:
		return repository.MarketplaceSearchSortRating
	case repository.MarketplaceSearchSortNewest:
		return repository.MarketplaceSearchSortNewest
	default:
		return repository.MarketplaceSearchSortRelevance
	}
}

func applyMarketplaceSearchSort(connectors []*repository.MarketplaceConnector, sort string) {
	slices.SortFunc(connectors, func(a, b *repository.MarketplaceConnector) int {
		switch sort {
		case repository.MarketplaceSearchSortPopularity:
			if a.InstallCount != b.InstallCount {
				return compareInts(b.InstallCount, a.InstallCount)
			}
		case repository.MarketplaceSearchSortRating:
			if a.Rating != b.Rating {
				if a.Rating > b.Rating {
					return -1
				}
				return 1
			}
		case repository.MarketplaceSearchSortNewest:
			return compareConnectorNewest(a, b)
		default:
			if a.Relevance != b.Relevance {
				if a.Relevance > b.Relevance {
					return -1
				}
				return 1
			}
		}
		return strings.Compare(a.ID, b.ID)
	})
}

func compareConnectorNewest(a, b *repository.MarketplaceConnector) int {
	aTime := a.CreatedAt
	if a.PublishedAt != nil {
		aTime = *a.PublishedAt
	}
	bTime := b.CreatedAt
	if b.PublishedAt != nil {
		bTime = *b.PublishedAt
	}
	if !aTime.Equal(bTime) {
		if aTime.After(bTime) {
			return -1
		}
		return 1
	}
	return strings.Compare(a.ID, b.ID)
}

func recommendedCategoriesForProvider(provider string) []string {
	switch provider {
	case "stripe":
		return []string{"payments", "crm", "analytics"}
	case "hubspot", "salesforce":
		return []string{"crm", "support", "analytics"}
	case "intercom", "zendesk":
		return []string{"support", "communication", "analytics"}
	case "posthog":
		return []string{"analytics", "devops", "crm"}
	default:
		return []string{"crm", "support", "analytics"}
	}
}

func connectorMatchesInstalled(connector *repository.MarketplaceConnector, installed map[string]struct{}) bool {
	connectorID := strings.ToLower(connector.ID)
	connectorName := strings.ToLower(connector.Name)
	for provider := range installed {
		if strings.Contains(connectorID, provider) || strings.Contains(connectorName, provider) {
			return true
		}
	}
	return false
}

func isValidMarketplaceStatus(status string) bool {
	switch status {
	case repository.MarketplaceConnectorStatusDraft,
		repository.MarketplaceConnectorStatusSubmitted,
		repository.MarketplaceConnectorStatusUnderReview,
		repository.MarketplaceConnectorStatusApproved,
		repository.MarketplaceConnectorStatusRejected,
		repository.MarketplaceConnectorStatusPublished,
		repository.MarketplaceConnectorStatusDeprecated:
		return true
	default:
		return false
	}
}
