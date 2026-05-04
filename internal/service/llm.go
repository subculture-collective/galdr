package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	defaultLLMModel             = "gpt-4o-mini"
	defaultLLMMaxTokens         = 512
	defaultLLMRequestsPerMinute = 60
	defaultLLMMaxTokensPerDay   = 100_000
	llmCostTokenDivisor         = 1_000_000
	gpt4oMiniInputPerMillion    = 0.15
	gpt4oMiniOutputPerMillion   = 0.60
)

var (
	// ErrLLMRateLimited is returned when an organization exceeds its request budget.
	ErrLLMRateLimited = errors.New("llm request rate limit exceeded")
	// ErrLLMTokenBudgetExceeded is returned when an organization exceeds its daily token budget.
	ErrLLMTokenBudgetExceeded = errors.New("llm daily token budget exceeded")
)

// LLMProvider describes a provider-specific completion backend.
type LLMProvider interface {
	Complete(ctx context.Context, req LLMProviderRequest) (*LLMProviderResponse, error)
	CountTokens(text string) int
	Name() string
}

// LLMProviderRequest is the provider-neutral completion request.
type LLMProviderRequest struct {
	Prompt    string
	Model     string
	MaxTokens int
}

// LLMProviderResponse is the provider-neutral completion response.
type LLMProviderResponse struct {
	Text         string
	Model        string
	InputTokens  int
	OutputTokens int
}

// LLMServiceConfig controls prompt rendering and per-org budgets.
type LLMServiceConfig struct {
	Model             string
	MaxTokens         int
	RequestsPerMinute int
	MaxTokensPerDay   int
	Templates         map[string]string
	FallbackProvider  LLMProvider
}

// LLMCompletionRequest describes an org-scoped completion request.
type LLMCompletionRequest struct {
	OrgID        uuid.UUID
	Prompt       string
	TemplateName string
	TemplateData map[string]string
	MaxTokens    int
}

// LLMCompletionResponse contains generated text and usage/cost metadata.
type LLMCompletionResponse struct {
	Text  string
	Usage LLMUsage
}

// LLMUsage records token and cost metadata for a single request.
type LLMUsage struct {
	OrgID        uuid.UUID `json:"org_id"`
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	TotalTokens  int       `json:"total_tokens"`
	CostUSD      float64   `json:"cost_usd"`
	Latency      time.Duration
	Timestamp    time.Time `json:"ts"`
}

// LLMService coordinates prompt templates, org budgets, provider calls, and usage tracking.
type LLMService struct {
	provider LLMProvider
	fallback LLMProvider
	model    string
	maxTokens int

	templates map[string]string
	requests  *orgRequestLimiter
	tokens    *orgTokenBudget
}

// NewLLMService creates an LLM service with per-org rate and token limits.
func NewLLMService(provider LLMProvider, cfg LLMServiceConfig) *LLMService {
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = defaultLLMModel
	}
	maxTokens := cfg.MaxTokens
	if maxTokens <= 0 {
		maxTokens = defaultLLMMaxTokens
	}

	return &LLMService{
		provider:  provider,
		fallback:  cfg.FallbackProvider,
		model:     model,
		maxTokens: maxTokens,
		templates: cloneTemplates(cfg.Templates),
		requests:  newOrgRequestLimiter(cfg.RequestsPerMinute),
		tokens:    newOrgTokenBudget(cfg.MaxTokensPerDay),
	}
}

// Complete renders the prompt, enforces org limits, calls the provider, and returns usage metadata.
func (s *LLMService) Complete(ctx context.Context, req LLMCompletionRequest) (*LLMCompletionResponse, error) {
	if req.OrgID == uuid.Nil {
		return nil, fmt.Errorf("org id is required")
	}
	if !s.requests.Allow(req.OrgID, time.Now()) {
		return nil, ErrLLMRateLimited
	}

	prompt, err := s.renderPrompt(req)
	if err != nil {
		return nil, err
	}
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = s.maxTokens
	}
	estimatedTokens := s.provider.CountTokens(prompt) + maxTokens
	if !s.tokens.Consume(req.OrgID, estimatedTokens, time.Now()) {
		return nil, ErrLLMTokenBudgetExceeded
	}

	started := time.Now()
	providerResp, providerName, err := s.completeWithFallback(ctx, LLMProviderRequest{
		Prompt:    prompt,
		Model:     s.model,
		MaxTokens: maxTokens,
	})
	if err != nil {
		return nil, fmt.Errorf("llm provider complete: %w", err)
	}

	usage := LLMUsage{
		OrgID:        req.OrgID,
		Provider:     providerName,
		Model:        providerResp.Model,
		InputTokens:  providerResp.InputTokens,
		OutputTokens: providerResp.OutputTokens,
		TotalTokens:  providerResp.InputTokens + providerResp.OutputTokens,
		Latency:      time.Since(started),
		Timestamp:    time.Now().UTC(),
	}
	usage.CostUSD = calculateLLMCostUSD(usage.Model, usage.InputTokens, usage.OutputTokens)

	slog.Info("llm request complete",
		"org_id", usage.OrgID,
		"provider", usage.Provider,
		"model", usage.Model,
		"input_tokens", usage.InputTokens,
		"output_tokens", usage.OutputTokens,
		"cost_usd", usage.CostUSD,
		"latency", usage.Latency,
	)

	return &LLMCompletionResponse{Text: providerResp.Text, Usage: usage}, nil
}

func (s *LLMService) completeWithFallback(ctx context.Context, req LLMProviderRequest) (*LLMProviderResponse, string, error) {
	resp, err := s.provider.Complete(ctx, req)
	if err == nil {
		return resp, s.provider.Name(), nil
	}
	if s.fallback == nil {
		return nil, "", err
	}

	fallbackResp, fallbackErr := s.fallback.Complete(ctx, req)
	if fallbackErr != nil {
		return nil, "", fmt.Errorf("primary provider failed: %v; fallback provider failed: %w", err, fallbackErr)
	}
	return fallbackResp, s.fallback.Name(), nil
}

func (s *LLMService) renderPrompt(req LLMCompletionRequest) (string, error) {
	if strings.TrimSpace(req.TemplateName) == "" {
		if strings.TrimSpace(req.Prompt) == "" {
			return "", fmt.Errorf("prompt or template name is required")
		}
		return req.Prompt, nil
	}

	template, ok := s.templates[req.TemplateName]
	if !ok {
		return "", fmt.Errorf("unknown prompt template %q", req.TemplateName)
	}
	for key, value := range req.TemplateData {
		template = strings.ReplaceAll(template, "{{"+key+"}}", value)
	}
	if strings.Contains(template, "{{") || strings.Contains(template, "}}") {
		return "", fmt.Errorf("prompt template %q has unresolved variables", req.TemplateName)
	}
	return template, nil
}

func cloneTemplates(templates map[string]string) map[string]string {
	cloned := make(map[string]string, len(templates))
	for key, value := range templates {
		cloned[key] = value
	}
	return cloned
}

func calculateLLMCostUSD(model string, inputTokens, outputTokens int) float64 {
	return (float64(inputTokens)*gpt4oMiniInputPerMillion + float64(outputTokens)*gpt4oMiniOutputPerMillion) / llmCostTokenDivisor
}

type orgRequestLimiter struct {
	mu       sync.Mutex
	limit    int
	requests map[uuid.UUID]requestWindow
}

type requestWindow struct {
	startedAt time.Time
	count     int
}

func newOrgRequestLimiter(limit int) *orgRequestLimiter {
	if limit <= 0 {
		limit = defaultLLMRequestsPerMinute
	}
	return &orgRequestLimiter{limit: limit, requests: make(map[uuid.UUID]requestWindow)}
}

func (l *orgRequestLimiter) Allow(orgID uuid.UUID, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	window := l.requests[orgID]
	if window.startedAt.IsZero() || now.Sub(window.startedAt) >= time.Minute {
		l.requests[orgID] = requestWindow{startedAt: now, count: 1}
		return true
	}
	if window.count >= l.limit {
		return false
	}
	window.count++
	l.requests[orgID] = window
	return true
}

type orgTokenBudget struct {
	mu    sync.Mutex
	limit int
	usage map[uuid.UUID]tokenWindow
}

type tokenWindow struct {
	day  time.Time
	used int
}

func newOrgTokenBudget(limit int) *orgTokenBudget {
	if limit <= 0 {
		limit = defaultLLMMaxTokensPerDay
	}
	return &orgTokenBudget{limit: limit, usage: make(map[uuid.UUID]tokenWindow)}
}

func (b *orgTokenBudget) Consume(orgID uuid.UUID, tokens int, now time.Time) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	window := b.usage[orgID]
	if window.day.IsZero() || !window.day.Equal(day) {
		window = tokenWindow{day: day}
	}
	if window.used+tokens > b.limit {
		return false
	}
	window.used += tokens
	b.usage[orgID] = window
	return true
}
