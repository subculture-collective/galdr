package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

const (
	// GenericWebhookEventSource marks customers and events created by the generic webhook receiver.
	GenericWebhookEventSource = "generic_webhook"

	genericWebhookCustomerEmailKey    = "customer_email"
	genericWebhookEventTypeKey        = "event_type"
	genericWebhookExternalEventIDKey  = "external_event_id"
	genericWebhookOccurredAtKey       = "occurred_at"
	genericWebhookDefaultCurrency     = "usd"
)

var genericWebhookReservedDataKeys = map[string]struct{}{
	genericWebhookCustomerEmailKey:   {},
	genericWebhookEventTypeKey:       {},
	genericWebhookExternalEventIDKey: {},
	genericWebhookOccurredAtKey:      {},
}

// GenericWebhookConfigRequest creates or updates generic webhook config.
type GenericWebhookConfigRequest struct {
	Name         string            `json:"name"`
	Secret       string            `json:"secret"`
	FieldMapping map[string]string `json:"field_mapping"`
	IsActive     *bool             `json:"is_active,omitempty"`
}

// GenericWebhookProcessResult summarizes a received webhook event.
type GenericWebhookProcessResult struct {
	EventID    uuid.UUID `json:"event_id"`
	CustomerID uuid.UUID `json:"customer_id"`
	EventType  string    `json:"event_type"`
}

// GenericWebhookTestRequest validates mapping without persisting an event.
type GenericWebhookTestRequest struct {
	Payload      map[string]any    `json:"payload"`
	FieldMapping map[string]string `json:"field_mapping"`
}

// GenericWebhookTestResult returns extracted mapping values.
type GenericWebhookTestResult struct {
	Mapped map[string]any `json:"mapped"`
}

type genericWebhookOrgStore interface {
	GetBySlug(ctx context.Context, slug string) (*repository.Organization, error)
}

type genericWebhookConfigStore interface {
	Create(ctx context.Context, c *repository.GenericWebhookConfig) error
	ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*repository.GenericWebhookConfig, error)
	GetByIDAndOrg(ctx context.Context, id, orgID uuid.UUID) (*repository.GenericWebhookConfig, error)
	GetActiveByIDAndOrg(ctx context.Context, id, orgID uuid.UUID) (*repository.GenericWebhookConfig, error)
	Update(ctx context.Context, c *repository.GenericWebhookConfig) error
	Delete(ctx context.Context, orgID, id uuid.UUID) error
}

type genericWebhookCustomerStore interface {
	GetByEmail(ctx context.Context, orgID uuid.UUID, email string) (*repository.Customer, error)
	UpsertByExternal(ctx context.Context, c *repository.Customer) error
}

type genericWebhookEventStore interface {
	Upsert(ctx context.Context, e *repository.CustomerEvent) error
}

// GenericWebhookService maps arbitrary signed JSON payloads to customer events.
type GenericWebhookService struct {
	orgs      genericWebhookOrgStore
	configs   genericWebhookConfigStore
	customers genericWebhookCustomerStore
	events    genericWebhookEventStore
}

// NewGenericWebhookService creates a GenericWebhookService.
func NewGenericWebhookService(orgs genericWebhookOrgStore, configs genericWebhookConfigStore, customers genericWebhookCustomerStore, events genericWebhookEventStore) *GenericWebhookService {
	return &GenericWebhookService{orgs: orgs, configs: configs, customers: customers, events: events}
}

// Create stores a webhook config.
func (s *GenericWebhookService) Create(ctx context.Context, orgID uuid.UUID, req GenericWebhookConfigRequest) (*repository.GenericWebhookConfig, error) {
	if err := validateGenericWebhookConfig(req); err != nil {
		return nil, err
	}
	cfg := newGenericWebhookConfig(orgID, req)
	if err := s.configs.Create(ctx, cfg); err != nil {
		return nil, fmt.Errorf("create generic webhook config: %w", err)
	}
	return cfg, nil
}

// List returns generic webhook configs for an org.
func (s *GenericWebhookService) List(ctx context.Context, orgID uuid.UUID) ([]*repository.GenericWebhookConfig, error) {
	configs, err := s.configs.ListByOrg(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("list generic webhook configs: %w", err)
	}
	return configs, nil
}

// Get returns one generic webhook config.
func (s *GenericWebhookService) Get(ctx context.Context, orgID, id uuid.UUID) (*repository.GenericWebhookConfig, error) {
	cfg, err := s.configs.GetByIDAndOrg(ctx, id, orgID)
	if err != nil {
		return nil, fmt.Errorf("get generic webhook config: %w", err)
	}
	if cfg == nil {
		return nil, &NotFoundError{Resource: "generic_webhook", Message: "generic webhook config not found"}
	}
	return cfg, nil
}

// Update replaces a generic webhook config.
func (s *GenericWebhookService) Update(ctx context.Context, orgID, id uuid.UUID, req GenericWebhookConfigRequest) (*repository.GenericWebhookConfig, error) {
	if err := validateGenericWebhookConfig(req); err != nil {
		return nil, err
	}
	cfg, err := s.Get(ctx, orgID, id)
	if err != nil {
		return nil, err
	}
	cfg.Name = strings.TrimSpace(req.Name)
	cfg.Secret = req.Secret
	cfg.FieldMapping = req.FieldMapping
	if req.IsActive != nil {
		cfg.IsActive = *req.IsActive
	}
	if err := s.configs.Update(ctx, cfg); err != nil {
		return nil, fmt.Errorf("update generic webhook config: %w", err)
	}
	return cfg, nil
}

// Delete removes a generic webhook config.
func (s *GenericWebhookService) Delete(ctx context.Context, orgID, id uuid.UUID) error {
	if err := s.configs.Delete(ctx, orgID, id); err != nil {
		return fmt.Errorf("delete generic webhook config: %w", err)
	}
	return nil
}

// Process verifies and persists one inbound generic webhook payload.
func (s *GenericWebhookService) Process(ctx context.Context, orgSlug string, webhookID uuid.UUID, payload []byte, signature string) (*GenericWebhookProcessResult, error) {
	org, err := s.orgs.GetBySlug(ctx, orgSlug)
	if err != nil {
		return nil, fmt.Errorf("get org by slug: %w", err)
	}
	if org == nil {
		return nil, &NotFoundError{Resource: "organization", Message: "organization not found"}
	}

	cfg, err := s.configs.GetActiveByIDAndOrg(ctx, webhookID, org.ID)
	if err != nil {
		return nil, fmt.Errorf("get generic webhook config: %w", err)
	}
	if cfg == nil {
		return nil, &NotFoundError{Resource: "generic_webhook", Message: "generic webhook config not found"}
	}
	if err := verifyGenericWebhookSignature(cfg.Secret, payload, signature); err != nil {
		return nil, err
	}

	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		return nil, &ValidationError{Field: "payload", Message: "payload must be a JSON object"}
	}
	mapped, err := applyGenericWebhookMapping(body, cfg.FieldMapping)
	if err != nil {
		return nil, err
	}

	customerEmail, eventType, err := requiredGenericWebhookFields(mapped)
	if err != nil {
		return nil, err
	}
	customer, err := s.findOrCreateGenericWebhookCustomer(ctx, org.ID, customerEmail)
	if err != nil {
		return nil, err
	}

	event, err := newGenericWebhookEvent(org.ID, customer.ID, eventType, payload, body, mapped)
	if err != nil {
		return nil, err
	}
	if err := s.events.Upsert(ctx, event); err != nil {
		return nil, fmt.Errorf("upsert generic webhook event: %w", err)
	}

	return &GenericWebhookProcessResult{EventID: event.ID, CustomerID: customer.ID, EventType: event.EventType}, nil
}

func newGenericWebhookConfig(orgID uuid.UUID, req GenericWebhookConfigRequest) *repository.GenericWebhookConfig {
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	return &repository.GenericWebhookConfig{
		OrgID:        orgID,
		Name:         strings.TrimSpace(req.Name),
		Secret:       req.Secret,
		FieldMapping: req.FieldMapping,
		IsActive:     active,
	}
}

func (s *GenericWebhookService) findOrCreateGenericWebhookCustomer(ctx context.Context, orgID uuid.UUID, email string) (*repository.Customer, error) {
	customer, err := s.customers.GetByEmail(ctx, orgID, email)
	if err != nil {
		return nil, fmt.Errorf("get customer by email: %w", err)
	}
	if customer != nil {
		return customer, nil
	}

	now := time.Now().UTC()
	customer = &repository.Customer{
		OrgID:       orgID,
		ExternalID:  email,
		Source:      GenericWebhookEventSource,
		Email:       email,
		Name:        email,
		Currency:    genericWebhookDefaultCurrency,
		FirstSeenAt: &now,
		LastSeenAt:  &now,
		Metadata:    map[string]any{"source": GenericWebhookEventSource},
	}
	if err := s.customers.UpsertByExternal(ctx, customer); err != nil {
		return nil, fmt.Errorf("upsert generic webhook customer: %w", err)
	}
	return customer, nil
}

func newGenericWebhookEvent(orgID, customerID uuid.UUID, eventType string, payload []byte, body map[string]any, mapped map[string]any) (*repository.CustomerEvent, error) {
	event := &repository.CustomerEvent{
		OrgID:           orgID,
		CustomerID:      customerID,
		EventType:       eventType,
		Source:          GenericWebhookEventSource,
		ExternalEventID: stringFromMapped(mapped, genericWebhookExternalEventIDKey),
		OccurredAt:      time.Now().UTC(),
		Data:            genericWebhookEventData(body, mapped),
	}
	if event.ExternalEventID == "" {
		event.ExternalEventID = genericWebhookPayloadID(payload)
	}
	if occurredAt := stringFromMapped(mapped, genericWebhookOccurredAtKey); occurredAt != "" {
		parsed, err := time.Parse(time.RFC3339, occurredAt)
		if err != nil {
			return nil, &ValidationError{Field: "field_mapping.occurred_at", Message: "occurred_at must be RFC3339"}
		}
		event.OccurredAt = parsed.UTC()
	}
	return event, nil
}

// Test maps a sample payload without creating customers or events.
func (s *GenericWebhookService) Test(ctx context.Context, orgID uuid.UUID, req GenericWebhookTestRequest) (*GenericWebhookTestResult, error) {
	_ = ctx
	_ = orgID
	if req.Payload == nil {
		return nil, &ValidationError{Field: "payload", Message: "payload is required"}
	}
	if req.FieldMapping == nil {
		return nil, &ValidationError{Field: "field_mapping", Message: "field_mapping is required"}
	}
	mapped, err := applyGenericWebhookMapping(req.Payload, req.FieldMapping)
	if err != nil {
		return nil, err
	}
	return &GenericWebhookTestResult{Mapped: mapped}, nil
}

func validateGenericWebhookConfig(req GenericWebhookConfigRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return &ValidationError{Field: "name", Message: "name is required"}
	}
	if req.FieldMapping == nil {
		return &ValidationError{Field: "field_mapping", Message: "field_mapping is required"}
	}
	if _, ok := req.FieldMapping[genericWebhookCustomerEmailKey]; !ok {
		return &ValidationError{Field: "field_mapping.customer_email", Message: "customer_email mapping is required"}
	}
	if _, ok := req.FieldMapping[genericWebhookEventTypeKey]; !ok {
		return &ValidationError{Field: "field_mapping.event_type", Message: "event_type mapping is required"}
	}
	return nil
}

func requiredGenericWebhookFields(mapped map[string]any) (string, string, error) {
	customerEmail := stringFromMapped(mapped, genericWebhookCustomerEmailKey)
	if customerEmail == "" {
		return "", "", &ValidationError{Field: "field_mapping.customer_email", Message: "customer_email mapping is required"}
	}
	eventType := stringFromMapped(mapped, genericWebhookEventTypeKey)
	if eventType == "" {
		return "", "", &ValidationError{Field: "field_mapping.event_type", Message: "event_type mapping is required"}
	}
	return customerEmail, eventType, nil
}

func verifyGenericWebhookSignature(secret string, payload []byte, signature string) error {
	if secret == "" {
		return nil
	}
	if signature == "" {
		return &ValidationError{Field: "signature", Message: "missing webhook signature"}
	}
	got := strings.TrimPrefix(signature, "sha256=")
	decoded, err := hex.DecodeString(got)
	if err != nil {
		return &ValidationError{Field: "signature", Message: "invalid webhook signature"}
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	if !hmac.Equal(decoded, mac.Sum(nil)) {
		return &ValidationError{Field: "signature", Message: "invalid webhook signature"}
	}
	return nil
}

func applyGenericWebhookMapping(payload map[string]any, mapping map[string]string) (map[string]any, error) {
	mapped := map[string]any{}
	for key, path := range mapping {
		value, ok := extractGenericWebhookPath(payload, path)
		if !ok {
			continue
		}
		mapped[key] = value
	}
	return mapped, nil
}

func extractGenericWebhookPath(payload map[string]any, path string) (any, bool) {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "$")
	path = strings.TrimPrefix(path, ".")
	if path == "" {
		return payload, true
	}
	var current any = payload
	for _, part := range strings.Split(path, ".") {
		if part == "" {
			return nil, false
		}
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = m[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func stringFromMapped(mapped map[string]any, key string) string {
	v, ok := mapped[key]
	if !ok || v == nil {
		return ""
	}
	switch typed := v.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(typed)
	default:
		return fmt.Sprint(typed)
	}
}

func genericWebhookEventData(payload map[string]any, mapped map[string]any) map[string]any {
	data := map[string]any{"payload": payload}
	for key, value := range mapped {
		if _, reserved := genericWebhookReservedDataKeys[key]; reserved {
			continue
		}
		data[key] = value
	}
	return data
}

func genericWebhookPayloadID(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}
