package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
	connectorsdk "github.com/onnwee/pulse-score/pkg/connector-sdk"
)

func TestPostHogConnectorManifestAndAuthenticate(t *testing.T) {
	client := NewPostHogClient("http://posthog.test", http.DefaultClient)
	connector := NewPostHogConnector(NewPostHogService(PostHogConfig{}, nil, client, nil, nil))

	manifest := connector.Manifest()
	if manifest.ID != "posthog" {
		t.Fatalf("expected posthog manifest, got %s", manifest.ID)
	}
	if manifest.Auth.Type != connectorsdk.AuthTypeAPIKey {
		t.Fatalf("expected api_key auth, got %s", manifest.Auth.Type)
	}
	if manifest.Auth.APIKey == nil || manifest.Auth.APIKey.HeaderName != "Authorization" {
		t.Fatalf("expected authorization api key config, got %#v", manifest.Auth.APIKey)
	}

	result, err := connector.Authenticate(context.Background(), connectorsdk.AuthRequest{
		OrgID:  uuid.NewString(),
		APIKey: "phx_test",
		Metadata: map[string]string{
			"project_id": "123",
		},
	})
	if err == nil {
		t.Fatal("expected validation error without connection store")
	}
	if result != nil {
		t.Fatalf("expected nil auth result, got %#v", result)
	}
}

func TestPostHogSyncUsesMockedAPIAndMapsUsageEvents(t *testing.T) {
	orgID := uuid.New()
	apiKey := "phx_test"
	projectID := "123"
	now := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+apiKey {
			t.Fatalf("expected bearer auth, got %q", got)
		}
		switch r.URL.Path {
		case "/api/projects/123/persons/":
			writePostHogTestJSON(t, w, map[string]any{
				"results": []map[string]any{
					{
						"id":           "person_1",
						"distinct_ids": []string{"user_1"},
						"properties": map[string]any{
							"email":        "ada@example.com",
							"name":         "Ada Lovelace",
							"company_name": "Analytical Engines Inc",
						},
					},
				},
			})
		case "/api/projects/123/events/":
			writePostHogTestJSON(t, w, map[string]any{
				"results": []map[string]any{
					{
						"id":          "evt_login",
						"event":       "user login",
						"distinct_id": "user_1",
						"timestamp":   now.Format(time.RFC3339),
						"properties": map[string]any{
							"feature": "auth",
						},
					},
					{
						"id":          "evt_feature",
						"event":       "Dashboard Viewed",
						"distinct_id": "user_1",
						"timestamp":   now.Add(time.Minute).Format(time.RFC3339),
					},
				},
			})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	customers := &fakePostHogCustomerStore{byExternalID: map[string]*repository.Customer{}, byEmail: map[string]*repository.Customer{}}
	events := &fakePostHogEventStore{}
	store := &fakePostHogConnectionStore{connection: &repository.IntegrationConnection{
		OrgID:                orgID,
		Provider:             "posthog",
		Status:               "active",
		AccessTokenEncrypted: []byte(apiKey),
		ExternalAccountID:    projectID,
		Metadata:             map[string]any{"project_id": projectID},
	}}
	svc := NewPostHogService(PostHogConfig{}, store, NewPostHogClient(server.URL, server.Client()), customers, events)

	result, err := svc.Sync(context.Background(), orgID, connectorsdk.SyncModeFull, nil)
	if err != nil {
		t.Fatalf("sync posthog: %v", err)
	}

	if len(customers.upserts) != 1 {
		t.Fatalf("expected one customer upsert, got %d", len(customers.upserts))
	}
	customer := customers.upserts[0]
	if customer.Source != "posthog" || customer.ExternalID != "user_1" || customer.Email != "ada@example.com" {
		t.Fatalf("unexpected customer mapping: %#v", customer)
	}
	if len(events.upserts) != 2 {
		t.Fatalf("expected two usage events, got %d", len(events.upserts))
	}
	if events.upserts[0].EventType != "login" || events.upserts[1].EventType != "feature_use" {
		t.Fatalf("expected normalized usage events, got %#v", events.upserts)
	}
	if result.Resources[0].Name != "persons" || result.Resources[0].Synced != 1 {
		t.Fatalf("unexpected persons resource result: %#v", result.Resources)
	}
	if result.Resources[1].Name != "events" || result.Resources[1].Synced != 2 {
		t.Fatalf("unexpected events resource result: %#v", result.Resources)
	}
}

func TestPostHogSyncMatchesPersonsByEmail(t *testing.T) {
	orgID := uuid.New()
	apiKey := "phx_test"
	projectID := "123"
	existingCustomerID := uuid.New()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/projects/123/persons/":
			writePostHogTestJSON(t, w, map[string]any{
				"results": []map[string]any{
					{
						"distinct_ids": []string{"posthog_user_1"},
						"properties": map[string]any{
							"email": "ada@example.com",
						},
					},
				},
			})
		case "/api/projects/123/events/":
			writePostHogTestJSON(t, w, map[string]any{
				"results": []map[string]any{
					{
						"id":          "evt_1",
						"event":       "Feature Used",
						"distinct_id": "posthog_user_1",
					},
				},
			})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	customers := &fakePostHogCustomerStore{
		byExternalID: map[string]*repository.Customer{},
		byEmail: map[string]*repository.Customer{
			"ada@example.com": {ID: existingCustomerID, OrgID: orgID, Source: "stripe", ExternalID: "cus_123", Email: "ada@example.com"},
		},
	}
	events := &fakePostHogEventStore{}
	store := &fakePostHogConnectionStore{connection: &repository.IntegrationConnection{
		OrgID:                orgID,
		Provider:             "posthog",
		Status:               "active",
		AccessTokenEncrypted: []byte(apiKey),
		ExternalAccountID:    projectID,
		Metadata:             map[string]any{"project_id": projectID},
	}}
	svc := NewPostHogService(PostHogConfig{}, store, NewPostHogClient(server.URL, server.Client()), customers, events)

	_, err := svc.Sync(context.Background(), orgID, connectorsdk.SyncModeFull, nil)
	if err != nil {
		t.Fatalf("sync posthog: %v", err)
	}

	if len(customers.upserts) != 0 {
		t.Fatalf("expected no duplicate PostHog customer upsert, got %d", len(customers.upserts))
	}
	if len(events.upserts) != 1 {
		t.Fatalf("expected one usage event, got %d", len(events.upserts))
	}
	if events.upserts[0].CustomerID != existingCustomerID {
		t.Fatalf("expected event for existing customer %s, got %s", existingCustomerID, events.upserts[0].CustomerID)
	}
}

func writePostHogTestJSON(t *testing.T, w http.ResponseWriter, payload any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		t.Fatalf("encode response: %v", err)
	}
}

type fakePostHogConnectionStore struct {
	connection *repository.IntegrationConnection
}

func (s *fakePostHogConnectionStore) Upsert(_ context.Context, conn *repository.IntegrationConnection) error {
	s.connection = conn
	return nil
}

func (s *fakePostHogConnectionStore) GetByOrgAndProvider(_ context.Context, _ uuid.UUID, provider string) (*repository.IntegrationConnection, error) {
	if s.connection == nil || s.connection.Provider != provider {
		return nil, nil
	}
	return s.connection, nil
}

type fakePostHogCustomerStore struct {
	upserts      []*repository.Customer
	byExternalID map[string]*repository.Customer
	byEmail      map[string]*repository.Customer
}

func (s *fakePostHogCustomerStore) UpsertByExternal(_ context.Context, customer *repository.Customer) error {
	if customer.ID == uuid.Nil {
		customer.ID = uuid.New()
	}
	s.upserts = append(s.upserts, customer)
	s.byExternalID[customer.ExternalID] = customer
	if customer.Email != "" {
		s.byEmail[customer.Email] = customer
	}
	return nil
}

func (s *fakePostHogCustomerStore) GetByExternalID(_ context.Context, _ uuid.UUID, source, externalID string) (*repository.Customer, error) {
	if source != "posthog" {
		return nil, nil
	}
	return s.byExternalID[externalID], nil
}

func (s *fakePostHogCustomerStore) GetByEmail(_ context.Context, _ uuid.UUID, email string) (*repository.Customer, error) {
	return s.byEmail[email], nil
}

type fakePostHogEventStore struct {
	upserts []*repository.CustomerEvent
}

func (s *fakePostHogEventStore) Upsert(_ context.Context, event *repository.CustomerEvent) error {
	s.upserts = append(s.upserts, event)
	return nil
}
