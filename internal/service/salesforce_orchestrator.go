package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

// SalesforceSyncOrchestratorService manages Salesforce sync connection state.
type SalesforceSyncOrchestratorService struct {
	connRepo *repository.IntegrationConnectionRepository
	syncSvc  *SalesforceSyncService
}

// NewSalesforceSyncOrchestratorService creates a Salesforce sync orchestrator.
func NewSalesforceSyncOrchestratorService(connRepo *repository.IntegrationConnectionRepository, syncSvc *SalesforceSyncService) *SalesforceSyncOrchestratorService {
	return &SalesforceSyncOrchestratorService{connRepo: connRepo, syncSvc: syncSvc}
}

// RunFullSync runs a full Salesforce sync.
func (s *SalesforceSyncOrchestratorService) RunFullSync(ctx context.Context, orgID uuid.UUID) *SalesforceSyncResult {
	return s.run(ctx, orgID, nil)
}

// RunIncrementalSync runs a Salesforce sync for records changed since the given time.
func (s *SalesforceSyncOrchestratorService) RunIncrementalSync(ctx context.Context, orgID uuid.UUID, since time.Time) *SalesforceSyncResult {
	return s.run(ctx, orgID, &since)
}

func (s *SalesforceSyncOrchestratorService) run(ctx context.Context, orgID uuid.UUID, since *time.Time) *SalesforceSyncResult {
	if s == nil || s.syncSvc == nil || s.connRepo == nil {
		return &SalesforceSyncResult{Errors: []string{"Salesforce sync orchestrator is not configured"}}
	}
	_ = s.connRepo.UpdateSyncStatus(ctx, orgID, providerSalesforce, "syncing", nil)
	result, err := s.syncSvc.Sync(ctx, orgID, since)
	if err != nil {
		_ = s.connRepo.UpdateErrorCount(ctx, orgID, providerSalesforce, err.Error())
		return &SalesforceSyncResult{Errors: []string{err.Error()}}
	}
	now := time.Now().UTC()
	_ = s.connRepo.UpdateSyncStatus(ctx, orgID, providerSalesforce, "active", &now)
	return result
}
