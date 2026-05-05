package ml

import (
	"encoding/json"
	"math"
	"strconv"
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
	eventLogin               = "login"
	eventFeatureUse          = "feature_use"
	eventAPICall             = "api_call"
	eventTicketOpened        = "ticket.opened"
	eventTicketResolved      = "ticket.resolved"
	eventConversationCreated = "conversation_created"
	eventConversationOpen    = "conversation_open"
	eventConversationClosed  = "conversation_closed"
	eventMRRChanged          = "mrr.changed"
)

const (
	scoreTrajectoryWindowDays = 30
	paymentWindowDays         = 90
	ticketWindowDays          = 90
	supportWindowDays         = 30
	usageWindowDays           = 30
	mrrWindowDays             = 90
	engagementWindowDays      = 30
	activityMaxDays           = 40
	contractMaxDays           = 365
)

var featureNames = [...]string{
	FeatureScoreTrajectorySlope30d,
	FeaturePaymentFailureFrequency90d,
	FeaturePaymentSuccessRate90d,
	FeatureSupportTicketTrend,
	FeatureUnresolvedTicketRatio,
	FeatureUsageFrequencyChange30d,
	FeatureMRRChangeRate,
	FeatureDaysSinceLastActivity,
	FeatureEngagementEventsPerDay,
	FeatureContractTenure,
	FeatureCurrentHealthScore,
}

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

// FeatureNames returns the stable model-input feature order.
func FeatureNames() []string {
	return append([]string(nil), featureNames[:]...)
}

// ValuesInOrder returns feature values in the stable model-input order.
func (v FeatureVector) ValuesInOrder() []float64 {
	values := make([]float64, len(featureNames))
	for i, name := range featureNames {
		values[i] = v.Values[name]
	}
	return values
}

// ExtractCustomerFeatures converts customer data into normalized churn-model inputs.
func ExtractCustomerFeatures(input FeatureInput) FeatureVector {
	now := input.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}

	vector := FeatureVector{
		Values:       make(map[string]float64, len(featureNames)),
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
		if score == nil || score.CalculatedAt.Before(daysAgo(now, scoreTrajectoryWindowDays)) || score.CalculatedAt.After(now) {
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
	pointsPer30Days := float64(latest.OverallScore-oldest.OverallScore) / days * scoreTrajectoryWindowDays
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
	start := daysAgo(now, paymentWindowDays)
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
	recentStart := daysAgo(now, supportWindowDays)
	previousStart := daysAgo(now, supportWindowDays*2)
	recent := countSupportOpenedEvents(events, recentStart, now)
	previous := countSupportOpenedEvents(events, previousStart, recentStart)
	if recent > previous {
		return 1
	}
	if recent < previous {
		return 0
	}
	return 0.5
}

func unresolvedTicketRatio(events []*repository.CustomerEvent, now time.Time) float64 {
	opened := countSupportOpenedEvents(events, daysAgo(now, ticketWindowDays), now)
	if opened == 0 {
		return 0
	}
	resolved := countSupportResolvedEvents(events, daysAgo(now, ticketWindowDays), now)
	unresolved := opened - resolved
	if unresolved < 0 {
		unresolved = 0
	}
	return clamp01(float64(unresolved) / float64(opened))
}

func usageFrequencyChange(events []*repository.CustomerEvent, now time.Time) float64 {
	recentStart := daysAgo(now, usageWindowDays)
	previousStart := daysAgo(now, usageWindowDays*2)
	recent := countUsageEvents(events, recentStart, now)
	previous := countUsageEvents(events, previousStart, recentStart)
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
	start := daysAgo(now, mrrWindowDays)
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
	return NormalizeMinMax(now.Sub(*lastActivity).Hours()/24, 0, activityMaxDays)
}

func engagementEventsPerDay(events []*repository.CustomerEvent, now time.Time) float64 {
	count := countAllEvents(events, daysAgo(now, engagementWindowDays), now)
	return NormalizeMinMax(float64(count)/engagementWindowDays, 0, 10)
}

func contractTenure(customer *repository.Customer, now time.Time) float64 {
	if customer == nil || customer.FirstSeenAt == nil {
		return 0
	}
	return NormalizeMinMax(now.Sub(*customer.FirstSeenAt).Hours()/24, 0, contractMaxDays)
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
	return countMatchingEvents(events, start, end, func(event *repository.CustomerEvent) bool {
		return isUsageEvent(event.EventType)
	})
}

func countSupportOpenedEvents(events []*repository.CustomerEvent, start, end time.Time) int {
	return countEventsMatching(events, isSupportOpenedEvent, start, end)
}

func countSupportResolvedEvents(events []*repository.CustomerEvent, start, end time.Time) int {
	return countEventsMatching(events, isSupportResolvedEvent, start, end)
}

func countEventsMatching(events []*repository.CustomerEvent, matches func(string) bool, start, end time.Time) int {
	count := 0
	for _, event := range events {
		if event == nil || !matches(event.EventType) || !occurredInHalfOpenWindow(event.OccurredAt, start, end) {
			continue
		}
		count++
	}
	return count
}
}

func countAllEvents(events []*repository.CustomerEvent, start, end time.Time) int {
	return countMatchingEvents(events, start, end, func(event *repository.CustomerEvent) bool {
		return true
	})
}

func countMatchingEvents(events []*repository.CustomerEvent, start, end time.Time, matches func(*repository.CustomerEvent) bool) int {
	count := 0
	for _, event := range events {
		if event == nil || !occurredInHalfOpenWindow(event.OccurredAt, start, end) || !matches(event) {
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

func isSupportOpenedEvent(eventType string) bool {
	switch eventType {
	case eventTicketOpened, eventConversationCreated, eventConversationOpen:
		return true
	default:
		return false
	}
}

func isSupportResolvedEvent(eventType string) bool {
	switch eventType {
	case eventTicketResolved, eventConversationClosed:
		return true
	default:
		return false
	}
}

func occurredInHalfOpenWindow(occurredAt, start, end time.Time) bool {
	return !occurredAt.Before(start) && occurredAt.Before(end)
}

func daysAgo(now time.Time, days int) time.Time {
	return now.AddDate(0, 0, -days)
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
	case json.Number:
		return parseFloatOrZero(string(value))
	case string:
		return parseFloatOrZero(value)
	default:
		return 0
	}
}

func parseFloatOrZero(value string) float64 {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return parsed
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
