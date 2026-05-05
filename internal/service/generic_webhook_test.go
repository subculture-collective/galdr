package service

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

func TestGenericWebhookProcessCreatesCustomerEventFromMapping(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	webhookID := uuid.New()
	customerID := uuid.New()
	payload := []byte(`{"id":"evt_123","event":"invoice.paid","data":{"email":"buyer@example.com","amount":4200},"created_at":"2026-05-05T10:30:00Z"}`)

	repos := &fakeGenericWebhookRepos{
		org: &repository.Organization{ID: orgID, Slug: "acme"},
		config: &repository.GenericWebhookConfig{
			ID:     webhookID,
			OrgID:  orgID,
			Name:   "Billing feed",
			Secret: "top-secret",
			FieldMapping: map[string]string{
				"customer_email":    "$.data.email",
				"event_type":        "$.event",
				"external_event_id": "$.id",
				"occurred_at":       "$.created_at",
				"amount":            "$.data.amount",
			},
		},
		customer: &repository.Customer{ID: customerID, OrgID: orgID, Email: "buyer@example.com"},
	}
	svc := NewGenericWebhookService(repos, repos, repos, repos)

	res, err := svc.Process(ctx, "acme", webhookID, payload, signGenericWebhookPayload("top-secret", payload))
	if err != nil {
		t.Fatalf("process webhook: %v", err)
	}

	if res.EventID == uuid.Nil || repos.upsertedEvent == nil {
		t.Fatal("expected event to be upserted")
	}
	if repos.upsertedEvent.OrgID != orgID || repos.upsertedEvent.CustomerID != customerID {
		t.Fatalf("unexpected event owner: %+v", repos.upsertedEvent)
	}
	if repos.upsertedEvent.EventType != "invoice.paid" || repos.upsertedEvent.ExternalEventID != "evt_123" {
		t.Fatalf("unexpected event identity: %+v", repos.upsertedEvent)
	}
	if repos.upsertedEvent.Source != GenericWebhookEventSource {
		t.Fatalf("expected generic source, got %q", repos.upsertedEvent.Source)
	}
	if repos.upsertedEvent.Data["amount"] != float64(4200) {
		t.Fatalf("expected mapped amount, got %+v", repos.upsertedEvent.Data)
	}
	expectedOccurredAt := time.Date(2026, 5, 5, 10, 30, 0, 0, time.UTC)
	if !repos.upsertedEvent.OccurredAt.Equal(expectedOccurredAt) {
		t.Fatalf("expected occurred_at %s, got %s", expectedOccurredAt, repos.upsertedEvent.OccurredAt)
	}
}

func TestGenericWebhookProcessRejectsInvalidSignature(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	webhookID := uuid.New()
	repos := &fakeGenericWebhookRepos{
		org: &repository.Organization{ID: orgID, Slug: "acme"},
		config: &repository.GenericWebhookConfig{ID: webhookID, OrgID: orgID, Secret: "top-secret", FieldMapping: map[string]string{"customer_email": "$.email", "event_type": "$.event"}},
	}
	svc := NewGenericWebhookService(repos, repos, repos, repos)

	_, err := svc.Process(ctx, "acme", webhookID, []byte(`{"email":"buyer@example.com","event":"login"}`), "sha256=bad")
	if err == nil {
		t.Fatal("expected invalid signature error")
	}
	if repos.upsertedEvent != nil {
		t.Fatal("event should not be created when signature is invalid")
	}
}

func TestGenericWebhookProcessCreatesCustomerWhenMissing(t *testing.T) {
	ctx := context.Background()
	orgID := uuid.New()
	webhookID := uuid.New()
	payload := []byte(`{"email":"new@example.com","event":"feature.used"}`)
	repos := &fakeGenericWebhookRepos{
		org:    &repository.Organization{ID: orgID, Slug: "acme"},
		config: &repository.GenericWebhookConfig{ID: webhookID, OrgID: orgID, FieldMapping: map[string]string{"customer_email": "$.email", "event_type": "$.event"}},
	}
	svc := NewGenericWebhookService(repos, repos, repos, repos)

	_, err := svc.Process(ctx, "acme", webhookID, payload, "")
	if err != nil {
		t.Fatalf("process webhook: %v", err)
	}

	if repos.upsertedCustomer == nil {
		t.Fatal("expected missing customer to be created")
	}
	if repos.upsertedCustomer.Email != "new@example.com" || repos.upsertedCustomer.ExternalID != "new@example.com" || repos.upsertedCustomer.Source != GenericWebhookEventSource {
		t.Fatalf("unexpected customer: %+v", repos.upsertedCustomer)
	}
	if repos.upsertedEvent == nil || repos.upsertedEvent.CustomerID != repos.upsertedCustomer.ID {
		t.Fatalf("event should reference created customer: event=%+v customer=%+v", repos.upsertedEvent, repos.upsertedCustomer)
	}
}

func signGenericWebhookPayload(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

type fakeGenericWebhookRepos struct {
	org              *repository.Organization
	config           *repository.GenericWebhookConfig
	customer         *repository.Customer
	upsertedCustomer *repository.Customer
	upsertedEvent    *repository.CustomerEvent
}

func (f *fakeGenericWebhookRepos) GetBySlug(_ context.Context, slug string) (*repository.Organization, error) {
	if f.org == nil || f.org.Slug != slug {
		return nil, nil
	}
	return f.org, nil
}

func (f *fakeGenericWebhookRepos) GetActiveByIDAndOrg(_ context.Context, id, orgID uuid.UUID) (*repository.GenericWebhookConfig, error) {
	if f.config == nil || f.config.ID != id || f.config.OrgID != orgID {
		return nil, nil
	}
	return f.config, nil
}

func (f *fakeGenericWebhookRepos) Create(_ context.Context, c *repository.GenericWebhookConfig) error {
	f.config = c
	return nil
}

func (f *fakeGenericWebhookRepos) ListByOrg(_ context.Context, orgID uuid.UUID) ([]*repository.GenericWebhookConfig, error) {
	if f.config == nil || f.config.OrgID != orgID {
		return nil, nil
	}
	return []*repository.GenericWebhookConfig{f.config}, nil
}

func (f *fakeGenericWebhookRepos) GetByIDAndOrg(_ context.Context, id, orgID uuid.UUID) (*repository.GenericWebhookConfig, error) {
	if f.config == nil || f.config.ID != id || f.config.OrgID != orgID {
		return nil, nil
	}
	return f.config, nil
}

func (f *fakeGenericWebhookRepos) Update(_ context.Context, c *repository.GenericWebhookConfig) error {
	f.config = c
	return nil
}

func (f *fakeGenericWebhookRepos) Delete(_ context.Context, orgID, id uuid.UUID) error {
	if f.config != nil && f.config.OrgID == orgID && f.config.ID == id {
		f.config = nil
	}
	return nil
}

func (f *fakeGenericWebhookRepos) GetByEmail(_ context.Context, orgID uuid.UUID, email string) (*repository.Customer, error) {
	if f.customer == nil || f.customer.OrgID != orgID || f.customer.Email != email {
		return nil, nil
	}
	return f.customer, nil
}

func (f *fakeGenericWebhookRepos) UpsertByExternal(_ context.Context, c *repository.Customer) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	f.upsertedCustomer = c
	f.customer = c
	return nil
}

func (f *fakeGenericWebhookRepos) Upsert(_ context.Context, e *repository.CustomerEvent) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	f.upsertedEvent = e
	return nil
}
