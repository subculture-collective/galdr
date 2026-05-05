package prompts

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"sort"
	"strings"
	"text/template"
	"time"
)

//go:embed *.tmpl
var templateFS embed.FS

type TemplateName string

const (
	CustomerAnalysisTemplate      TemplateName = "customer_analysis"
	RiskAssessmentTemplate        TemplateName = "risk_assessment"
	ActionRecommendationsTemplate TemplateName = "action_recommendations"
	SummaryTemplate               TemplateName = "summary"
)

const SystemPrompt = "You are PulseScore's customer health analyst. Use only provided customer health signals. Do not invent facts. Return only valid JSON that matches the requested schema."

type TemplateSpec struct {
	Name            TemplateName
	Filename        string
	MaxPromptTokens int
	MaxOutputTokens int
}

type RenderedPrompt struct {
	TemplateName    TemplateName
	Prompt          string
	EstimatedTokens int
	Spec            TemplateSpec
}

type CustomerAnalysisData struct {
	Customer    CustomerSnapshot
	HealthScore HealthScoreSnapshot
	Events      []CustomerEventSummary
}

type CustomerSnapshot struct {
	Name        string
	Email       string
	CompanyName string
	Source      string
	MRRCents    int
	Currency    string
	Tenure      time.Duration
}

type HealthScoreSnapshot struct {
	OverallScore   int
	RiskLevel      string
	ScoreChange7d  int
	ScoreChange30d int
	Factors        []ScoreFactor
}

type ScoreFactor struct {
	Name        string
	Score       float64
	Explanation string
}

type CustomerEventSummary struct {
	Type    string
	Source  string
	AgeDays int
	Summary string
}

type StructuredInsight struct {
	InsightType     string           `json:"insight_type"`
	Summary         string           `json:"summary"`
	RiskLevel       string           `json:"risk_level"`
	Confidence      string           `json:"confidence"`
	Signals         []InsightSignal  `json:"signals"`
	Recommendations []Recommendation `json:"recommendations"`
}

type InsightSignal struct {
	Name     string `json:"name"`
	Evidence string `json:"evidence"`
	Impact   string `json:"impact"`
}

type Recommendation struct {
	Action    string `json:"action"`
	Owner     string `json:"owner"`
	Priority  string `json:"priority"`
	Rationale string `json:"rationale"`
}

var specs = map[TemplateName]TemplateSpec{
	CustomerAnalysisTemplate: {
		Name:            CustomerAnalysisTemplate,
		Filename:        "customer_analysis.tmpl",
		MaxPromptTokens: 900,
		MaxOutputTokens: 700,
	},
	RiskAssessmentTemplate: {
		Name:            RiskAssessmentTemplate,
		Filename:        "risk_assessment.tmpl",
		MaxPromptTokens: 700,
		MaxOutputTokens: 500,
	},
	ActionRecommendationsTemplate: {
		Name:            ActionRecommendationsTemplate,
		Filename:        "action_recommendations.tmpl",
		MaxPromptTokens: 700,
		MaxOutputTokens: 550,
	},
	SummaryTemplate: {
		Name:            SummaryTemplate,
		Filename:        "summary.tmpl",
		MaxPromptTokens: 500,
		MaxOutputTokens: 250,
	},
}

func Specs() map[TemplateName]TemplateSpec {
	out := make(map[TemplateName]TemplateSpec, len(specs))
	for name, spec := range specs {
		out[name] = spec
	}
	return out
}

func TemplateNames() []TemplateName {
	names := make([]TemplateName, 0, len(specs))
	for name := range specs {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool { return names[i] < names[j] })
	return names
}

func Templates() (map[string]string, error) {
	out := make(map[string]string, len(specs))
	for _, name := range TemplateNames() {
		spec := specs[name]
		text, err := promptTemplateText(spec)
		if err != nil {
			return nil, err
		}
		out[string(name)] = text
	}
	return out, nil
}

func Render(name TemplateName, data CustomerAnalysisData) (*RenderedPrompt, error) {
	if err := validateData(data); err != nil {
		return nil, err
	}
	spec, ok := specs[name]
	if !ok {
		return nil, fmt.Errorf("unknown prompt template %q", name)
	}
	text, err := promptTemplateText(spec)
	if err != nil {
		return nil, err
	}
	tmpl, err := template.New(spec.Filename).Option("missingkey=error").Funcs(FuncMap()).Parse(text)
	if err != nil {
		return nil, fmt.Errorf("parse prompt template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("render prompt template: %w", err)
	}
	prompt := buf.String()
	return &RenderedPrompt{
		TemplateName:    name,
		Prompt:          prompt,
		EstimatedTokens: EstimateTokens(prompt),
		Spec:            spec,
	}, nil
}

func ParseStructuredOutput(raw []byte) (*StructuredInsight, error) {
	var insight StructuredInsight
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&insight); err != nil {
		return nil, fmt.Errorf("parse structured insight: %w", err)
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return nil, errors.New("structured insight must contain a single JSON object")
	}
	if err := insight.Validate(); err != nil {
		return nil, err
	}
	return &insight, nil
}

func (s StructuredInsight) Validate() error {
	if strings.TrimSpace(s.InsightType) == "" {
		return errors.New("insight_type is required")
	}
	if strings.TrimSpace(s.Summary) == "" {
		return errors.New("summary is required")
	}
	if !oneOf(s.RiskLevel, "green", "yellow", "red") {
		return fmt.Errorf("risk_level must be green, yellow, or red")
	}
	if !oneOf(s.Confidence, "high", "medium", "low") {
		return fmt.Errorf("confidence must be high, medium, or low")
	}
	if len(s.Signals) == 0 {
		return errors.New("at least one signal is required")
	}
	for i, signal := range s.Signals {
		if err := validateSignal(i, signal); err != nil {
			return err
		}
	}
	if len(s.Recommendations) == 0 {
		return errors.New("at least one recommendation is required")
	}
	for i, rec := range s.Recommendations {
		if err := validateRecommendation(i, rec); err != nil {
			return err
		}
	}
	return nil
}

func validateSignal(index int, signal InsightSignal) error {
	if strings.TrimSpace(signal.Name) == "" || strings.TrimSpace(signal.Evidence) == "" {
		return fmt.Errorf("signal %d requires name and evidence", index)
	}
	if !oneOf(signal.Impact, "positive", "negative", "neutral") {
		return fmt.Errorf("signal %d impact must be positive, negative, or neutral", index)
	}
	return nil
}

func validateRecommendation(index int, rec Recommendation) error {
	if strings.TrimSpace(rec.Action) == "" || strings.TrimSpace(rec.Owner) == "" || strings.TrimSpace(rec.Rationale) == "" {
		return fmt.Errorf("recommendation %d requires action, owner, and rationale", index)
	}
	if !oneOf(rec.Priority, "high", "medium", "low") {
		return fmt.Errorf("recommendation %d priority must be high, medium, or low", index)
	}
	return nil
}

func EstimateTokens(text string) int {
	words := len(strings.Fields(text))
	punctuation := strings.Count(text, "{") + strings.Count(text, "}") + strings.Count(text, "[") + strings.Count(text, "]")
	return int(math.Ceil(float64(words+punctuation) * 1.3))
}

func loadTemplate(spec TemplateSpec) (string, error) {
	content, err := templateFS.ReadFile(spec.Filename)
	if err != nil {
		return "", fmt.Errorf("load prompt template %s: %w", spec.Filename, err)
	}
	return string(content), nil
}

func promptTemplateText(spec TemplateSpec) (string, error) {
	body, err := loadTemplate(spec)
	if err != nil {
		return "", err
	}
	return SystemPrompt + "\n\n" + body, nil
}

func validateData(data CustomerAnalysisData) error {
	if strings.TrimSpace(data.Customer.Name) == "" {
		return errors.New("customer name is required")
	}
	if strings.TrimSpace(data.HealthScore.RiskLevel) == "" {
		return errors.New("risk level is required")
	}
	if len(data.HealthScore.Factors) == 0 {
		return errors.New("at least one score factor is required")
	}
	return nil
}

// FuncMap returns template helpers used by bundled prompt templates.
func FuncMap() template.FuncMap {
	return template.FuncMap{
		"formatMoney":     formatMoney,
		"formatTenure":    formatTenure,
		"formatSignedInt": formatSignedInt,
		"formatFloat":     formatFloat,
	}
}

func formatMoney(cents int, currency string) string {
	if currency == "" {
		currency = "usd"
	}
	return fmt.Sprintf("%s %s", strings.ToUpper(currency), formatCents(cents))
}

func formatCents(cents int) string {
	dollars := cents / 100
	remainder := cents % 100
	return fmt.Sprintf("%s.%02d", commaInt(dollars), remainder)
}

func commaInt(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var parts []string
	for len(s) > 3 {
		parts = append([]string{s[len(s)-3:]}, parts...)
		s = s[:len(s)-3]
	}
	parts = append([]string{s}, parts...)
	return strings.Join(parts, ",")
}

func formatTenure(tenure time.Duration) string {
	if tenure <= 0 {
		return "unknown"
	}
	months := int(math.Round(tenure.Hours() / 24 / 30))
	if months < 1 {
		return "less than 1 month"
	}
	if months == 1 {
		return "1 month"
	}
	return fmt.Sprintf("%d months", months)
}

func formatSignedInt(n int) string {
	if n > 0 {
		return fmt.Sprintf("+%d", n)
	}
	return fmt.Sprintf("%d", n)
}

func formatFloat(n float64) string {
	return fmt.Sprintf("%.2f", n)
}

func oneOf(value string, allowed ...string) bool {
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}
