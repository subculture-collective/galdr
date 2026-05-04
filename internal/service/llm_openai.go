package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const openAIBaseURL = "https://api.openai.com"

// OpenAIConfig holds OpenAI provider settings.
type OpenAIConfig struct {
	APIKey       string
	Model        string
	MaxRetries   int
	RetryBackoff time.Duration
	BaseURL      string
	HTTPClient   *http.Client
}

// OpenAIProvider implements LLMProvider using OpenAI chat completions.
type OpenAIProvider struct {
	apiKey       string
	model        string
	maxRetries   int
	retryBackoff time.Duration
	baseURL      string
	client       *http.Client
}

// NewOpenAIProvider creates an OpenAI-backed LLM provider.
func NewOpenAIProvider(cfg OpenAIConfig) *OpenAIProvider {
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = defaultLLMModel
	}
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	if baseURL == "" {
		baseURL = openAIBaseURL
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	retryBackoff := cfg.RetryBackoff
	if retryBackoff <= 0 {
		retryBackoff = 100 * time.Millisecond
	}
	maxRetries := cfg.MaxRetries
	if maxRetries < 0 {
		maxRetries = 0
	}
	if maxRetries == 0 {
		maxRetries = 2
	}

	return &OpenAIProvider{
		apiKey:       cfg.APIKey,
		model:        model,
		maxRetries:   maxRetries,
		retryBackoff: retryBackoff,
		baseURL:      baseURL,
		client:       client,
	}
}

// Name returns the provider identifier.
func (p *OpenAIProvider) Name() string { return "openai" }

// CountTokens estimates tokens before the API request.
func (p *OpenAIProvider) CountTokens(text string) int {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return 0
	}
	tokens := len([]rune(trimmed)) / 4
	if tokens == 0 {
		return 1
	}
	return tokens
}

// Complete sends a chat completion request to OpenAI and retries rate-limit responses.
func (p *OpenAIProvider) Complete(ctx context.Context, req LLMProviderRequest) (*LLMProviderResponse, error) {
	if strings.TrimSpace(p.apiKey) == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is required")
	}

	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = p.model
	}
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 512
	}

	payload := openAIChatCompletionRequest{
		Model: model,
		Messages: []openAIChatMessage{{
			Role:    "user",
			Content: req.Prompt,
		}},
		MaxTokens: maxTokens,
	}

	var lastErr error
	for attempt := 0; attempt <= p.maxRetries; attempt++ {
		resp, err := p.doComplete(ctx, payload)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		var rateLimitErr *openAIRateLimitError
		if !errors.As(err, &rateLimitErr) || attempt == p.maxRetries {
			break
		}
		if err := sleepWithContext(ctx, retryDelay(rateLimitErr.retryAfter, p.retryBackoff, attempt)); err != nil {
			return nil, err
		}
	}
	return nil, lastErr
}

func (p *OpenAIProvider) doComplete(ctx context.Context, payload openAIChatCompletionRequest) (*LLMProviderResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal openai request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create openai request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openai http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read openai response: %w", err)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		return nil, &openAIRateLimitError{retryAfter: parseRetryAfter(resp.Header.Get("Retry-After")), body: string(respBody)}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openai api error: status %d, body: %s", resp.StatusCode, string(respBody))
	}

	var parsed openAIChatCompletionResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("parse openai response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("openai response has no choices")
	}

	return &LLMProviderResponse{
		Text:         parsed.Choices[0].Message.Content,
		Model:        parsed.Model,
		InputTokens:  parsed.Usage.PromptTokens,
		OutputTokens: parsed.Usage.CompletionTokens,
	}, nil
}

type openAIChatCompletionRequest struct {
	Model     string              `json:"model"`
	Messages  []openAIChatMessage `json:"messages"`
	MaxTokens int                 `json:"max_tokens"`
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatCompletionResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message openAIChatMessage `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

type openAIRateLimitError struct {
	retryAfter time.Duration
	body       string
}

func (e *openAIRateLimitError) Error() string {
	return "openai rate limited: " + e.body
}

func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}
	seconds, err := strconv.Atoi(value)
	if err != nil || seconds < 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}

func retryDelay(retryAfter, backoff time.Duration, attempt int) time.Duration {
	if retryAfter > 0 {
		return retryAfter
	}
	if attempt <= 0 {
		return backoff
	}
	return backoff * time.Duration(1<<attempt)
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return fmt.Errorf("retry wait cancelled: %w", ctx.Err())
	case <-timer.C:
		return nil
	}
}
