package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

const defaultSalesforceLoginURL = "https://login.salesforce.com"

// SalesforceOAuthConfig holds Salesforce OAuth settings.
type SalesforceOAuthConfig struct {
	ClientID         string
	ClientSecret     string
	OAuthRedirectURL string
	EncryptionKey    string
	LoginURL         string
}

// SalesforceAccess is the decrypted Salesforce API access context for an org.
type SalesforceAccess struct {
	AccessToken string
	InstanceURL string
}

// SalesforceOAuthService handles Salesforce OAuth connect flow.
type SalesforceOAuthService struct {
	cfg      SalesforceOAuthConfig
	connRepo *repository.IntegrationConnectionRepository
	client   *http.Client
}

// NewSalesforceOAuthService creates a new SalesforceOAuthService.
func NewSalesforceOAuthService(cfg SalesforceOAuthConfig, connRepo *repository.IntegrationConnectionRepository) *SalesforceOAuthService {
	if cfg.LoginURL == "" {
		cfg.LoginURL = defaultSalesforceLoginURL
	}
	return &SalesforceOAuthService{cfg: cfg, connRepo: connRepo, client: http.DefaultClient}
}

// ConnectURL generates the Salesforce OAuth authorization URL.
func (s *SalesforceOAuthService) ConnectURL(orgID uuid.UUID) (string, error) {
	if s.cfg.ClientID == "" {
		return "", &ValidationError{Field: providerSalesforce, Message: "Salesforce integration is not configured"}
	}
	state := fmt.Sprintf("%s:%d", orgID.String(), time.Now().UnixNano())
	params := url.Values{
		"response_type": {"code"},
		"client_id":     {s.cfg.ClientID},
		"redirect_uri":  {s.cfg.OAuthRedirectURL},
		"scope":         {"api refresh_token"},
		"state":         {state},
	}
	return strings.TrimRight(s.cfg.LoginURL, "/") + "/services/oauth2/authorize?" + params.Encode(), nil
}

// ExchangeCode exchanges the OAuth code for tokens and stores the connection.
func (s *SalesforceOAuthService) ExchangeCode(ctx context.Context, orgID uuid.UUID, code, state string) error {
	if code == "" {
		return &ValidationError{Field: "code", Message: "authorization code is required"}
	}
	if err := validateOAuthState(orgID, state); err != nil {
		return err
	}

	tokenResp, err := s.exchangeCode(ctx, code)
	if err != nil {
		return fmt.Errorf("exchange code with salesforce: %w", err)
	}

	accessToken, err := encryptToken(tokenResp.AccessToken, s.cfg.EncryptionKey)
	if err != nil {
		return fmt.Errorf("encrypt access token: %w", err)
	}
	refreshToken, err := encryptToken(tokenResp.RefreshToken, s.cfg.EncryptionKey)
	if err != nil {
		return fmt.Errorf("encrypt refresh token: %w", err)
	}

	conn := &repository.IntegrationConnection{
		OrgID:                 orgID,
		Provider:              providerSalesforce,
		Status:                "active",
		AccessTokenEncrypted:  accessToken,
		RefreshTokenEncrypted: refreshToken,
		ExternalAccountID:     tokenResp.ID,
		Scopes:                []string{"api", "refresh_token"},
		Metadata: map[string]any{
			"instance_url": tokenResp.InstanceURL,
			"token_type":   tokenResp.TokenType,
			"issued_at":    tokenResp.IssuedAt,
		},
	}
	if err := s.connRepo.Upsert(ctx, conn); err != nil {
		return fmt.Errorf("store connection: %w", err)
	}

	slog.Info("salesforce connection established", "org_id", orgID)
	return nil
}

// GetAccess retrieves the decrypted token and Salesforce instance URL.
func (s *SalesforceOAuthService) GetAccess(ctx context.Context, orgID uuid.UUID) (*SalesforceAccess, error) {
	conn, err := s.connRepo.GetByOrgAndProvider(ctx, orgID, providerSalesforce)
	if err != nil {
		return nil, fmt.Errorf("get connection: %w", err)
	}
	if conn == nil {
		return nil, &NotFoundError{Resource: "salesforce_connection", Message: "no Salesforce connection found"}
	}
	if conn.Status != "active" && conn.Status != "syncing" {
		return nil, &ValidationError{Field: providerSalesforce, Message: "Salesforce connection is not active"}
	}
	token, err := decryptToken(conn.AccessTokenEncrypted, s.cfg.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt access token: %w", err)
	}
	instanceURL, _ := conn.Metadata["instance_url"].(string)
	if instanceURL == "" {
		return nil, &ValidationError{Field: "instance_url", Message: "Salesforce instance URL is missing"}
	}
	return &SalesforceAccess{AccessToken: token, InstanceURL: instanceURL}, nil
}

// GetStatus returns the current Salesforce connection status for an org.
func (s *SalesforceOAuthService) GetStatus(ctx context.Context, orgID uuid.UUID) (*SalesforceConnectionStatus, error) {
	conn, err := s.connRepo.GetByOrgAndProvider(ctx, orgID, providerSalesforce)
	if err != nil {
		return nil, fmt.Errorf("get connection: %w", err)
	}
	if conn == nil {
		return &SalesforceConnectionStatus{Status: "disconnected"}, nil
	}
	instanceURL, _ := conn.Metadata["instance_url"].(string)
	return &SalesforceConnectionStatus{
		Status:            conn.Status,
		ExternalAccountID: conn.ExternalAccountID,
		InstanceURL:       instanceURL,
		LastSyncAt:        conn.LastSyncAt,
		LastSyncError:     conn.LastSyncError,
		ConnectedAt:       conn.CreatedAt,
	}, nil
}

// Disconnect removes a Salesforce connection.
func (s *SalesforceOAuthService) Disconnect(ctx context.Context, orgID uuid.UUID) error {
	return s.connRepo.Delete(ctx, orgID, providerSalesforce)
}

// SalesforceConnectionStatus holds Salesforce status data for frontend display.
type SalesforceConnectionStatus struct {
	Status            string     `json:"status"`
	ExternalAccountID string     `json:"external_account_id,omitempty"`
	InstanceURL       string     `json:"instance_url,omitempty"`
	LastSyncAt        *time.Time `json:"last_sync_at,omitempty"`
	LastSyncError     string     `json:"last_sync_error,omitempty"`
	ConnectedAt       time.Time  `json:"connected_at,omitempty"`
}

type salesforceTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	InstanceURL  string `json:"instance_url"`
	ID           string `json:"id"`
	IssuedAt     string `json:"issued_at"`
	TokenType    string `json:"token_type"`
}

func (s *SalesforceOAuthService) exchangeCode(ctx context.Context, code string) (*salesforceTokenResponse, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {s.cfg.ClientID},
		"client_secret": {s.cfg.ClientSecret},
		"redirect_uri":  {s.cfg.OAuthRedirectURL},
	}
	endpoint := strings.TrimRight(s.cfg.LoginURL, "/") + "/services/oauth2/token"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("salesforce token exchange failed with status %d", resp.StatusCode)
	}

	var tokenResp salesforceTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}
	if tokenResp.AccessToken == "" || tokenResp.InstanceURL == "" {
		return nil, &ValidationError{Field: providerSalesforce, Message: "Salesforce token response missing access token or instance URL"}
	}
	return &tokenResp, nil
}

func validateOAuthState(orgID uuid.UUID, state string) error {
	parts := strings.SplitN(state, ":", 2)
	if len(parts) != 2 {
		return &ValidationError{Field: "state", Message: "invalid state parameter"}
	}
	stateOrgID, err := uuid.Parse(parts[0])
	if err != nil || stateOrgID != orgID {
		return &ValidationError{Field: "state", Message: "invalid state parameter"}
	}
	return nil
}
