package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/onnwee/pulse-score/internal/repository"
)

const maxPlaybookNameLength = 255

type playbookRepository interface {
	List(ctx context.Context, orgID uuid.UUID) ([]*repository.Playbook, error)
	GetByID(ctx context.Context, id, orgID uuid.UUID) (*repository.Playbook, error)
	Create(ctx context.Context, p *repository.Playbook) error
	Update(ctx context.Context, p *repository.Playbook) error
	Delete(ctx context.Context, id, orgID uuid.UUID) error
}

type playbookActionRepository interface {
	ListByPlaybook(ctx context.Context, playbookID uuid.UUID) ([]*repository.PlaybookAction, error)
	Create(ctx context.Context, a *repository.PlaybookAction) error
	DeleteByPlaybook(ctx context.Context, playbookID uuid.UUID) error
}

// PlaybookService manages playbook CRUD and nested actions.
type PlaybookService struct {
	playbooks playbookRepository
	actions   playbookActionRepository
}

type PlaybookActionRequest struct {
	ActionType   string         `json:"action_type"`
	ActionConfig map[string]any `json:"action_config"`
}

type CreatePlaybookRequest struct {
	Name          string                  `json:"name"`
	Description   string                  `json:"description"`
	TriggerType   string                  `json:"trigger_type"`
	TriggerConfig map[string]any          `json:"trigger_config"`
	Actions       []PlaybookActionRequest `json:"actions"`
}

type UpdatePlaybookRequest struct {
	Name          string                  `json:"name"`
	Description   string                  `json:"description"`
	TriggerType   string                  `json:"trigger_type"`
	TriggerConfig map[string]any          `json:"trigger_config"`
	Actions       []PlaybookActionRequest `json:"actions"`
}

type SetPlaybookEnabledRequest struct {
	Enabled bool `json:"enabled"`
}

type PlaybookResponse struct {
	repository.Playbook
	Actions []*repository.PlaybookAction `json:"actions"`
}

func NewPlaybookService(playbooks playbookRepository, actions playbookActionRepository) *PlaybookService {
	return &PlaybookService{playbooks: playbooks, actions: actions}
}

func (s *PlaybookService) List(ctx context.Context, orgID uuid.UUID) ([]*PlaybookResponse, error) {
	playbooks, err := s.playbooks.List(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("list playbooks: %w", err)
	}
	responses := make([]*PlaybookResponse, 0, len(playbooks))
	for _, playbook := range playbooks {
		response, err := s.withActions(ctx, playbook)
		if err != nil {
			return nil, err
		}
		responses = append(responses, response)
	}
	return responses, nil
}

func (s *PlaybookService) GetByID(ctx context.Context, id, orgID uuid.UUID) (*PlaybookResponse, error) {
	playbook, err := s.playbooks.GetByID(ctx, id, orgID)
	if err != nil {
		return nil, fmt.Errorf("get playbook: %w", err)
	}
	if playbook == nil {
		return nil, playbookNotFound()
	}
	return s.withActions(ctx, playbook)
}

func (s *PlaybookService) Create(ctx context.Context, orgID uuid.UUID, req CreatePlaybookRequest) (*PlaybookResponse, error) {
	name, err := validatePlaybookFields(req.Name, req.TriggerType, req.TriggerConfig, req.Actions)
	if err != nil {
		return nil, err
	}

	playbook := &repository.Playbook{
		OrgID:         orgID,
		Name:          name,
		Description:   strings.TrimSpace(req.Description),
		Enabled:       true,
		TriggerType:   req.TriggerType,
		TriggerConfig: nonNilMap(req.TriggerConfig),
	}
	if err := s.playbooks.Create(ctx, playbook); err != nil {
		return nil, fmt.Errorf("create playbook: %w", err)
	}
	if err := s.createActions(ctx, playbook.ID, req.Actions); err != nil {
		return nil, err
	}
	return s.withActions(ctx, playbook)
}

func (s *PlaybookService) Update(ctx context.Context, id, orgID uuid.UUID, req UpdatePlaybookRequest) (*PlaybookResponse, error) {
	name, err := validatePlaybookFields(req.Name, req.TriggerType, req.TriggerConfig, req.Actions)
	if err != nil {
		return nil, err
	}

	playbook, err := s.playbooks.GetByID(ctx, id, orgID)
	if err != nil {
		return nil, fmt.Errorf("get playbook: %w", err)
	}
	if playbook == nil {
		return nil, playbookNotFound()
	}
	playbook.Name = name
	playbook.Description = strings.TrimSpace(req.Description)
	playbook.TriggerType = req.TriggerType
	playbook.TriggerConfig = nonNilMap(req.TriggerConfig)
	if err := s.playbooks.Update(ctx, playbook); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, playbookNotFound()
		}
		return nil, fmt.Errorf("update playbook: %w", err)
	}
	if err := s.actions.DeleteByPlaybook(ctx, id); err != nil {
		return nil, fmt.Errorf("delete playbook actions: %w", err)
	}
	if err := s.createActions(ctx, id, req.Actions); err != nil {
		return nil, err
	}
	return s.withActions(ctx, playbook)
}

func (s *PlaybookService) Delete(ctx context.Context, id, orgID uuid.UUID) error {
	if err := s.playbooks.Delete(ctx, id, orgID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return playbookNotFound()
		}
		return fmt.Errorf("delete playbook: %w", err)
	}
	return nil
}

func (s *PlaybookService) SetEnabled(ctx context.Context, id, orgID uuid.UUID, req SetPlaybookEnabledRequest) (*PlaybookResponse, error) {
	playbook, err := s.playbooks.GetByID(ctx, id, orgID)
	if err != nil {
		return nil, fmt.Errorf("get playbook: %w", err)
	}
	if playbook == nil {
		return nil, playbookNotFound()
	}
	playbook.Enabled = req.Enabled
	if err := s.playbooks.Update(ctx, playbook); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, playbookNotFound()
		}
		return nil, fmt.Errorf("set playbook enabled: %w", err)
	}
	return s.withActions(ctx, playbook)
}

func (s *PlaybookService) withActions(ctx context.Context, playbook *repository.Playbook) (*PlaybookResponse, error) {
	actions, err := s.actions.ListByPlaybook(ctx, playbook.ID)
	if err != nil {
		return nil, fmt.Errorf("list playbook actions: %w", err)
	}
	return &PlaybookResponse{Playbook: *playbook, Actions: actions}, nil
}

func (s *PlaybookService) createActions(ctx context.Context, playbookID uuid.UUID, actions []PlaybookActionRequest) error {
	for i, req := range actions {
		action := &repository.PlaybookAction{
			PlaybookID:    playbookID,
			ActionType:    req.ActionType,
			ActionConfig:  nonNilMap(req.ActionConfig),
			OrderIndex:    i,
		}
		if err := s.actions.Create(ctx, action); err != nil {
			return fmt.Errorf("create playbook action: %w", err)
		}
	}
	return nil
}

func validatePlaybookFields(name, triggerType string, triggerConfig map[string]any, actions []PlaybookActionRequest) (string, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "", &ValidationError{Field: "name", Message: "name is required"}
	}
	if len(trimmed) > maxPlaybookNameLength {
		return "", &ValidationError{Field: "name", Message: "name must be 255 characters or less"}
	}
	if err := validatePlaybookTrigger(triggerType, triggerConfig); err != nil {
		return "", err
	}
	for i, action := range actions {
		if err := validatePlaybookAction(i, action); err != nil {
			return "", err
		}
	}
	return trimmed, nil
}

func validatePlaybookTrigger(triggerType string, config map[string]any) error {
	switch triggerType {
	case repository.PlaybookTriggerScoreThreshold:
		threshold, ok := numberConfigValue(config, "threshold")
		if !ok || threshold < 0 || threshold > 100 {
			return &ValidationError{Field: "trigger_config.threshold", Message: "threshold must be between 0 and 100"}
		}
		direction := stringConfigValue(config, "direction")
		if direction != "" && direction != "above" && direction != "below" {
			return &ValidationError{Field: "trigger_config.direction", Message: "direction must be above or below"}
		}
	case repository.PlaybookTriggerCustomerEvent:
		if strings.TrimSpace(stringConfigValue(config, "event_type")) == "" {
			return &ValidationError{Field: "trigger_config.event_type", Message: "event_type is required"}
		}
	case repository.PlaybookTriggerSchedule:
		if _, ok := numberConfigValue(config, "interval_minutes"); !ok && strings.TrimSpace(stringConfigValue(config, "cron")) == "" {
			return &ValidationError{Field: "trigger_config", Message: "schedule requires interval_minutes or cron"}
		}
	default:
		return &ValidationError{Field: "trigger_type", Message: "unsupported trigger type"}
	}
	return nil
}

func validatePlaybookAction(index int, action PlaybookActionRequest) error {
	field := fmt.Sprintf("actions[%d].action_config", index)
	switch action.ActionType {
	case repository.PlaybookActionSendEmail:
		if strings.TrimSpace(stringConfigValue(action.ActionConfig, "subject")) == "" {
			return &ValidationError{Field: field + ".subject", Message: "subject is required"}
		}
		if strings.TrimSpace(stringConfigValue(action.ActionConfig, "template")) == "" && strings.TrimSpace(stringConfigValue(action.ActionConfig, "body")) == "" {
			return &ValidationError{Field: field, Message: "template or body is required"}
		}
	case repository.PlaybookActionInternalAlert:
		if strings.TrimSpace(stringConfigValue(action.ActionConfig, "message")) == "" {
			return &ValidationError{Field: field + ".message", Message: "message is required"}
		}
	case repository.PlaybookActionTagCustomer:
		if strings.TrimSpace(stringConfigValue(action.ActionConfig, "tag")) == "" {
			return &ValidationError{Field: field + ".tag", Message: "tag is required"}
		}
	case repository.PlaybookActionWebhook:
		cfg, err := parseWebhookActionConfig(action.ActionConfig)
		if err != nil {
			return &ValidationError{Field: field, Message: err.Error()}
		}
		if err := validateWebhookURL(cfg.URL); err != nil {
			return &ValidationError{Field: field + ".url", Message: err.Error()}
		}
	default:
		return &ValidationError{Field: fmt.Sprintf("actions[%d].action_type", index), Message: "unsupported action type"}
	}
	return nil
}

func numberConfigValue(values map[string]any, key string) (float64, bool) {
	if values == nil {
		return 0, false
	}
	switch value := values[key].(type) {
	case float64:
		return value, true
	case float32:
		return float64(value), true
	case int:
		return float64(value), true
	case int64:
		return float64(value), true
	case jsonNumber:
		parsed, err := value.Float64()
		return parsed, err == nil
	default:
		return 0, false
	}
}

type jsonNumber interface {
	Float64() (float64, error)
}

func nonNilMap(values map[string]any) map[string]any {
	if values == nil {
		return map[string]any{}
	}
	return values
}

func playbookNotFound() *NotFoundError {
	return &NotFoundError{Resource: "playbook", Message: "playbook not found"}
}
