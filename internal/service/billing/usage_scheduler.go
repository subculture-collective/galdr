package billing

import (
	"context"
	"log/slog"
	"time"

	"github.com/onnwee/pulse-score/internal/repository"
)

type usageOrganizationLister interface {
	ListActive(ctx context.Context) ([]repository.Organization, error)
}

// UsageScheduler records daily usage snapshots for all active organizations.
type UsageScheduler struct {
	orgs     usageOrganizationLister
	usage    *UsageService
	interval time.Duration
}

func NewUsageScheduler(orgs usageOrganizationLister, usage *UsageService, interval time.Duration) *UsageScheduler {
	return &UsageScheduler{orgs: orgs, usage: usage, interval: interval}
}

func (s *UsageScheduler) Start(ctx context.Context) {
	if s.interval <= 0 {
		return
	}
	s.recordAll(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.recordAll(ctx)
		}
	}
}

func (s *UsageScheduler) recordAll(ctx context.Context) {
	orgs, err := s.orgs.ListActive(ctx)
	if err != nil {
		slog.Error("list orgs for usage snapshots", "error", err)
		return
	}
	for _, org := range orgs {
		if err := s.usage.RecordDailySnapshot(ctx, org.ID); err != nil {
			slog.Error("record usage snapshot", "org_id", org.ID, "error", err)
		}
	}
}
