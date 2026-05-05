package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/onnwee/pulse-score/internal/repository"
)

// ZendeskMetrics exposes support ticket metrics for scoring.
type ZendeskMetrics struct {
	CustomerID          uuid.UUID `json:"customer_id"`
	OpenTickets         int       `json:"open_tickets"`
	SolvedTickets       int       `json:"solved_tickets"`
	TotalTickets        int       `json:"total_tickets"`
	AvgResolutionHours  float64   `json:"avg_resolution_hours"`
}

// ZendeskMetricsService calculates Zendesk-derived support metrics.
type ZendeskMetricsService struct{ tickets *repository.ZendeskTicketRepository }

// NewZendeskMetricsService creates a ZendeskMetricsService.
func NewZendeskMetricsService(tickets *repository.ZendeskTicketRepository) *ZendeskMetricsService {
	return &ZendeskMetricsService{tickets: tickets}
}

// GetCustomerMetrics returns 90-day Zendesk metrics for one customer.
func (s *ZendeskMetricsService) GetCustomerMetrics(ctx context.Context, customerID uuid.UUID) (*ZendeskMetrics, error) {
	since := time.Now().AddDate(0, 0, -90)
	open, solved, err := s.tickets.CountByCustomerAndStatus(ctx, customerID, since)
	if err != nil {
		return nil, err
	}
	avg, err := s.tickets.AvgResolutionHours(ctx, customerID, since)
	if err != nil {
		return nil, err
	}
	return &ZendeskMetrics{CustomerID: customerID, OpenTickets: open, SolvedTickets: solved, TotalTickets: open + solved, AvgResolutionHours: avg}, nil
}
