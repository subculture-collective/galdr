package ml

import (
	"math"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

const (
	FeatureScoreTrajectorySlope30d   = "score_trajectory_slope_30d"
	FeaturePaymentFailureFrequency90d = "payment_failure_frequency_90d"
	FeaturePaymentSuccessRate90d      = "payment_success_rate_90d"
	FeatureSupportTicketTrend         = "support_ticket_trend"
	FeatureUnresolvedTicketRatio      = "unresolved_ticket_ratio"
	FeatureUsageFrequencyChange30d    = "usage_frequency_change_30d"
	FeatureMRRChangeRate              = "mrr_change_rate"
	FeatureDaysSinceLastActivity      = "days_since_last_activity"
	FeatureEngagementEventsPerDay     = "engagement_events_per_day"
	FeatureContractTenure             = "contract_tenure"
	FeatureCurrentHealthScore         = "current_health_score"
)

const (
	featureCount = 11

	eventLogin          = "login"
	eventFeatureUse     = "feature_use"
	eventAPICall        = "api_call"
	eventTicketOpened   = "ticket.opened"
	eventTicketResolved = "ticket.resolved"
	eventMRRChanged     = "mrr.changed"
)

var usageEventTypes = map[string]struct{}{
	eventLogin:      {},
	eventFeatureUse: {},
	eventAPICall:    {},
}

// FeatureInput is the raw customer data used for churn model feature extraction.
type FeatureInput struct {
	Now                time.Time
	Customer           *repository.Customer
	HealthScoreHistory []*repository.HealthScore
	Payments           []*repository.StripePayment
	Events             []*repository.CustomerEvent
}

// FeatureVector is the normalized model input for one customer.
type FeatureVector struct {
	OrgID        uuid.UUID
	CustomerID   uuid.UUID
	Values       map[string]float64
	CalculatedAt time.Time
}

// ExtractCustomerFeatures converts customer data into normalized churn-model inputs.
func ExtractCustomerFeatures(input FeatureInput) FeatureVector {
	now := input.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	vector := FeatureVector{
		Values:       make(map[string]float64, featureCount),
		CalculatedAt: now,
	}
	if input.Customer != nil {
		vector.OrgID = input.Customer.OrgID
		vector.CustomerID = input.Customer.ID
	}

	vector.Values[FeatureScoreTrajectorySlope30d] = scoreTrajectorySlope(input.HealthScoreHistory, now)
	vector.Values[FeaturePaymentFailureFrequency90d] = paymentFailureFrequency(input.Payments, now)
	vector.Values[FeaturePaymentSuccessRate90d] = paymentSuccessRate(input.Payments, now)
	vector.Values[FeatureSupportTicketTrend] = supportTicketTrend(input.Events, now)
	vector.Values[FeatureUnresolvedTicketRatio] = unresolvedTicketRatio(input.Events, now)
	vector.Values[FeatureUsageFrequencyChange30d] = usageFrequencyChange(input.Events, now)
	vector.Values[FeatureMRRChangeRate] = mrrChangeRate(input.Events, now)
	vector.Values[FeatureDaysSinceLastActivity] = daysSinceLastActivity(input.Customer, input.Events, now)
	vector.Values[FeatureEngagementEventsPerDay] = engagementEventsPerDay(input.Events, now)
	vector.Values[FeatureContractTenure] = contractTenure(input.Customer, now)
	vector.Values[FeatureCurrentHealthScore] = currentHealthScore(input.HealthScoreHistory)

	return vector
}

// NormalizeMinMax scales value into [0,1] and clamps outliers.
func NormalizeMinMax(value, min, max float64) float64 {
	if max <= min {
		return 0
	}
	return clamp01((value - min) / (max - min))
}

func scoreTrajectorySlope(history []*repository.HealthScore, now time.Time) float64 {
	var latest, oldest *repository.HealthScore
	for _, score := range history {
		if score == nil || score.CalculatedAt.Before(now.AddDate(0, 0, -30)) || score.CalculatedAt.After(now) {
			continue
		}
		if latest == nil || score.CalculatedAt.After(latest.CalculatedAt) {
			latest = score
		}
		if oldest == nil || score.CalculatedAt.Before(oldest.CalculatedAt) {
			oldest = score
		}
	}
	if latest == nil || oldest == nil || latest.CalculatedAt.Equal(oldest.CalculatedAt) {
		return 0.5
	}
	days := latest.CalculatedAt.Sub(oldest.CalculatedAt).Hours() / 24
	if days <= 0 {
		return 0.5
	}
	pointsPer30Days := float64(latest.OverallScore-oldest.OverallScore) / days * 30
	return NormalizeMinMax(pointsPer30Days, -50, 50)
}

func paymentFailureFrequency(payments []*repository.StripePayment, now time.Time) float64 {
	total, failed := paymentCounts90d(payments, now)
	if total == 0 {
		return 0
	}
	return clamp01(float64(failed) / float64(total))
}

func paymentSuccessRate(payments []*repository.StripePayment, now time.Time) float64 {
	total, failed := paymentCounts90d(payments, now)
	if total == 0 {
		return 1
	}
	return clamp01(float64(total-failed) / float64(total))
}

func paymentCounts90d(payments []*repository.StripePayment, now time.Time) (int, int) {
	var total, failed int
	start := now.AddDate(0, 0, -90)
	for _, payment := range payments {
		if payment == nil {
			continue
		}
		at := payment.CreatedAt
		if payment.PaidAt != nil {
			at = *payment.PaidAt
		}
		if at.Before(start) || at.After(now) {
			continue
		}
		total++
		if payment.Status == "failed" {
			failed++
		}
	}
	return total, failed
}

func supportTicketTrend(events []*repository.CustomerEvent, now time.Time) float64 {
	recent := countEvents(events, eventTicketOpened, now.AddDate(0, 0, -30), now)
	previous := countEvents(events, eventTicketOpened, now.AddDate(0, 0, -60), now.AddDate(0, 0, -30))
	if recent > previous {
		return 1
	}
	if recent < previous {
		return 0
	}
	return 0.5
}

func unresolvedTicketRatio(events []*repository.CustomerEvent, now time.Time) float64 {
	opened := countEvents(events, eventTicketOpened, now.AddDate(0, 0, -90), now)
	if opened == 0 {
		return 0
	}
	resolved := countEvents(events, eventTicketResolved, now.AddDate(0, 0, -90), now)
	unresolved := opened - resolved
	if unresolved < 0 {
		unresolved = 0
	}
	return clamp01(float64(unresolved) / float64(opened))
}

func usageFrequencyChange(events []*repository.CustomerEvent, now time.Time) float64 {
	recent := countUsageEvents(events, now.AddDate(0, 0, -30), now)
	previous := countUsageEvents(events, now.AddDate(0, 0, -60), now.AddDate(0, 0, -30))
	if previous == 0 {
		if recent == 0 {
			return 0.5
		}
		return 1
	}
	change := (float64(recent) - float64(previous)) / float64(previous)
	return NormalizeMinMax(change, -1, 1)
}

func mrrChangeRate(events []*repository.CustomerEvent, now time.Time) float64 {
	var oldest, latest *repository.CustomerEvent
	start := now.AddDate(0, 0, -90)
	for _, event := range events {
		if event == nil || event.EventType != eventMRRChanged || event.OccurredAt.Before(start) || event.OccurredAt.After(now) {
			continue
		}
		if oldest == nil || event.OccurredAt.Before(oldest.OccurredAt) {
			oldest = event
		}
		if latest == nil || event.OccurredAt.After(latest.OccurredAt) {
			latest = event
		}
	}
	if oldest == nil || latest == nil {
		return 0.5
	}
	oldMRR := numberFromEvent(oldest, "old_mrr_cents")
	newMRR := numberFromEvent(latest, "new_mrr_cents")
	if oldMRR <= 0 {
		return 0.5
	}
	change := (newMRR - oldMRR) / oldMRR
	return NormalizeMinMax(change, -1, 1)
}

func daysSinceLastActivity(customer *repository.Customer, events []*repository.CustomerEvent, now time.Time) float64 {
	lastActivity := latestActivityAt(customer, events, now)
	if lastActivity == nil {
		return 1
	}
	return NormalizeMinMax(now.Sub(*lastActivity).Hours()/24, 0, 40)
}

func engagementEventsPerDay(events []*repository.CustomerEvent, now time.Time) float64 {
	count := countAllEvents(events, now.AddDate(0, 0, -30), now)
	return NormalizeMinMax(float64(count)/30, 0, 10)
}

func contractTenure(customer *repository.Customer, now time.Time) float64 {
	if customer == nil || customer.FirstSeenAt == nil {
		return 0
	}
	return NormalizeMinMax(now.Sub(*customer.FirstSeenAt).Hours()/24, 0, 365)
}

func currentHealthScore(history []*repository.HealthScore) float64 {
	var latest *repository.HealthScore
	for _, score := range history {
		if score == nil {
			continue
		}
		if latest == nil || score.CalculatedAt.After(latest.CalculatedAt) {
			latest = score
		}
	}
	if latest == nil {
		return 0.5
	}
	return NormalizeMinMax(float64(latest.OverallScore), 0, 100)
}

func latestActivityAt(customer *repository.Customer, events []*repository.CustomerEvent, now time.Time) *time.Time {
	var latest *time.Time
	if customer != nil && customer.LastSeenAt != nil && !customer.LastSeenAt.After(now) {
		latest = customer.LastSeenAt
	}
	for _, event := range events {
		if event == nil || event.OccurredAt.After(now) {
			continue
		}
		occurredAt := event.OccurredAt
		if latest == nil || occurredAt.After(*latest) {
			latest = &occurredAt
		}
	}
	return latest
}

func countUsageEvents(events []*repository.CustomerEvent, start, end time.Time) int {
	count := 0
	for _, event := range events {
		if event == nil || !isUsageEvent(event.EventType) || !occurredInHalfOpenWindow(event.OccurredAt, start, end) {
			continue
		}
		count++
	}
	return count
}

func countEvents(events []*repository.CustomerEvent, eventType string, start, end time.Time) int {
	count := 0
	for _, event := range events {
		if event == nil || event.EventType != eventType || !occurredInHalfOpenWindow(event.OccurredAt, start, end) {
			continue
		}
		count++
	}
	return count
}

func countAllEvents(events []*repository.CustomerEvent, start, end time.Time) int {
	count := 0
	for _, event := range events {
		if event == nil || !occurredInHalfOpenWindow(event.OccurredAt, start, end) {
			continue
		}
		count++
	}
	return count
}

func isUsageEvent(eventType string) bool {
	_, ok := usageEventTypes[eventType]
	return ok
}

func occurredInHalfOpenWindow(occurredAt, start, end time.Time) bool {
	return !occurredAt.Before(start) && occurredAt.Before(end)
}

func numberFromEvent(event *repository.CustomerEvent, key string) float64 {
	if event == nil || event.Data == nil {
		return 0
	}
	switch value := event.Data[key].(type) {
	case int:
		return float64(value)
	case int64:
		return float64(value)
	case float64:
		return value
	case float32:
		return float64(value)
	default:
		return 0
	}
}

func clamp01(value float64) float64 {
	if math.IsNaN(value) || value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}
