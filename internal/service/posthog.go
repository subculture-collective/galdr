package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
	connectorsdk "github.com/onnwee/pulse-score/pkg/connector-sdk"
)

const (
	providerPostHog        = "posthog"
	postHogDefaultBaseURL  = "https://app.posthog.com"
	postHogProjectIDKey    = "project_id"
	postHogReadScope       = "read"
	postHogResourcePersons = "persons"
	postHogResourceEvents  = "events"
)

var _ connectorsdk.Connector = (*PostHogConnector)(nil)

// PostHogConfig holds API-key connector settings.
type PostHogConfig struct {
	EncryptionKey string
}

type postHogConnectionStore interface {
	Upsert(ctx context.Context, conn *repository.IntegrationConnection) error
	GetByOrgAndProvider(ctx context.Context, orgID uuid.UUID, provider string) (*repository.IntegrationConnection, error)
}

type postHogCustomerStore interface {
	UpsertByExternal(ctx context.Context, customer *repository.Customer) error
	GetByExternalID(ctx context.Context, orgID uuid.UUID, source, externalID string) (*repository.Customer, error)
	GetByEmail(ctx context.Context, orgID uuid.UUID, email string) (*repository.Customer, error)
}

type postHogEventStore interface {
	Upsert(ctx context.Context, e *repository.CustomerEvent) error
}

// PostHogService validates credentials and syncs usage data.
type PostHogService struct {
	cfg       PostHogConfig
	connStore postHogConnectionStore
	client    *PostHogClient
	customers postHogCustomerStore
	events    postHogEventStore
}

// NewPostHogService creates a PostHog API-key service.
func NewPostHogService(cfg PostHogConfig, connStore postHogConnectionStore, client *PostHogClient, customers postHogCustomerStore, events postHogEventStore) *PostHogService {
	if client == nil {
		client = NewPostHogClient("", nil)
	}
	return &PostHogService{
		cfg:       cfg,
		connStore: connStore,
		client:    client,
		customers: customers,
		events:    events,
	}
}

// Connect validates and stores an org's PostHog API-key connection.
func (s *PostHogService) Connect(ctx context.Context, orgID uuid.UUID, projectID, apiKey string) (*connectorsdk.AuthResult, error) {
	projectID = strings.TrimSpace(projectID)
	apiKey = strings.TrimSpace(apiKey)
	if projectID == "" {
		return nil, &ValidationError{Field: "project_id", Message: "project id is required"}
	}
	if apiKey == "" {
		return nil, &ValidationError{Field: "api_key", Message: "api key is required"}
	}
	if s.connStore == nil {
		return nil, &ValidationError{Field: providerPostHog, Message: "PostHog connection store is not configured"}
	}
	if err := s.client.ValidateAPIKey(ctx, projectID, apiKey); err != nil {
		return nil, err
	}

	encrypted, err := encryptToken(apiKey, s.cfg.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt posthog api key: %w", err)
	}
	conn := &repository.IntegrationConnection{
		OrgID:                orgID,
		Provider:             providerPostHog,
		Status:               "active",
		AccessTokenEncrypted: encrypted,
		ExternalAccountID:    projectID,
		Scopes:               []string{postHogReadScope},
		Metadata:             map[string]any{postHogProjectIDKey: projectID},
	}
	if err := s.connStore.Upsert(ctx, conn); err != nil {
		return nil, fmt.Errorf("store posthog connection: %w", err)
	}
	return &connectorsdk.AuthResult{
		ExternalAccountID: projectID,
		Scopes:            []string{postHogReadScope},
		Metadata:          map[string]string{postHogProjectIDKey: projectID},
	}, nil
}

// Sync imports PostHog persons and events.
func (s *PostHogService) Sync(ctx context.Context, orgID uuid.UUID, mode connectorsdk.SyncMode, since *time.Time) (*connectorsdk.SyncResult, error) {
	if s.customers == nil || s.events == nil {
		return nil, &ValidationError{Field: providerPostHog, Message: "PostHog sync stores are not configured"}
	}
	if mode == connectorsdk.SyncModeIncremental && since == nil {
		return nil, &ValidationError{Field: "since", Message: "incremental sync requires since"}
	}
	projectID, apiKey, err := s.credentials(ctx, orgID)
	if err != nil {
		return nil, err
	}

	persons, err := s.client.ListPersons(ctx, projectID, apiKey)
	if err != nil {
		return nil, fmt.Errorf("list posthog persons: %w", err)
	}
	personProgress := connectorsdk.ResourceResult{Name: postHogResourcePersons}
	personMatches := make(map[string]*repository.Customer)
	for _, person := range persons.Results {
		customer, err := s.upsertPerson(ctx, orgID, person)
		if err != nil {
			personProgress.Skipped++
			continue
		}
		personProgress.Synced++
		for _, distinctID := range person.matchIDs() {
			personMatches[distinctID] = customer
		}
	}

	events, err := s.client.ListEvents(ctx, projectID, apiKey, since)
	if err != nil {
		return nil, fmt.Errorf("list posthog events: %w", err)
	}
	eventProgress := connectorsdk.ResourceResult{Name: postHogResourceEvents}
	for _, event := range events.Results {
		if err := s.upsertEvent(ctx, orgID, event, personMatches); err != nil {
			eventProgress.Skipped++
			continue
		}
		eventProgress.Synced++
	}

	return &connectorsdk.SyncResult{Resources: []connectorsdk.ResourceResult{personProgress, eventProgress}}, nil
}

func (s *PostHogService) credentials(ctx context.Context, orgID uuid.UUID) (string, string, error) {
	if s.connStore == nil {
		return "", "", &ValidationError{Field: providerPostHog, Message: "PostHog connection store is not configured"}
	}
	conn, err := s.connStore.GetByOrgAndProvider(ctx, orgID, providerPostHog)
	if err != nil {
		return "", "", fmt.Errorf("get posthog connection: %w", err)
	}
	if conn == nil {
		return "", "", &NotFoundError{Resource: "posthog_connection", Message: "no PostHog connection found"}
	}
	if conn.Status != "active" {
		return "", "", &ValidationError{Field: providerPostHog, Message: "PostHog connection is not active"}
	}
	projectID := conn.ExternalAccountID
	if raw, ok := conn.Metadata[postHogProjectIDKey].(string); ok && raw != "" {
		projectID = raw
	}
	apiKey, err := decryptToken(conn.AccessTokenEncrypted, s.cfg.EncryptionKey)
	if err != nil {
		return "", "", fmt.Errorf("decrypt posthog api key: %w", err)
	}
	return projectID, apiKey, nil
}

func (s *PostHogService) upsertPerson(ctx context.Context, orgID uuid.UUID, person PostHogPerson) (*repository.Customer, error) {
	externalID := person.ExternalID()
	if externalID == "" {
		return nil, &ValidationError{Field: "distinct_id", Message: "PostHog person distinct id is required"}
	}
	now := time.Now()
	email := postHogString(person.Properties, "email", "$email")
	name := postHogString(person.Properties, "name", "$name")
	if email != "" {
		existing, err := s.customers.GetByEmail(ctx, orgID, email)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return existing, nil
		}
	}
	customer := &repository.Customer{
		OrgID:       orgID,
		ExternalID:  externalID,
		Source:      providerPostHog,
		Email:       email,
		Name:        name,
		CompanyName: postHogString(person.Properties, "company_name", "company", "$company"),
		FirstSeenAt: parsePostHogTimePtr(person.CreatedAt),
		LastSeenAt:  parsePostHogTimePtr(postHogString(person.Properties, "last_seen", "$last_seen")),
		Metadata:    map[string]any{"posthog": person.Properties},
	}
	if customer.FirstSeenAt == nil {
		customer.FirstSeenAt = &now
	}
	if customer.LastSeenAt == nil {
		customer.LastSeenAt = customer.FirstSeenAt
	}
	if err := s.customers.UpsertByExternal(ctx, customer); err != nil {
		return nil, err
	}
	return customer, nil
}

func (s *PostHogService) upsertEvent(ctx context.Context, orgID uuid.UUID, event PostHogEvent, personMatches map[string]*repository.Customer) error {
	if event.DistinctID == "" {
		return &ValidationError{Field: "distinct_id", Message: "PostHog event distinct id is required"}
	}
	customer, err := s.matchEventCustomer(ctx, orgID, event, personMatches)
	if err != nil {
		return err
	}
	if customer == nil {
		return &NotFoundError{Resource: "posthog_customer", Message: "no customer matched PostHog event"}
	}
	occurredAt := time.Now()
	if parsed := parsePostHogTimePtr(event.Timestamp); parsed != nil {
		occurredAt = *parsed
	}
	externalEventID := event.ID
	if externalEventID == "" {
		externalEventID = fmt.Sprintf("%s:%s:%d", event.DistinctID, event.Event, occurredAt.UnixNano())
	}
	return s.events.Upsert(ctx, &repository.CustomerEvent{
		OrgID:           orgID,
		CustomerID:      customer.ID,
		EventType:       normalizePostHogEventType(event.Event),
		Source:          providerPostHog,
		ExternalEventID: externalEventID,
		OccurredAt:      occurredAt,
		Data:            map[string]any{"posthog_event": event.Event, "properties": event.Properties},
	})
}

func (s *PostHogService) matchEventCustomer(ctx context.Context, orgID uuid.UUID, event PostHogEvent, personMatches map[string]*repository.Customer) (*repository.Customer, error) {
	if customer := personMatches[event.DistinctID]; customer != nil {
		return customer, nil
	}

	customer, err := s.customers.GetByExternalID(ctx, orgID, providerPostHog, event.DistinctID)
	if err != nil || customer != nil {
		return customer, err
	}

	email := postHogString(event.Properties, "email", "$email")
	if email == "" {
		return nil, nil
	}
	return s.customers.GetByEmail(ctx, orgID, email)
}

// PostHogConnector adapts PostHog API-key sync to connector SDK.
type PostHogConnector struct {
	service *PostHogService
}

// NewPostHogConnector creates a PostHog connector adapter.
func NewPostHogConnector(service *PostHogService) *PostHogConnector {
	return &PostHogConnector{service: service}
}

// Manifest returns PostHog connector metadata.
func (c *PostHogConnector) Manifest() connectorsdk.ConnectorManifest {
	return connectorsdk.ConnectorManifest{
		ID:          providerPostHog,
		Name:        "PostHog",
		Version:     "1.0.0",
		Description: "Syncs PostHog product usage persons and events.",
		Categories:  []string{"analytics", "product-usage"},
		Auth: connectorsdk.AuthConfig{
			Type: connectorsdk.AuthTypeAPIKey,
			APIKey: &connectorsdk.APIKeyConfig{
				HeaderName: "Authorization",
				Prefix:     "Bearer",
			},
		},
		Sync: connectorsdk.SyncConfig{
			SupportedModes: []connectorsdk.SyncMode{connectorsdk.SyncModeFull, connectorsdk.SyncModeIncremental},
			DefaultMode:    connectorsdk.SyncModeFull,
			Resources: []connectorsdk.ResourceConfig{
				{Name: postHogResourcePersons, Description: "PostHog persons and user identities", Required: true},
				{Name: postHogResourceEvents, Description: "PostHog product usage events", Required: true},
			},
		},
	}
}

// Authenticate validates and stores API-key credentials.
func (c *PostHogConnector) Authenticate(ctx context.Context, req connectorsdk.AuthRequest) (*connectorsdk.AuthResult, error) {
	if c.service == nil {
		return nil, &ValidationError{Field: providerPostHog, Message: "PostHog service is not configured"}
	}
	orgID, err := parseConnectorOrgID(req.OrgID)
	if err != nil {
		return nil, err
	}
	return c.service.Connect(ctx, orgID, req.Metadata[postHogProjectIDKey], req.APIKey)
}

// Sync runs a PostHog sync.
func (c *PostHogConnector) Sync(ctx context.Context, req connectorsdk.SyncRequest) (*connectorsdk.SyncResult, error) {
	if c.service == nil {
		return nil, &ValidationError{Field: providerPostHog, Message: "PostHog service is not configured"}
	}
	orgID, err := parseConnectorOrgID(req.OrgID)
	if err != nil {
		return nil, err
	}
	return c.service.Sync(ctx, orgID, req.Mode, req.Since)
}

// HandleEvent accepts no provider webhooks yet.
func (c *PostHogConnector) HandleEvent(context.Context, connectorsdk.EventRequest) (*connectorsdk.EventResult, error) {
	return &connectorsdk.EventResult{Accepted: false}, nil
}

// PostHogClient calls the PostHog API.
type PostHogClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewPostHogClient creates a PostHog API client.
func NewPostHogClient(baseURL string, httpClient *http.Client) *PostHogClient {
	if baseURL == "" {
		baseURL = postHogDefaultBaseURL
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &PostHogClient{baseURL: strings.TrimRight(baseURL, "/"), httpClient: httpClient}
}

func (c *PostHogClient) ValidateAPIKey(ctx context.Context, projectID, apiKey string) error {
	_, err := c.do(ctx, projectID, apiKey, "persons/", nil)
	return err
}

func (c *PostHogClient) ListPersons(ctx context.Context, projectID, apiKey string) (*PostHogPersonListResponse, error) {
	var out PostHogPersonListResponse
	if err := c.doJSON(ctx, projectID, apiKey, "persons/", nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *PostHogClient) ListEvents(ctx context.Context, projectID, apiKey string, since *time.Time) (*PostHogEventListResponse, error) {
	query := url.Values{}
	if since != nil {
		query.Set("after", since.Format(time.RFC3339))
	}
	var out PostHogEventListResponse
	if err := c.doJSON(ctx, projectID, apiKey, "events/", query, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *PostHogClient) doJSON(ctx context.Context, projectID, apiKey, resource string, query url.Values, out any) error {
	body, err := c.do(ctx, projectID, apiKey, resource, query)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, out)
}

func (c *PostHogClient) do(ctx context.Context, projectID, apiKey, resource string, query url.Values) ([]byte, error) {
	path := fmt.Sprintf("%s/api/projects/%s/%s", c.baseURL, url.PathEscape(projectID), resource)
	if len(query) > 0 {
		path += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("posthog api returned status %d", resp.StatusCode)
	}
	return body, nil
}

// PostHogPersonListResponse is a paginated persons response.
type PostHogPersonListResponse struct {
	Results []PostHogPerson `json:"results"`
}

// PostHogPerson is a PostHog person payload.
type PostHogPerson struct {
	ID          string         `json:"id"`
	DistinctIDs []string       `json:"distinct_ids"`
	Properties  map[string]any `json:"properties"`
	CreatedAt   string         `json:"created_at"`
}

func (p PostHogPerson) ExternalID() string {
	if len(p.DistinctIDs) > 0 && p.DistinctIDs[0] != "" {
		return p.DistinctIDs[0]
	}
	return p.ID
}

func (p PostHogPerson) matchIDs() []string {
	ids := make([]string, 0, len(p.DistinctIDs)+1)
	for _, distinctID := range p.DistinctIDs {
		if distinctID != "" {
			ids = append(ids, distinctID)
		}
	}
	if p.ID != "" {
		ids = append(ids, p.ID)
	}
	return ids
}

// PostHogEventListResponse is a paginated events response.
type PostHogEventListResponse struct {
	Results []PostHogEvent `json:"results"`
}

// PostHogEvent is a PostHog event payload.
type PostHogEvent struct {
	ID         string         `json:"id"`
	Event      string         `json:"event"`
	DistinctID string         `json:"distinct_id"`
	Timestamp  string         `json:"timestamp"`
	Properties map[string]any `json:"properties"`
}

func postHogString(properties map[string]any, keys ...string) string {
	for _, key := range keys {
		if raw, ok := properties[key].(string); ok {
			return raw
		}
	}
	return ""
}

func parsePostHogTimePtr(raw string) *time.Time {
	if raw == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil
	}
	return &parsed
}

func normalizePostHogEventType(event string) string {
	lower := strings.ToLower(event)
	switch {
	case strings.Contains(lower, "login") || strings.Contains(lower, "sign in"):
		return "login"
	case strings.Contains(lower, "api"):
		return "api_call"
	default:
		return "feature_use"
	}
}
