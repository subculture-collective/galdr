package service

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	connectorsdk "github.com/onnwee/pulse-score/pkg/connector-sdk"

	"github.com/onnwee/pulse-score/internal/repository"
)

// SyncSchedulerService runs periodic incremental syncs for all active connections.
type SyncSchedulerService struct {
	connRepo *repository.IntegrationConnectionRepository
	syncer   ConnectorSyncer
	registry *connectorsdk.Registry
	interval time.Duration

	// Per-connection lock to prevent overlapping syncs
	locks map[uuid.UUID]*sync.Mutex
	mu    sync.Mutex
}

// NewSyncSchedulerService creates a new SyncSchedulerService.
func NewSyncSchedulerService(
	connRepo *repository.IntegrationConnectionRepository,
	syncer ConnectorSyncer,
	registry *connectorsdk.Registry,
	intervalMinutes int,
) *SyncSchedulerService {
	return &SyncSchedulerService{
		connRepo: connRepo,
		syncer:   syncer,
		registry: registry,
		interval: time.Duration(intervalMinutes) * time.Minute,
		locks:    make(map[uuid.UUID]*sync.Mutex),
	}
}

// Start begins the periodic sync scheduler. Cancel the context to stop.
func (s *SyncSchedulerService) Start(ctx context.Context) {
	slog.Info("sync scheduler started", "interval", s.interval)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("sync scheduler stopped")
			return
		case <-ticker.C:
			s.runCycle(ctx)
		}
	}
}

func (s *SyncSchedulerService) runCycle(ctx context.Context) {
	if s.registry == nil || s.syncer == nil {
		slog.Error("scheduler: connector registry is not configured")
		return
	}

	for _, registered := range s.registry.List() {
		s.runProviderCycle(ctx, registered.Manifest.ID)
	}
}

func (s *SyncSchedulerService) runProviderCycle(ctx context.Context, provider string) {
	conns, err := s.connRepo.ListActiveByProvider(ctx, provider)
	if err != nil {
		slog.Error("scheduler: failed to list provider connections", "provider", provider, "error", err)
		return
	}

	for _, conn := range conns {
		lock := s.getLock(conn.OrgID)
		if !lock.TryLock() {
			slog.Debug("scheduler: skipping org (sync in progress)", "provider", provider, "org_id", conn.OrgID)
			continue
		}

		go func(orgID uuid.UUID, lastSync *time.Time, lock *sync.Mutex) {
			defer lock.Unlock()

			syncCtx, cancel := context.WithTimeout(ctx, 10*time.Minute)
			defer cancel()

			mode := connectorsdk.SyncModeFull
			if lastSync != nil {
				mode = connectorsdk.SyncModeIncremental
			}

			if _, err := s.syncer.Sync(syncCtx, provider, orgID, mode, lastSync); err != nil {
				slog.Error("scheduler: provider sync failed", "provider", provider, "org_id", orgID, "error", err)
			}
		}(conn.OrgID, conn.LastSyncAt, lock)
	}
}

func (s *SyncSchedulerService) getLock(orgID uuid.UUID) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.locks[orgID]; !ok {
		s.locks[orgID] = &sync.Mutex{}
	}
	return s.locks[orgID]
}
