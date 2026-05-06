package service

import (
	"context"
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
	CreateInstallation(ctx context.Context, installation *repository.ConnectorInstallation) error
	CreateReviewResult(ctx context.Context, result *repository.ConnectorReviewResult) error
	UpdateConnectorStatus(ctx context.Context, id, version, status string) error
}

type connectorStatusNotifier interface {
	NotifyConnectorStatusChange(ctx context.Context, connector *repository.MarketplaceConnector, status string) error
}

type RegisterConnectorRequest struct {
	Manifest connectorsdk.ConnectorManifest `json:"manifest"`
	Status   string                         `json:"status,omitempty"`
}

type InstallConnectorRequest struct {
	Config map[string]any `json:"config,omitempty"`
}

// MarketplaceService handles connector registration and discovery.
type MarketplaceService struct {
	repo     marketplaceRepository
	notifier connectorStatusNotifier
}

func NewMarketplaceService(repo marketplaceRepository) *MarketplaceService {
	return &MarketplaceService{repo: repo}
}

func NewMarketplaceServiceWithNotifier(repo marketplaceRepository, notifier connectorStatusNotifier) *MarketplaceService {
	return &MarketplaceService{repo: repo, notifier: notifier}
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

func (s *MarketplaceService) Install(ctx context.Context, orgID uuid.UUID, id string, req InstallConnectorRequest) (*repository.ConnectorInstallation, error) {
	connector, err := s.GetPublished(ctx, id)
	if err != nil {
		return nil, err
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
	return installation, nil
}

func (s *MarketplaceService) ListReviewQueue(ctx context.Context) ([]*repository.MarketplaceConnector, error) {
	return s.repo.ListConnectorReviewQueue(ctx)
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
