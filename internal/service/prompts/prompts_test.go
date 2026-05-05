package prompts

import (
	"strings"
	"testing"
	"time"
)

func TestRenderCustomerAnalysisTemplateUsesHealthContext(t *testing.T) {
	data := sampleCustomerAnalysisData()

	rendered, err := Render(CustomerAnalysisTemplate, data)
	if err != nil {
		t.Fatalf("expected render success, got %v", err)
	}

	assertContains(t, rendered.Prompt, SystemPrompt)
	assertContains(t, rendered.Prompt, "Acme Corp")
	assertContains(t, rendered.Prompt, "Risk level: red")
	assertContains(t, rendered.Prompt, "Score change, 30d: -18")
	assertContains(t, rendered.Prompt, "MRR: USD 12,500.00")
	assertContains(t, rendered.Prompt, "Tenure: 14 months")
	assertContains(t, rendered.Prompt, "payment_failed")
	assertContains(t, rendered.Prompt, "payment_recency: 0.22")
	assertContains(t, rendered.Prompt, `"summary"`)
	if rendered.EstimatedTokens > rendered.Spec.MaxPromptTokens {
		t.Fatalf("prompt exceeds budget: got %d want <= %d", rendered.EstimatedTokens, rendered.Spec.MaxPromptTokens)
	}
}

func TestRenderReturnsMissingVariableError(t *testing.T) {
	data := sampleCustomerAnalysisData()
	data.Customer.Name = ""

	_, err := Render(CustomerAnalysisTemplate, data)
	if err == nil {
		t.Fatal("expected missing customer name error")
	}
}

func TestAllTemplatesRenderParseableJSONInstructionsWithinBudget(t *testing.T) {
	for _, name := range TemplateNames() {
		t.Run(string(name), func(t *testing.T) {
			rendered, err := Render(name, sampleCustomerAnalysisData())
			if err != nil {
				t.Fatalf("expected render success, got %v", err)
			}
			assertContains(t, rendered.Prompt, "Return only valid JSON")
			assertContains(t, rendered.Prompt, `"insight_type"`)
			if rendered.EstimatedTokens > rendered.Spec.MaxPromptTokens {
				t.Fatalf("prompt exceeds budget: got %d want <= %d", rendered.EstimatedTokens, rendered.Spec.MaxPromptTokens)
			}
		})
	}
}

func TestTemplatesReturnsSystemPromptPrefixedTemplateText(t *testing.T) {
	templates, err := Templates()
	if err != nil {
		t.Fatalf("expected templates load success, got %v", err)
	}

	if len(templates) != len(TemplateNames()) {
		t.Fatalf("expected %d templates, got %d", len(TemplateNames()), len(templates))
	}
	assertContains(t, templates[string(CustomerAnalysisTemplate)], SystemPrompt)
	assertContains(t, templates[string(CustomerAnalysisTemplate)], "Use the customer health context below")
}

func TestParseStructuredOutputValidatesInsight(t *testing.T) {
	insight, err := ParseStructuredOutput([]byte(validStructuredOutput()))
	if err != nil {
		t.Fatalf("expected parse success, got %v", err)
	}
	if insight.InsightType != "risk_assessment" || len(insight.Recommendations) != 1 {
		t.Fatalf("unexpected parsed insight: %+v", insight)
	}

	_, err = ParseStructuredOutput([]byte(`{"summary":"missing required fields"}`))
	if err == nil {
		t.Fatal("expected invalid output error")
	}
}

func TestParseStructuredOutputRejectsTrailingText(t *testing.T) {
	_, err := ParseStructuredOutput([]byte(validStructuredOutput() + "\nHere is why."))
	if err == nil {
		t.Fatal("expected trailing text to be rejected")
	}
}

func validStructuredOutput() string {
	return `{
		"insight_type":"risk_assessment",
		"summary":"Acme is at high churn risk because payments failed and engagement dropped.",
		"risk_level":"red",
		"confidence":"high",
		"signals":[{"name":"Failed payments","evidence":"Three recent failed payments","impact":"negative"}],
		"recommendations":[{"action":"Schedule billing recovery call","owner":"customer_success","priority":"high","rationale":"Recover payment method before renewal"}]
	}`
}

func sampleCustomerAnalysisData() CustomerAnalysisData {
	return CustomerAnalysisData{
		Customer: CustomerSnapshot{
			Name:        "Acme Corp",
			Email:       "owner@acme.example",
			CompanyName: "Acme Corp",
			Source:      "stripe",
			MRRCents:    1250000,
			Currency:    "usd",
			Tenure:      14 * 30 * 24 * time.Hour,
		},
		HealthScore: HealthScoreSnapshot{
			OverallScore: 32,
			RiskLevel:    "red",
			ScoreChange7d: -9,
			ScoreChange30d: -18,
			Factors: []ScoreFactor{
				{Name: "payment_recency", Score: 0.22, Explanation: "Last successful payment was 52 days ago"},
				{Name: "failed_payments", Score: 0.10, Explanation: "Three unresolved failures"},
			},
		},
		Events: []CustomerEventSummary{
			{Type: "payment_failed", Source: "stripe", AgeDays: 2, Summary: "Invoice payment failed"},
			{Type: "risk_level.changed", Source: "scoring", AgeDays: 1, Summary: "Risk moved yellow to red"},
		},
	}
}

func assertContains(t *testing.T, got, want string) {
	t.Helper()
	if !strings.Contains(got, want) {
		t.Fatalf("expected %q to contain %q", got, want)
	}
}
