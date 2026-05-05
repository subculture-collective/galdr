package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	defaultOpenAIBaseURL = "https://api.openai.com/v1"
	defaultOpenAIModel   = "gpt-4o-mini"
)

// OpenAIProviderConfig holds OpenAI provider settings.
type OpenAIProviderConfig struct {
	APIKey    string
	Model     string
	BaseURL   string
	MaxTokens int
	Client    *http.Client
}

// OpenAIProvider implements LLMProvider using OpenAI chat completions.
type OpenAIProvider struct {
	cfg    OpenAIProviderConfig
	client *http.Client
}

// NewOpenAIProvider creates an OpenAI-backed LLM provider.
func NewOpenAIProvider(cfg OpenAIProviderConfig) *OpenAIProvider {
	if cfg.Model == "" {
		cfg.Model = defaultOpenAIModel
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaultOpenAIBaseURL
	}
	if cfg.MaxTokens <= 0 {
		cfg.MaxTokens = defaultLLMMaxTokens
	}
	client := cfg.Client
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &OpenAIProvider{cfg: cfg, client: client}
}

// NewOpenAILLMService wires the OpenAI provider into the LLM service.
func NewOpenAILLMService(openAICfg OpenAIProviderConfig, tracker LLMUsageTracker, svcCfg LLMServiceConfig) *LLMService {
	if svcCfg.DefaultMaxTokens <= 0 && openAICfg.MaxTokens > 0 {
		svcCfg.DefaultMaxTokens = openAICfg.MaxTokens
	}
	return NewLLMService(NewOpenAIProvider(openAICfg), tracker, svcCfg)
}

// Name returns the configured model name.
func (p *OpenAIProvider) Name() string {
	return p.cfg.Model
}

// CountTokens estimates token count without an extra dependency.
func (p *OpenAIProvider) CountTokens(text string) int {
	words := len(strings.Fields(text))
	if words == 0 {
		return 0
	}
	return (words*4 + 2) / 3
}

// Complete calls OpenAI chat completions.
func (p *OpenAIProvider) Complete(ctx context.Context, req LLMProviderRequest) (*LLMProviderResponse, error) {
	if strings.TrimSpace(p.cfg.APIKey) == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is required")
	}
	maxTokens := req.MaxTokens
	if maxTokens <= 0 {
		maxTokens = p.cfg.MaxTokens
	}

	payload := openAIChatRequest{
		Model: p.cfg.Model,
		Messages: []openAIMessage{
			{Role: "user", Content: req.Prompt},
		},
		MaxTokens: maxTokens,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal openai request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(p.cfg.BaseURL, "/")+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create openai request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read openai response: %w", err)
	}
	if resp.StatusCode >= 400 {
		return nil, &LLMProviderError{
			StatusCode: resp.StatusCode,
			Message:    parseOpenAIErrorMessage(respBody),
			RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After")),
		}
	}

	var parsed openAIChatResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("parse openai response: %w", err)
	}
	if len(parsed.Choices) == 0 {
		return nil, fmt.Errorf("openai response contained no choices")
	}

	return &LLMProviderResponse{
		Text:         parsed.Choices[0].Message.Content,
		InputTokens:  parsed.Usage.PromptTokens,
		OutputTokens: parsed.Usage.CompletionTokens,
	}, nil
}

type openAIChatRequest struct {
	Model     string          `json:"model"`
	Messages  []openAIMessage `json:"messages"`
	MaxTokens int             `json:"max_tokens"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message openAIMessage `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type openAIErrorResponse struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

func parseOpenAIErrorMessage(body []byte) string {
	var parsed openAIErrorResponse
	if err := json.Unmarshal(body, &parsed); err == nil && parsed.Error.Message != "" {
		return parsed.Error.Message
	}
	return string(body)
}

func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}
	seconds, err := strconv.Atoi(value)
	if err == nil {
		return time.Duration(seconds) * time.Second
	}
	when, err := http.ParseTime(value)
	if err != nil {
		return 0
	}
	delay := time.Until(when)
	if delay < 0 {
		return 0
	}
	return delay
}
