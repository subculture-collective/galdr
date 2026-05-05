package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/onnwee/pulse-score/internal/repository"
)

const (
	defaultWebhookTimeout = 10 * time.Second
	defaultWebhookAttempts = 3
	webhookSignatureHeader = "X-PulseScore-Signature"

	webhookConfigURL                 = "url"
	webhookConfigHeaders             = "headers"
	webhookConfigIncludeCustomerData = "include_customer_data"
	webhookConfigSigningSecret       = "signing_secret"
)

// ErrWebhookURLMustBeHTTPS is returned when an action uses an unsafe webhook URL.
var ErrWebhookURLMustBeHTTPS = errors.New("webhook url must be https")

// WebhookActionExecutorConfig controls webhook delivery behavior.
type WebhookActionExecutorConfig struct {
	HTTPClient  *http.Client
	RetryDelays []time.Duration
	Now         func() time.Time
}

// WebhookActionExecutor sends playbook webhook actions.
type WebhookActionExecutor struct {
	httpClient  *http.Client
	retryDelays []time.Duration
	now         func() time.Time
}

// WebhookActionRequest is the public input for executing one webhook action.
type WebhookActionRequest struct {
	Playbook     *repository.Playbook
	Action       *repository.PlaybookAction
	Customer     *repository.Customer
	TriggerEvent map[string]any
}

// WebhookActionResult captures delivery details for playbook execution history.
type WebhookActionResult struct {
	StatusCode int    `json:"status_code"`
	Attempts   int    `json:"attempts"`
	LatencyMS  int64  `json:"latency_ms"`
	Error      string `json:"error,omitempty"`
}

type webhookActionConfig struct {
	URL                 string
	Headers             map[string]string
	IncludeCustomerData bool
	SigningSecret       string
}

// NewWebhookActionExecutor creates a webhook action executor.
func NewWebhookActionExecutor(cfg WebhookActionExecutorConfig) *WebhookActionExecutor {
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: defaultWebhookTimeout}
	}
	if client.Timeout == 0 {
		copyClient := *client
		copyClient.Timeout = defaultWebhookTimeout
		client = &copyClient
	}
	retryDelays := cfg.RetryDelays
	if retryDelays == nil {
		retryDelays = []time.Duration{time.Second, 2 * time.Second}
	}
	now := cfg.Now
	if now == nil {
		now = time.Now
	}
	return &WebhookActionExecutor{httpClient: client, retryDelays: retryDelays, now: now}
}

// Execute sends the configured webhook and returns delivery metadata.
func (e *WebhookActionExecutor) Execute(ctx context.Context, req WebhookActionRequest) (*WebhookActionResult, error) {
	if req.Playbook == nil {
		return nil, errors.New("playbook is required")
	}
	if req.Action == nil {
		return nil, errors.New("action is required")
	}
	if req.Action.ActionType != repository.PlaybookActionWebhook {
		return nil, fmt.Errorf("webhook executor cannot run %q actions", req.Action.ActionType)
	}
	cfg, err := parseWebhookActionConfig(req.Action.ActionConfig)
	if err != nil {
		return nil, err
	}
	if err := validateWebhookURL(cfg.URL); err != nil {
		return nil, err
	}

	payload, err := json.Marshal(e.webhookPayload(req, cfg.IncludeCustomerData))
	if err != nil {
		return nil, fmt.Errorf("marshal webhook payload: %w", err)
	}

	var lastResult *WebhookActionResult
	var lastErr error
	for attempt := 1; attempt <= defaultWebhookAttempts; attempt++ {
		result, err := e.send(ctx, cfg, payload, attempt)
		lastResult = result
		lastErr = err
		if err == nil && isSuccessfulWebhookStatus(result.StatusCode) {
			return result, nil
		}
		if err == nil && !isRetryableWebhookStatus(result.StatusCode) {
			break
		}
		if attempt == defaultWebhookAttempts {
			break
		}
		delay := e.retryDelay(attempt)
		if delay > 0 {
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return lastResult, ctx.Err()
			}
		}
	}
	if lastErr != nil {
		return lastResult, lastErr
	}
	lastResult.Error = fmt.Sprintf("webhook returned status %d", lastResult.StatusCode)
	return lastResult, errors.New(lastResult.Error)
}

func (e *WebhookActionExecutor) send(ctx context.Context, cfg webhookActionConfig, payload []byte, attempt int) (*WebhookActionResult, error) {
	started := e.now()
	httpReq, err := newWebhookHTTPRequest(ctx, cfg, payload)
	if err != nil {
		return nil, err
	}

	res, err := e.httpClient.Do(httpReq)
	latency := e.now().Sub(started).Milliseconds()
	result := &WebhookActionResult{Attempts: attempt, LatencyMS: latency}
	if err != nil {
		result.Error = err.Error()
		return result, err
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, res.Body)
	result.StatusCode = res.StatusCode
	return result, nil
}

func newWebhookHTTPRequest(ctx context.Context, cfg webhookActionConfig, payload []byte) (*http.Request, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.URL, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create webhook request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	for name, value := range cfg.Headers {
		httpReq.Header.Set(name, value)
	}
	httpReq.Header.Set(webhookSignatureHeader, signWebhookPayload(cfg.SigningSecret, payload))
	return httpReq, nil
}

func (e *WebhookActionExecutor) webhookPayload(req WebhookActionRequest, includeCustomer bool) map[string]any {
	payload := map[string]any{
		"trigger_event": req.TriggerEvent,
		"playbook": map[string]any{
			"id":           req.Playbook.ID.String(),
			"name":         req.Playbook.Name,
			"trigger_type": req.Playbook.TriggerType,
		},
		"action": map[string]any{
			"id":   req.Action.ID.String(),
			"type": req.Action.ActionType,
		},
		"sent_at": e.now().UTC().Format(time.RFC3339Nano),
	}
	if includeCustomer && req.Customer != nil {
		payload["customer"] = map[string]any{
			"id":           req.Customer.ID.String(),
			"email":        req.Customer.Email,
			"name":         req.Customer.Name,
			"company_name": req.Customer.CompanyName,
			"mrr_cents":    req.Customer.MRRCents,
			"source":       req.Customer.Source,
		}
	}
	return payload
}

func (e *WebhookActionExecutor) retryDelay(attempt int) time.Duration {
	idx := attempt - 1
	if idx >= 0 && idx < len(e.retryDelays) {
		return e.retryDelays[idx]
	}
	return 0
}

func parseWebhookActionConfig(values map[string]any) (webhookActionConfig, error) {
	cfg := webhookActionConfig{Headers: parseWebhookHeaders(values)}
	cfg.URL = stringConfigValue(values, webhookConfigURL)
	if cfg.URL == "" {
		return cfg, errors.New("webhook url is required")
	}
	cfg.IncludeCustomerData = boolConfigValue(values, webhookConfigIncludeCustomerData)
	cfg.SigningSecret = stringConfigValue(values, webhookConfigSigningSecret)
	if cfg.SigningSecret == "" {
		return cfg, errors.New("webhook signing_secret is required")
	}
	return cfg, nil
}

func parseWebhookHeaders(values map[string]any) map[string]string {
	headers := make(map[string]string)
	if values == nil {
		return headers
	}
	switch rawHeaders := values[webhookConfigHeaders].(type) {
	case map[string]any:
		for name, value := range rawHeaders {
			headers[name] = fmt.Sprint(value)
		}
	case map[string]string:
		for name, value := range rawHeaders {
			headers[name] = value
		}
	}
	return headers
}

func validateWebhookURL(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("parse webhook url: %w", err)
	}
	if parsed.Scheme != "https" || parsed.Host == "" {
		return ErrWebhookURLMustBeHTTPS
	}
	return nil
}

func signWebhookPayload(secret string, payload []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func isRetryableWebhookStatus(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || statusCode >= 500
}

func isSuccessfulWebhookStatus(statusCode int) bool {
	return statusCode >= 200 && statusCode < 300
}

func stringConfigValue(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	switch value := values[key].(type) {
	case string:
		return value
	case fmt.Stringer:
		return value.String()
	default:
		return ""
	}
}

func boolConfigValue(values map[string]any, key string) bool {
	if values == nil {
		return false
	}
	switch value := values[key].(type) {
	case bool:
		return value
	case string:
		parsed, _ := strconv.ParseBool(value)
		return parsed
	default:
		return false
	}
}
