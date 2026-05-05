package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

const zendeskProvider = "zendesk"

var zendeskSubdomainPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,61}[a-z0-9]$|^[a-z0-9]$`)

// ZendeskOAuthConfig holds Zendesk OAuth settings.
type ZendeskOAuthConfig struct {
	ClientID         string
	ClientSecret     string
	OAuthRedirectURL string
	EncryptionKey    string
}

// ZendeskOAuthService handles Zendesk OAuth connect flow.
type ZendeskOAuthService struct {
	cfg      ZendeskOAuthConfig
	connRepo *repository.IntegrationConnectionRepository
}

// NewZendeskOAuthService creates a new ZendeskOAuthService.
func NewZendeskOAuthService(cfg ZendeskOAuthConfig, connRepo *repository.IntegrationConnectionRepository) *ZendeskOAuthService {
	return &ZendeskOAuthService{cfg: cfg, connRepo: connRepo}
}

// ConnectURL generates the Zendesk OAuth authorization URL for one Zendesk subdomain.
func (s *ZendeskOAuthService) ConnectURL(orgID uuid.UUID, rawSubdomain string) (string, error) {
	if s.cfg.ClientID == "" {
		return "", &ValidationError{Field: zendeskProvider, Message: "Zendesk integration is not configured"}
	}

	subdomain, err := normalizeZendeskSubdomain(rawSubdomain)
	if err != nil {
		return "", err
	}

	state := fmt.Sprintf("%s:%s:%d", orgID.String(), subdomain, time.Now().UnixNano())
	params := url.Values{
		"client_id":     {s.cfg.ClientID},
		"redirect_uri":  {s.cfg.OAuthRedirectURL},
		"response_type": {"code"},
		"scope":         {"read"},
		"state":         {state},
	}

	return fmt.Sprintf("https://%s.zendesk.com/oauth/authorizations/new?%s", subdomain, params.Encode()), nil
}

// ExchangeCode exchanges a Zendesk OAuth code and stores the connection.
func (s *ZendeskOAuthService) ExchangeCode(ctx context.Context, orgID uuid.UUID, code, state string) error {
	if code == "" {
		return &ValidationError{Field: "code", Message: "authorization code is required"}
	}
	subdomain, err := zendeskSubdomainFromState(orgID, state)
	if err != nil {
		return err
	}

	tokenResp, err := s.exchangeCodeWithZendesk(code, subdomain)
	if err != nil {
		return fmt.Errorf("exchange code with zendesk: %w", err)
	}

	encrypted, err := encryptToken(tokenResp.AccessToken, s.cfg.EncryptionKey)
	if err != nil {
		return fmt.Errorf("encrypt access token: %w", err)
	}

	conn := &repository.IntegrationConnection{
		OrgID:                orgID,
		Provider:             zendeskProvider,
		Status:               "active",
		AccessTokenEncrypted: encrypted,
		ExternalAccountID:    subdomain,
		Scopes:               []string{"read"},
		Metadata: map[string]any{
			"subdomain":  subdomain,
			"token_type": tokenResp.TokenType,
		},
	}

	if err := s.connRepo.Upsert(ctx, conn); err != nil {
		return fmt.Errorf("store connection: %w", err)
	}

	slog.Info("zendesk connection established", "org_id", orgID, "subdomain", subdomain)
	return nil
}

// GetAccessToken retrieves and decrypts the Zendesk access token.
func (s *ZendeskOAuthService) GetAccessToken(ctx context.Context, orgID uuid.UUID) (string, string, error) {
	conn, err := s.connRepo.GetByOrgAndProvider(ctx, orgID, zendeskProvider)
	if err != nil {
		return "", "", fmt.Errorf("get connection: %w", err)
	}
	if conn == nil {
		return "", "", &NotFoundError{Resource: "zendesk_connection", Message: "no Zendesk connection found"}
	}
	if conn.Status != "active" && conn.Status != "syncing" {
		return "", "", &ValidationError{Field: zendeskProvider, Message: "Zendesk connection is not active"}
	}

	token, err := decryptToken(conn.AccessTokenEncrypted, s.cfg.EncryptionKey)
	if err != nil {
		return "", "", fmt.Errorf("decrypt access token: %w", err)
	}
	return token, zendeskSubdomainFromConnection(conn), nil
}

// GetStatus returns current Zendesk connection status.
func (s *ZendeskOAuthService) GetStatus(ctx context.Context, orgID uuid.UUID) (*ZendeskConnectionStatus, error) {
	conn, err := s.connRepo.GetByOrgAndProvider(ctx, orgID, zendeskProvider)
	if err != nil {
		return nil, fmt.Errorf("get connection: %w", err)
	}
	if conn == nil {
		return &ZendeskConnectionStatus{Status: "disconnected"}, nil
	}

	return &ZendeskConnectionStatus{
		Status:            conn.Status,
		Subdomain:         zendeskSubdomainFromConnection(conn),
		ExternalAccountID: conn.ExternalAccountID,
		LastSyncAt:        conn.LastSyncAt,
		LastSyncError:     conn.LastSyncError,
		ConnectedAt:       conn.CreatedAt,
	}, nil
}

// Disconnect removes a Zendesk connection.
func (s *ZendeskOAuthService) Disconnect(ctx context.Context, orgID uuid.UUID) error {
	return s.connRepo.Delete(ctx, orgID, zendeskProvider)
}

// ZendeskConnectionStatus holds frontend status data.
type ZendeskConnectionStatus struct {
	Status            string     `json:"status"`
	Subdomain         string     `json:"subdomain,omitempty"`
	ExternalAccountID string     `json:"external_account_id,omitempty"`
	LastSyncAt        *time.Time `json:"last_sync_at,omitempty"`
	LastSyncError     string     `json:"last_sync_error,omitempty"`
	ConnectedAt       time.Time  `json:"connected_at,omitempty"`
	TicketCount       int        `json:"ticket_count,omitempty"`
	UserCount         int        `json:"user_count,omitempty"`
}

type zendeskTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

func (s *ZendeskOAuthService) exchangeCodeWithZendesk(code, subdomain string) (*zendeskTokenResponse, error) {
	body := map[string]string{
		"grant_type":    "authorization_code",
		"code":          code,
		"client_id":     s.cfg.ClientID,
		"client_secret": s.cfg.ClientSecret,
		"redirect_uri":  s.cfg.OAuthRedirectURL,
		"scope":         "read",
	}
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("https://%s.zendesk.com/oauth/tokens", subdomain), bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("zendesk token exchange failed with status %d", resp.StatusCode)
	}

	var tokenResp zendeskTokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if tokenResp.AccessToken == "" {
		return nil, fmt.Errorf("zendesk token exchange returned no access token")
	}
	return &tokenResp, nil
}

func normalizeZendeskSubdomain(raw string) (string, error) {
	value := strings.TrimSpace(strings.ToLower(raw))
	if value == "" {
		return "", &ValidationError{Field: "subdomain", Message: "Zendesk subdomain is required"}
	}
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		parsed, err := url.Parse(value)
		if err != nil {
			return "", &ValidationError{Field: "subdomain", Message: "valid Zendesk subdomain is required"}
		}
		value = parsed.Hostname()
	}
	value = strings.TrimSuffix(value, ".zendesk.com")
	if strings.Contains(value, ".") || !zendeskSubdomainPattern.MatchString(value) {
		return "", &ValidationError{Field: "subdomain", Message: "valid Zendesk subdomain is required"}
	}
	return value, nil
}

func zendeskSubdomainFromState(orgID uuid.UUID, state string) (string, error) {
	parts := strings.Split(state, ":")
	if len(parts) != 3 {
		return "", &ValidationError{Field: "state", Message: "invalid state parameter"}
	}
	stateOrgID, err := uuid.Parse(parts[0])
	if err != nil || stateOrgID != orgID {
		return "", &ValidationError{Field: "state", Message: "invalid state parameter"}
	}
	return normalizeZendeskSubdomain(parts[1])
}

func zendeskSubdomainFromConnection(conn *repository.IntegrationConnection) string {
	if conn == nil {
		return ""
	}
	if raw, ok := conn.Metadata["subdomain"].(string); ok {
		return raw
	}
	return conn.ExternalAccountID
}
