package service

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"text/template"
	"time"

	"github.com/google/uuid"
	"github.com/onnwee/pulse-score/internal/service/prompts"
	"golang.org/x/time/rate"
)

const (
	defaultLLMRequestsPerMinute = 60
	defaultLLMMaxTokensPerDay   = 100_000
	defaultLLMMaxTokens         = 1_000

	gpt4oMiniInputCostPer1M  = 0.15
	gpt4oMiniOutputCostPer1M = 0.60
)

// LLMProvider completes prompts through a concrete model provider.
type LLMProvider interface {
	Complete(ctx context.Context, req LLMProviderRequest) (*LLMProviderResponse, error)
	CountTokens(text string) int
	Name() string
}

// LLMUsageTracker persists or emits per-request LLM usage.
type LLMUsageTracker interface {
	TrackLLMUsage(ctx context.Context, usage LLMUsage) error
}

// LLMProviderRequest is the provider-facing completion request.
type LLMProviderRequest struct {
	Prompt    string
	MaxTokens int
}

// LLMProviderResponse is the provider-facing completion response.
type LLMProviderResponse struct {
	Text         string
	InputTokens  int
	OutputTokens int
}

// LLMCompletionRequest is the service-facing completion request.
type LLMCompletionRequest struct {
	OrgID        uuid.UUID
	Prompt       string
	TemplateName string
	TemplateData any
	MaxTokens    int
}

// LLMCompletionResponse is the service-facing completion response.
type LLMCompletionResponse struct {
	Text         string
	Prompt       string
	Provider     string
	InputTokens  int
	OutputTokens int
	CostUSD      float64
}

// LLMUsage captures request cost and token consumption.
type LLMUsage struct {
	OrgID        uuid.UUID
	Provider     string
	InputTokens  int
	OutputTokens int
	CostUSD      float64
	Latency      time.Duration
	CreatedAt    time.Time
}

// LLMServiceConfig controls limits, retries, and prompt templates.
type LLMServiceConfig struct {
	RequestsPerMinute int
	MaxTokensPerDay   int
	DefaultMaxTokens  int
	Templates         map[string]string
	RetryDelays       []time.Duration
	FallbackProviders []LLMProvider
}

// LLMService orchestrates provider calls with per-org safety controls.
type LLMService struct {
	providers []LLMProvider
	tracker   LLMUsageTracker
	cfg       LLMServiceConfig

	mu          sync.Mutex
	limiters    map[uuid.UUID]*rate.Limiter
	dailyTokens map[llmTokenBudgetKey]int
}

type llmTokenBudgetKey struct {
	orgID uuid.UUID
	day   string
}

// NewLLMService creates an LLM service around any provider implementation.
func NewLLMService(provider LLMProvider, tracker LLMUsageTracker, cfg LLMServiceConfig) *LLMService {
	if cfg.RequestsPerMinute <= 0 {
		cfg.RequestsPerMinute = defaultLLMRequestsPerMinute
	}
	if cfg.MaxTokensPerDay <= 0 {
		cfg.MaxTokensPerDay = defaultLLMMaxTokensPerDay
	}
	if cfg.DefaultMaxTokens <= 0 {
		cfg.DefaultMaxTokens = defaultLLMMaxTokens
	}
	if len(cfg.RetryDelays) == 0 {
		cfg.RetryDelays = []time.Duration{time.Second, 2 * time.Second, 4 * time.Second}
	}
	if cfg.Templates == nil {
		if templates, err := prompts.Templates(); err == nil {
			cfg.Templates = templates
		}
	}

	providers := make([]LLMProvider, 0, 1+len(cfg.FallbackProviders))
	if provider != nil {
		providers = append(providers, provider)
	}
	for _, fallback := range cfg.FallbackProviders {
		if fallback != nil {
			providers = append(providers, fallback)
		}
	}

	return &LLMService{
		providers:   providers,
		tracker:     tracker,
		cfg:         cfg,
		limiters:    make(map[uuid.UUID]*rate.Limiter),
		dailyTokens: make(map[llmTokenBudgetKey]int),
	}
}

// Complete renders the prompt, enforces org limits, calls the provider, and tracks cost.
func (s *LLMService) Complete(ctx context.Context, req LLMCompletionRequest) (*LLMCompletionResponse, error) {
	if len(s.providers) == 0 {
		return nil, errors.New("llm provider is required")
	}
	if req.OrgID == uuid.Nil {
		return nil, errors.New("org id is required")
	}

	prompt, err := s.renderPrompt(req)
	if err != nil {
		return nil, err
	}
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = s.cfg.DefaultMaxTokens
	}

	estimatedTokens := s.providers[0].CountTokens(prompt) + maxTokens
	if err := s.reserveTokens(req.OrgID, estimatedTokens); err != nil {
		return nil, err
	}
	if !s.allowRequest(req.OrgID) {
		s.releaseTokenReservation(req.OrgID, estimatedTokens)
		return nil, ErrLLMRateLimited
	}

	start := time.Now()
	providerRes, providerName, err := s.completeWithFallback(ctx, LLMProviderRequest{Prompt: prompt, MaxTokens: maxTokens})
	if err != nil {
		s.releaseTokenReservation(req.OrgID, estimatedTokens)
		return nil, err
	}

	actualTokens := providerRes.InputTokens + providerRes.OutputTokens
	s.adjustTokenReservation(req.OrgID, estimatedTokens, actualTokens)
	cost := CalculateLLMCostUSD(providerRes.InputTokens, providerRes.OutputTokens)
	res := &LLMCompletionResponse{
		Text:         providerRes.Text,
		Prompt:       prompt,
		Provider:     providerName,
		InputTokens:  providerRes.InputTokens,
		OutputTokens: providerRes.OutputTokens,
		CostUSD:      cost,
	}

	usage := LLMUsage{
		OrgID:        req.OrgID,
		Provider:     providerName,
		InputTokens:  providerRes.InputTokens,
		OutputTokens: providerRes.OutputTokens,
		CostUSD:      cost,
		Latency:      time.Since(start),
		CreatedAt:    time.Now().UTC(),
	}
	if s.tracker != nil {
		if err := s.tracker.TrackLLMUsage(ctx, usage); err != nil {
			return nil, fmt.Errorf("track llm usage: %w", err)
		}
	} else {
		slog.Info("llm usage", "org_id", usage.OrgID, "provider", usage.Provider, "input_tokens", usage.InputTokens, "output_tokens", usage.OutputTokens, "cost_usd", usage.CostUSD)
	}

	return res, nil
}

// CalculateLLMCostUSD calculates GPT-4o-mini cost from token usage.
func CalculateLLMCostUSD(inputTokens, outputTokens int) float64 {
	inputCost := (float64(inputTokens) / 1_000_000) * gpt4oMiniInputCostPer1M
	outputCost := (float64(outputTokens) / 1_000_000) * gpt4oMiniOutputCostPer1M
	return inputCost + outputCost
}

func (s *LLMService) renderPrompt(req LLMCompletionRequest) (string, error) {
	if req.TemplateName == "" {
		if req.Prompt == "" {
			return "", errors.New("prompt or template name is required")
		}
		return req.Prompt, nil
	}

	templateText, ok := s.cfg.Templates[req.TemplateName]
	if !ok {
		return "", fmt.Errorf("unknown llm template %q", req.TemplateName)
	}
	tmpl, err := template.New(req.TemplateName).Option("missingkey=error").Funcs(prompts.FuncMap()).Parse(templateText)
	if err != nil {
		return "", fmt.Errorf("parse llm template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, req.TemplateData); err != nil {
		return "", fmt.Errorf("render llm template: %w", err)
	}
	return buf.String(), nil
}

func (s *LLMService) allowRequest(orgID uuid.UUID) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	limiter, ok := s.limiters[orgID]
	if !ok {
		limiter = rate.NewLimiter(rate.Every(time.Minute/time.Duration(s.cfg.RequestsPerMinute)), s.cfg.RequestsPerMinute)
		s.limiters[orgID] = limiter
	}
	return limiter.Allow()
}

func (s *LLMService) reserveTokens(orgID uuid.UUID, tokens int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := llmTokenBudgetKey{orgID: orgID, day: time.Now().UTC().Format(time.DateOnly)}
	if s.dailyTokens[key]+tokens > s.cfg.MaxTokensPerDay {
		return ErrLLMTokenBudgetExceeded
	}
	s.dailyTokens[key] += tokens
	return nil
}

func (s *LLMService) releaseTokenReservation(orgID uuid.UUID, tokens int) {
	s.adjustTokenReservation(orgID, tokens, 0)
}

func (s *LLMService) adjustTokenReservation(orgID uuid.UUID, estimatedTokens, actualTokens int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := llmTokenBudgetKey{orgID: orgID, day: time.Now().UTC().Format(time.DateOnly)}
	s.dailyTokens[key] += actualTokens - estimatedTokens
	if s.dailyTokens[key] < 0 {
		s.dailyTokens[key] = 0
	}
}

func (s *LLMService) completeWithFallback(ctx context.Context, req LLMProviderRequest) (*LLMProviderResponse, string, error) {
	var lastErr error
	for _, provider := range s.providers {
		res, err := s.completeWithRetry(ctx, provider, req)
		if err == nil {
			return res, provider.Name(), nil
		}
		lastErr = err
		if ctx.Err() != nil {
			return nil, "", err
		}
	}
	return nil, "", lastErr
}

func (s *LLMService) completeWithRetry(ctx context.Context, provider LLMProvider, req LLMProviderRequest) (*LLMProviderResponse, error) {
	var lastErr error
	for attempt := 0; attempt <= len(s.cfg.RetryDelays); attempt++ {
		res, err := provider.Complete(ctx, req)
		if err == nil {
			return res, nil
		}
		lastErr = err

		var providerErr *LLMProviderError
		if !errors.As(err, &providerErr) || providerErr.StatusCode != http.StatusTooManyRequests || attempt == len(s.cfg.RetryDelays) {
			return nil, err
		}

		delay := s.cfg.RetryDelays[attempt]
		if providerErr.RetryAfter > 0 {
			delay = providerErr.RetryAfter
		}
		if err := waitLLMRetry(ctx, delay); err != nil {
			return nil, err
		}
	}
	return nil, lastErr
}

func waitLLMRetry(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// LLMProviderError represents a retry-aware provider error.
type LLMProviderError struct {
	StatusCode int
	Message    string
	RetryAfter time.Duration
}

func (e *LLMProviderError) Error() string {
	return fmt.Sprintf("llm provider status %d: %s", e.StatusCode, e.Message)
}

var (
	// ErrLLMRateLimited means the org exceeded its per-minute request limit.
	ErrLLMRateLimited = errors.New("llm rate limit exceeded")
	// ErrLLMTokenBudgetExceeded means the org exceeded its daily token budget.
	ErrLLMTokenBudgetExceeded = errors.New("llm daily token budget exceeded")
)
