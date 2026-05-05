package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

// ZendeskSyncResult contains Zendesk sync results.
type ZendeskSyncResult struct {
	Users        *SyncProgress        `json:"users"`
	Tickets      *SyncProgress        `json:"tickets"`
	Deduplicated *DeduplicationResult `json:"deduplicated,omitempty"`
	Duration     string               `json:"duration"`
	Errors       []string             `json:"errors,omitempty"`
}

// ZendeskSyncOrchestratorService orchestrates Zendesk sync.
type ZendeskSyncOrchestratorService struct {
	connRepo *repository.IntegrationConnectionRepository
	syncSvc  *ZendeskSyncService
	mergeSvc *CustomerMergeService
}

// NewZendeskSyncOrchestratorService creates a new ZendeskSyncOrchestratorService.
func NewZendeskSyncOrchestratorService(connRepo *repository.IntegrationConnectionRepository, syncSvc *ZendeskSyncService, mergeSvc *CustomerMergeService) *ZendeskSyncOrchestratorService {
	return &ZendeskSyncOrchestratorService{connRepo: connRepo, syncSvc: syncSvc, mergeSvc: mergeSvc}
}

// RunFullSync runs complete Zendesk sync.
func (s *ZendeskSyncOrchestratorService) RunFullSync(ctx context.Context, orgID uuid.UUID) *ZendeskSyncResult {
	start := time.Now()
	result := &ZendeskSyncResult{}
	if err := s.connRepo.UpdateSyncStatus(ctx, orgID, zendeskProvider, "syncing", nil); err != nil {
		slog.Error("failed to update zendesk sync status", "error", err)
	}
	users, err := s.syncSvc.SyncUsers(ctx, orgID)
	result.Users = users
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("user sync: %v", err))
		s.markSyncError(ctx, orgID, err.Error())
	}
	tickets, err := s.syncSvc.SyncTickets(ctx, orgID)
	result.Tickets = tickets
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("ticket sync: %v", err))
		s.markSyncError(ctx, orgID, err.Error())
	}
	if s.mergeSvc != nil {
		dedup, err := s.mergeSvc.DeduplicateCustomers(ctx, orgID)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("dedup: %v", err))
		} else {
			result.Deduplicated = dedup
		}
	}
	now := time.Now()
	if err := s.connRepo.UpdateSyncStatus(ctx, orgID, zendeskProvider, "active", &now); err != nil {
		slog.Error("failed to update zendesk sync status", "error", err)
	}
	result.Duration = time.Since(start).String()
	return result
}

// RunIncrementalSync runs Zendesk incremental sync.
func (s *ZendeskSyncOrchestratorService) RunIncrementalSync(ctx context.Context, orgID uuid.UUID, since time.Time) *ZendeskSyncResult {
	start := time.Now()
	result := &ZendeskSyncResult{}
	if err := s.connRepo.UpdateSyncStatus(ctx, orgID, zendeskProvider, "syncing", nil); err != nil {
		slog.Error("failed to update zendesk sync status", "error", err)
	}
	users, err := s.syncSvc.SyncUsersSince(ctx, orgID, since)
	result.Users = users
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("incremental user sync: %v", err))
		s.markSyncError(ctx, orgID, err.Error())
	}
	tickets, err := s.syncSvc.SyncTicketsSince(ctx, orgID, since)
	result.Tickets = tickets
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("incremental ticket sync: %v", err))
		s.markSyncError(ctx, orgID, err.Error())
	}
	if s.mergeSvc != nil {
		dedup, err := s.mergeSvc.DeduplicateCustomers(ctx, orgID)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("dedup: %v", err))
		} else {
			result.Deduplicated = dedup
		}
	}
	now := time.Now()
	if err := s.connRepo.UpdateSyncStatus(ctx, orgID, zendeskProvider, "active", &now); err != nil {
		slog.Error("failed to update zendesk sync status", "error", err)
	}
	result.Duration = time.Since(start).String()
	return result
}

func (s *ZendeskSyncOrchestratorService) markSyncError(ctx context.Context, orgID uuid.UUID, errMsg string) {
	if err := s.connRepo.UpdateErrorCount(ctx, orgID, zendeskProvider, errMsg); err != nil {
		slog.Error("failed to update zendesk error count", "error", err)
	}
}
