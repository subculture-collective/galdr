package service

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"strings"

	"github.com/google/uuid"
	"github.com/onnwee/pulse-score/internal/repository"
)

const (
	playbookCustomerEventConfigEventType  = "event_type"
	playbookCustomerEventConfigEventTypes = "event_types"
	playbookCustomerEventConfigConditions = "conditions"

	conditionOperatorEquals       = "equals"
	conditionOperatorNotEquals    = "not_equals"
	conditionOperatorGreaterThan  = "greater_than"
	conditionOperatorGreaterEqual = "greater_than_or_equal"
	conditionOperatorLessThan     = "less_than"
	conditionOperatorLessEqual    = "less_than_or_equal"
	conditionOperatorContains     = "contains"
)

type customerEventPlaybookStore interface {
	ListEnabledByTrigger(ctx context.Context, orgID uuid.UUID, triggerType string) ([]*repository.Playbook, error)
}

type customerEventExecutionStore interface {
	Create(ctx context.Context, execution *repository.PlaybookExecution) error
}

// PlaybookCustomerEventTriggerService records playbook executions for matching customer events.
type PlaybookCustomerEventTriggerService struct {
	playbooks  customerEventPlaybookStore
	executions customerEventExecutionStore
}

// NewPlaybookCustomerEventTriggerService creates a customer-event playbook trigger service.
func NewPlaybookCustomerEventTriggerService(playbooks customerEventPlaybookStore, executions customerEventExecutionStore) *PlaybookCustomerEventTriggerService {
	return &PlaybookCustomerEventTriggerService{playbooks: playbooks, executions: executions}
}

// EvaluateCustomerEvent records pending executions for enabled playbooks matching the event.
func (s *PlaybookCustomerEventTriggerService) EvaluateCustomerEvent(ctx context.Context, event *repository.CustomerEvent) error {
	if s == nil || s.playbooks == nil || s.executions == nil || event == nil {
		return nil
	}
	if strings.TrimSpace(event.EventType) == "" || event.OrgID == uuid.Nil || event.CustomerID == uuid.Nil {
		return nil
	}

	playbooks, err := s.playbooks.ListEnabledByTrigger(ctx, event.OrgID, repository.PlaybookTriggerCustomerEvent)
	if err != nil {
		return fmt.Errorf("list customer-event playbooks: %w", err)
	}

	for _, playbook := range playbooks {
		cfg, ok := parsePlaybookCustomerEventConfig(playbook.TriggerConfig)
		if !ok || !cfg.matches(event) {
			continue
		}

		execution := newCustomerEventExecution(playbook.ID, event)
		if err := s.executions.Create(ctx, execution); err != nil {
			return fmt.Errorf("create customer-event playbook execution: %w", err)
		}
	}

	return nil
}

type playbookCustomerEventConfig struct {
	eventTypes map[string]struct{}
	conditions []playbookCustomerEventCondition
}

type playbookCustomerEventCondition struct {
	field    string
	operator string
	value    any
}

func parsePlaybookCustomerEventConfig(values map[string]any) (playbookCustomerEventConfig, bool) {
	eventTypes := stringSliceConfigValue(values, playbookCustomerEventConfigEventTypes)
	if legacy := strings.TrimSpace(stringConfigValue(values, playbookCustomerEventConfigEventType)); legacy != "" {
		eventTypes = append(eventTypes, legacy)
	}
	if len(eventTypes) == 0 {
		return playbookCustomerEventConfig{}, false
	}

	cfg := playbookCustomerEventConfig{eventTypes: make(map[string]struct{}, len(eventTypes))}
	for _, eventType := range eventTypes {
		trimmed := strings.TrimSpace(eventType)
		if trimmed != "" {
			cfg.eventTypes[trimmed] = struct{}{}
		}
	}
	if len(cfg.eventTypes) == 0 {
		return playbookCustomerEventConfig{}, false
	}

	cfg.conditions = parseCustomerEventConditions(values[playbookCustomerEventConfigConditions])
	return cfg, true
}

func (cfg playbookCustomerEventConfig) matches(event *repository.CustomerEvent) bool {
	if _, ok := cfg.eventTypes[event.EventType]; !ok {
		return false
	}
	for _, condition := range cfg.conditions {
		if !condition.matches(event) {
			return false
		}
	}
	return true
}

func (c playbookCustomerEventCondition) matches(event *repository.CustomerEvent) bool {
	actual, ok := eventFieldValue(event, c.field)
	if !ok {
		return false
	}
	switch c.operator {
	case conditionOperatorEquals:
		return valuesEqual(actual, c.value)
	case conditionOperatorNotEquals:
		return !valuesEqual(actual, c.value)
	case conditionOperatorGreaterThan:
		return compareNumbers(actual, c.value, func(a, b float64) bool { return a > b })
	case conditionOperatorGreaterEqual:
		return compareNumbers(actual, c.value, func(a, b float64) bool { return a >= b })
	case conditionOperatorLessThan:
		return compareNumbers(actual, c.value, func(a, b float64) bool { return a < b })
	case conditionOperatorLessEqual:
		return compareNumbers(actual, c.value, func(a, b float64) bool { return a <= b })
	case conditionOperatorContains:
		return strings.Contains(fmt.Sprint(actual), fmt.Sprint(c.value))
	default:
		return false
	}
}

func newCustomerEventExecution(playbookID uuid.UUID, event *repository.CustomerEvent) *repository.PlaybookExecution {
	customerID := event.CustomerID
	return &repository.PlaybookExecution{
		PlaybookID: playbookID,
		CustomerID: &customerID,
		Status:     repository.PlaybookExecutionPending,
		Result: map[string]any{
			"trigger":           repository.PlaybookTriggerCustomerEvent,
			"customer_event_id": event.ID.String(),
			"event_type":        event.EventType,
			"source":            event.Source,
			"external_event_id": event.ExternalEventID,
			"event_data":        event.Data,
		},
	}
}

func parseCustomerEventConditions(raw any) []playbookCustomerEventCondition {
	items, ok := raw.([]any)
	if !ok {
		if typed, typedOK := raw.([]map[string]any); typedOK {
			items = make([]any, 0, len(typed))
			for _, item := range typed {
				items = append(items, item)
			}
		} else {
			return nil
		}
	}
	conditions := make([]playbookCustomerEventCondition, 0, len(items))
	for _, item := range items {
		values, ok := item.(map[string]any)
		if !ok {
			continue
		}
		condition := playbookCustomerEventCondition{
			field:    strings.TrimSpace(stringConfigValue(values, "field")),
			operator: strings.TrimSpace(stringConfigValue(values, "operator")),
			value:    values["value"],
		}
		if condition.field != "" && condition.operator != "" {
			conditions = append(conditions, condition)
		}
	}
	return conditions
}

func eventFieldValue(event *repository.CustomerEvent, field string) (any, bool) {
	switch field {
	case "event_type":
		return event.EventType, true
	case "source":
		return event.Source, true
	case "external_event_id":
		return event.ExternalEventID, true
	case "customer_id":
		return event.CustomerID.String(), true
	}
	return nestedMapValue(event.Data, strings.Split(field, "."))
}

func nestedMapValue(values map[string]any, path []string) (any, bool) {
	if len(path) == 0 || values == nil {
		return nil, false
	}
	current, ok := values[path[0]]
	if !ok || len(path) == 1 {
		return current, ok
	}
	nested, ok := current.(map[string]any)
	if !ok {
		return nil, false
	}
	return nestedMapValue(nested, path[1:])
}

func stringSliceConfigValue(values map[string]any, key string) []string {
	if values == nil {
		return nil
	}
	switch raw := values[key].(type) {
	case []string:
		return raw
	case []any:
		items := make([]string, 0, len(raw))
		for _, item := range raw {
			if value := strings.TrimSpace(fmt.Sprint(item)); value != "" {
				items = append(items, value)
			}
		}
		return items
	default:
		return nil
	}
}

func valuesEqual(left, right any) bool {
	if leftNumber, ok := anyToFloat64(left); ok {
		if rightNumber, ok := anyToFloat64(right); ok {
			return math.Abs(leftNumber-rightNumber) < 0.000001
		}
	}
	return reflect.DeepEqual(left, right) || fmt.Sprint(left) == fmt.Sprint(right)
}

func compareNumbers(left, right any, compare func(float64, float64) bool) bool {
	leftNumber, leftOK := anyToFloat64(left)
	rightNumber, rightOK := anyToFloat64(right)
	return leftOK && rightOK && compare(leftNumber, rightNumber)
}

func anyToFloat64(value any) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	case jsonNumber:
		parsed, err := v.Float64()
		return parsed, err == nil
	default:
		return 0, false
	}
}
