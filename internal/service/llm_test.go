package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

type stubLLMProvider struct {
	lastPrompt string
	err        error
	name       string
}

func (p *stubLLMProvider) Complete(ctx context.Context, req LLMProviderRequest) (*LLMProviderResponse, error) {
	p.lastPrompt = req.Prompt
	if p.err != nil {
		return nil, p.err
	}
	return &LLMProviderResponse{
		Text:         "Customer is healthy.",
		Model:        req.Model,
		InputTokens:  12,
		OutputTokens: 4,
	}, nil
}

func (p *stubLLMProvider) CountTokens(text string) int { return len(text) / 4 }
func (p *stubLLMProvider) Name() string {
	if p.name != "" {
		return p.name
	}
	return "stub"
}

func TestLLMServiceCompleteRendersTemplateAndTracksUsage(t *testing.T) {
	provider := &stubLLMProvider{}
	svc := NewLLMService(provider, LLMServiceConfig{
		Model:             "gpt-4o-mini",
		RequestsPerMinute: 60,
		MaxTokensPerDay:   1000,
		Templates: map[string]string{
			"customer_health": "Summarize {{customer}} with score {{score}}.",
		},
	})

	orgID := uuid.New()
	resp, err := svc.Complete(context.Background(), LLMCompletionRequest{
		OrgID:        orgID,
		TemplateName: "customer_health",
		TemplateData: map[string]string{"customer": "Acme", "score": "84"},
		MaxTokens:    200,
	})
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}

	if provider.lastPrompt != "Summarize Acme with score 84." {
		t.Fatalf("expected rendered prompt, got %q", provider.lastPrompt)
	}
	if resp.Text != "Customer is healthy." {
		t.Fatalf("expected completion text, got %q", resp.Text)
	}
	if resp.Usage.OrgID != orgID {
		t.Fatalf("expected org usage to be tracked")
	}
	if resp.Usage.Provider != "stub" || resp.Usage.Model != "gpt-4o-mini" {
		t.Fatalf("expected provider/model usage, got %+v", resp.Usage)
	}
	if resp.Usage.InputTokens != 12 || resp.Usage.OutputTokens != 4 || resp.Usage.TotalTokens != 16 {
		t.Fatalf("expected token usage to be tracked, got %+v", resp.Usage)
	}
	if resp.Usage.CostUSD <= 0 {
		t.Fatalf("expected cost to be calculated, got %+v", resp.Usage)
	}
}

func TestLLMServiceCompleteEnforcesOrgLimits(t *testing.T) {
	provider := &stubLLMProvider{}
	svc := NewLLMService(provider, LLMServiceConfig{
		Model:             "gpt-4o-mini",
		RequestsPerMinute: 1,
		MaxTokensPerDay:   210,
	})

	orgID := uuid.New()
	_, err := svc.Complete(context.Background(), LLMCompletionRequest{
		OrgID:     orgID,
		Prompt:    "Summarize Acme.",
		MaxTokens: 200,
	})
	if err != nil {
		t.Fatalf("first completion returned error: %v", err)
	}

	_, err = svc.Complete(context.Background(), LLMCompletionRequest{
		OrgID:     orgID,
		Prompt:    "Summarize Beta.",
		MaxTokens: 1,
	})
	if err != ErrLLMRateLimited {
		t.Fatalf("expected rate limit error, got %v", err)
	}

	otherOrgID := uuid.New()
	_, err = svc.Complete(context.Background(), LLMCompletionRequest{
		OrgID:     otherOrgID,
		Prompt:    "Summarize Gamma.",
		MaxTokens: 300,
	})
	if err != ErrLLMTokenBudgetExceeded {
		t.Fatalf("expected token budget error, got %v", err)
	}
}

func TestLLMServiceCompleteFallsBackWhenPrimaryProviderFails(t *testing.T) {
	primary := &stubLLMProvider{name: "primary", err: errors.New("primary unavailable")}
	fallback := &stubLLMProvider{name: "fallback"}
	svc := NewLLMService(primary, LLMServiceConfig{
		Model:             "gpt-4o-mini",
		RequestsPerMinute: 60,
		MaxTokensPerDay:   1000,
		FallbackProvider:  fallback,
	})

	resp, err := svc.Complete(context.Background(), LLMCompletionRequest{
		OrgID:     uuid.New(),
		Prompt:    "Summarize Acme.",
		MaxTokens: 100,
	})
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}

	if primary.lastPrompt != "Summarize Acme." {
		t.Fatalf("expected primary provider to be attempted")
	}
	if fallback.lastPrompt != "Summarize Acme." {
		t.Fatalf("expected fallback provider to receive prompt")
	}
	if resp.Usage.Provider != "fallback" {
		t.Fatalf("expected fallback usage to be tracked, got %+v", resp.Usage)
	}
}

func TestOpenAIProviderCompleteRetriesRateLimitAndParsesUsage(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer sk_test" {
			t.Fatalf("missing auth header")
		}

		if requests == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"slow down"}`))
			return
		}

		var payload openAIChatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload.Model != "gpt-4o-mini" || payload.Messages[0].Content != "Summarize Acme." {
			t.Fatalf("unexpected payload %+v", payload)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"model":"gpt-4o-mini",
			"choices":[{"message":{"role":"assistant","content":"Acme is healthy."}}],
			"usage":{"prompt_tokens":10,"completion_tokens":5}
		}`))
	}))
	defer server.Close()

	provider := NewOpenAIProvider(OpenAIConfig{
		APIKey:       "sk_test",
		BaseURL:      server.URL,
		MaxRetries:   1,
		RetryBackoff: 0,
		HTTPClient:   server.Client(),
	})

	resp, err := provider.Complete(context.Background(), LLMProviderRequest{
		Prompt:    "Summarize Acme.",
		Model:     "gpt-4o-mini",
		MaxTokens: 100,
	})
	if err != nil {
		t.Fatalf("Complete returned error: %v", err)
	}
	if requests != 2 {
		t.Fatalf("expected retry, got %d requests", requests)
	}
	if resp.Text != "Acme is healthy." || resp.InputTokens != 10 || resp.OutputTokens != 5 {
		t.Fatalf("unexpected response %+v", resp)
	}
}
