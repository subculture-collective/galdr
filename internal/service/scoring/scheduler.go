package scoring

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

// AlertCallback is called after a score is calculated, for real-time alert evaluation.
type AlertCallback func(ctx context.Context, customerID, orgID uuid.UUID)

// InsightCallback is called after a score is stored, with previous and current scores.
type InsightCallback func(ctx context.Context, previous, current *repository.HealthScore)

// ScoreScheduler handles periodic and event-triggered score recalculation.
type ScoreScheduler struct {
	aggregator      *ScoreAggregator
	healthScores    *repository.HealthScoreRepository
	customers       *repository.CustomerRepository
	connections     *repository.IntegrationConnectionRepository
	changeDetector  *ChangeDetector
	alertCallback   AlertCallback
	insightCallback InsightCallback
	interval        time.Duration
	workers         int
}

// NewScoreScheduler creates a new ScoreScheduler.
func NewScoreScheduler(
	aggregator *ScoreAggregator,
	healthScores *repository.HealthScoreRepository,
	customers *repository.CustomerRepository,
	connections *repository.IntegrationConnectionRepository,
	changeDetector *ChangeDetector,
	interval time.Duration,
	workers int,
) *ScoreScheduler {
	if workers <= 0 {
		workers = 5
	}
	return &ScoreScheduler{
		aggregator:     aggregator,
		healthScores:   healthScores,
		customers:      customers,
		connections:    connections,
		changeDetector: changeDetector,
		interval:       interval,
		workers:        workers,
	}
}

// SetAlertCallback registers a callback for real-time alert evaluation after score changes.
func (s *ScoreScheduler) SetAlertCallback(cb AlertCallback) {
	s.alertCallback = cb
}

// SetInsightCallback registers a callback for AI insight generation after score changes.
func (s *ScoreScheduler) SetInsightCallback(cb InsightCallback) {
	s.insightCallback = cb
}

// Start begins the periodic score recalculation. Cancel the context to stop.
func (s *ScoreScheduler) Start(ctx context.Context) {
	slog.Info("score scheduler started", "interval", s.interval, "workers", s.workers)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("score scheduler stopped")
			return
		case <-ticker.C:
			s.RunBatch(ctx)
		}
	}
}

// RunBatch recalculates scores for all customers across all active orgs.
func (s *ScoreScheduler) RunBatch(ctx context.Context) {
	conns, err := s.connections.ListActiveByProvider(ctx, "stripe")
	if err != nil {
		slog.Error("score scheduler: failed to list connections", "error", err)
		return
	}

	var totalCustomers, totalErrors int

	for _, conn := range conns {
		customers, err := s.customers.ListByOrg(ctx, conn.OrgID)
		if err != nil {
			slog.Error("score scheduler: failed to list customers", "org_id", conn.OrgID, "error", err)
			continue
		}

		processed, errors := s.processCustomersBatch(ctx, customers, conn.OrgID)
		totalCustomers += processed
		totalErrors += errors
	}

	slog.Info("score batch recalculation complete",
		"total_customers", totalCustomers,
		"errors", totalErrors,
	)
}

// RecalculateCustomer recalculates the score for a single customer (event-triggered).
func (s *ScoreScheduler) RecalculateCustomer(ctx context.Context, customerID, orgID uuid.UUID) error {
	return s.calculateAndStore(ctx, customerID, orgID)
}

// RecalculateOrg recalculates scores for all customers in an org.
func (s *ScoreScheduler) RecalculateOrg(ctx context.Context, orgID uuid.UUID) error {
	customers, err := s.customers.ListByOrg(ctx, orgID)
	if err != nil {
		return err
	}

	s.processCustomersBatch(ctx, customers, orgID)
	return nil
}

// processCustomersBatch processes a batch of customers with a worker pool.
func (s *ScoreScheduler) processCustomersBatch(ctx context.Context, customers []*repository.Customer, orgID uuid.UUID) (int, int) {
	if len(customers) == 0 {
		return 0, 0
	}

	type job struct {
		customerID uuid.UUID
		orgID      uuid.UUID
	}

	jobs := make(chan job, len(customers))
	var wg sync.WaitGroup
	var mu sync.Mutex
	var processed, errors int

	// Start workers
	for i := 0; i < s.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				if ctx.Err() != nil {
					return
				}
				if err := s.calculateAndStore(ctx, j.customerID, j.orgID); err != nil {
					slog.Error("score calculation error",
						"customer_id", j.customerID,
						"error", err,
					)
					mu.Lock()
					errors++
					mu.Unlock()
				} else {
					mu.Lock()
					processed++
					mu.Unlock()
				}
			}
		}()
	}

	// Send jobs
	for _, c := range customers {
		jobs <- job{customerID: c.ID, orgID: orgID}
	}
	close(jobs)

	wg.Wait()
	return processed, errors
}

// calculateAndStore calculates a score and persists it.
func (s *ScoreScheduler) calculateAndStore(ctx context.Context, customerID, orgID uuid.UUID) error {
	// Get previous score for change detection
	previous, err := s.healthScores.GetByCustomerID(ctx, customerID, orgID)
	if err != nil {
		slog.Error("get previous score error", "customer_id", customerID, "error", err)
		// Continue with calculation even if we can't get previous
	}

	result, err := s.aggregator.Calculate(ctx, customerID, orgID)
	if err != nil {
		return err
	}

	// Persist current score
	healthScore := &repository.HealthScore{
		OrgID:        orgID,
		CustomerID:   customerID,
		OverallScore: result.OverallScore,
		RiskLevel:    result.RiskLevel,
		Factors:      result.Factors,
		CalculatedAt: result.CalculatedAt,
	}

	if err := s.healthScores.UpsertCurrent(ctx, healthScore); err != nil {
		return err
	}

	// Insert into history
	if err := s.healthScores.InsertHistory(ctx, healthScore); err != nil {
		slog.Error("insert history error", "customer_id", customerID, "error", err)
	}

	// Run change detection
	if s.changeDetector != nil {
		if err := s.changeDetector.DetectAndRecord(ctx, previous, result); err != nil {
			slog.Error("change detection error", "customer_id", customerID, "error", err)
		}
	}

	// Fire alert callback for real-time evaluation
	if s.alertCallback != nil {
		s.alertCallback(ctx, customerID, orgID)
	}

	if s.insightCallback != nil {
		s.insightCallback(ctx, previous, healthScore)
	}

	return nil
}
