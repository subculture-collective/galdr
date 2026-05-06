package service

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/onnwee/pulse-score/internal/repository"
)

const (
	playbookThresholdDirectionDropsBelow = "drops_below"
	playbookThresholdDirectionRisesAbove = "rises_above"
	playbookThresholdDefaultCooldown     = 7 * 24 * time.Hour
)

type thresholdPlaybookStore interface {
	ListEnabledByTrigger(ctx context.Context, orgID uuid.UUID, triggerType string) ([]*repository.Playbook, error)
}

type thresholdExecutionStore interface {
	HasRecentCustomerExecution(ctx context.Context, playbookID, customerID uuid.UUID, since time.Time) (bool, error)
	Create(ctx context.Context, execution *repository.PlaybookExecution) error
}

// PlaybookThresholdTriggerConfig configures score-threshold trigger evaluation.
type PlaybookThresholdTriggerConfig struct {
	Playbooks  thresholdPlaybookStore
	Executions thresholdExecutionStore
	Now        func() time.Time
}

// PlaybookThresholdTriggerService records playbook executions for score threshold crossings.
type PlaybookThresholdTriggerService struct {
	playbooks  thresholdPlaybookStore
	executions thresholdExecutionStore
	now        func() time.Time
}

// NewPlaybookThresholdTriggerService creates a score-threshold playbook trigger service.
func NewPlaybookThresholdTriggerService(cfg PlaybookThresholdTriggerConfig) *PlaybookThresholdTriggerService {
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	return &PlaybookThresholdTriggerService{playbooks: cfg.Playbooks, executions: cfg.Executions, now: now}
}

// EvaluateScoreChange records pending executions for playbooks whose threshold was crossed.
func (s *PlaybookThresholdTriggerService) EvaluateScoreChange(ctx context.Context, previous, current *repository.HealthScore) error {
	if previous == nil || current == nil || s == nil || s.playbooks == nil || s.executions == nil {
		return nil
	}
	if previous.OrgID != current.OrgID || previous.CustomerID != current.CustomerID {
		return nil
	}

	playbooks, err := s.playbooks.ListEnabledByTrigger(ctx, current.OrgID, repository.PlaybookTriggerScoreThreshold)
	if err != nil {
		return fmt.Errorf("list score threshold playbooks: %w", err)
	}

	for _, playbook := range playbooks {
		cfg, ok := parsePlaybookThresholdConfig(playbook.TriggerConfig)
		if !ok || !thresholdCrossed(previous.OverallScore, current.OverallScore, cfg) {
			continue
		}

		recent, err := s.executions.HasRecentCustomerExecution(ctx, playbook.ID, current.CustomerID, s.now().Add(-cfg.cooldown))
		if err != nil {
			return fmt.Errorf("check playbook threshold cooldown: %w", err)
		}
		if recent {
			continue
		}

		customerID := current.CustomerID
		execution := &repository.PlaybookExecution{
			PlaybookID: playbook.ID,
			CustomerID: &customerID,
			Status:     repository.PlaybookExecutionPending,
			Result: map[string]any{
				"trigger":        repository.PlaybookTriggerScoreThreshold,
				"direction":      cfg.direction,
				"threshold":      cfg.threshold,
				"previous_score": previous.OverallScore,
				"current_score":  current.OverallScore,
			},
		}
		if err := s.executions.Create(ctx, execution); err != nil {
			return fmt.Errorf("create playbook threshold execution: %w", err)
		}
	}

	return nil
}

type playbookThresholdConfig struct {
	threshold int
	direction string
	cooldown  time.Duration
}

func parsePlaybookThresholdConfig(values map[string]any) (playbookThresholdConfig, bool) {
	threshold, ok := intConfigValue(values, "threshold")
	if !ok || threshold < 0 || threshold > 100 {
		return playbookThresholdConfig{}, false
	}
	direction := stringConfigValue(values, "direction")
	if direction != playbookThresholdDirectionDropsBelow && direction != playbookThresholdDirectionRisesAbove {
		return playbookThresholdConfig{}, false
	}
	cfg := playbookThresholdConfig{threshold: threshold, direction: direction, cooldown: playbookThresholdDefaultCooldown}
	if hours, ok := intConfigValue(values, "cooldown_hours"); ok && hours > 0 {
		cfg.cooldown = time.Duration(hours) * time.Hour
	}
	return cfg, true
}

func thresholdCrossed(previous, current int, cfg playbookThresholdConfig) bool {
	switch cfg.direction {
	case playbookThresholdDirectionDropsBelow:
		return previous >= cfg.threshold && current < cfg.threshold
	case playbookThresholdDirectionRisesAbove:
		return previous <= cfg.threshold && current > cfg.threshold
	default:
		return false
	}
}

func intConfigValue(values map[string]any, key string) (int, bool) {
	if values == nil {
		return 0, false
	}
	switch value := values[key].(type) {
	case int:
		return value, true
	case int32:
		return int(value), true
	case int64:
		return int(value), true
	case float64:
		if math.Trunc(value) == value {
			return int(value), true
		}
	case string:
		parsed, err := strconv.Atoi(value)
		return parsed, err == nil
	}
	return 0, false
}
