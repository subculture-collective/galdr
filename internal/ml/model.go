package ml

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

const (
	defaultTrainIterations = 1000
	defaultLearningRate    = 0.2
	defaultDecisionCutoff  = 0.5
	maxSigmoidInput        = 35
)

// TrainOptions controls logistic-regression model training.
type TrainOptions struct {
	Version      string
	Iterations   int
	LearningRate float64
	Cutoff       float64
}

// ChurnModel predicts customer churn probability from prepared features.
type ChurnModel struct {
	Version      string          `json:"version"`
	FeatureNames []string        `json:"feature_names"`
	Weights      []float64       `json:"weights"`
	Bias         float64         `json:"bias"`
	Cutoff       float64         `json:"cutoff"`
	Metrics      ModelMetrics    `json:"metrics"`
	TrainedAt    time.Time       `json:"trained_at"`
}

// ModelMetrics summarizes held-out model quality.
type ModelMetrics struct {
	AUCROC    float64 `json:"auc_roc"`
	Precision float64 `json:"precision"`
	Recall    float64 `json:"recall"`
	F1        float64 `json:"f1"`
}

// ChurnPrediction is a model-scored churn probability for one customer.
type ChurnPrediction struct {
	OrgID            uuid.UUID `json:"org_id"`
	CustomerID       uuid.UUID `json:"customer_id"`
	ModelVersion     string    `json:"model_version"`
	ChurnProbability float64   `json:"churn_probability"`
	PredictedAt       time.Time `json:"predicted_at"`
}

// TrainChurnModel trains a weighted logistic-regression churn model.
func TrainChurnModel(dataset TrainingDataset, opts TrainOptions) (*ChurnModel, error) {
	if len(dataset.Train) == 0 {
		return nil, fmt.Errorf("training data is required")
	}
	featureNames := append([]string(nil), dataset.FeatureNames...)
	if len(featureNames) == 0 {
		featureNames = FeatureNames()
	}
	if err := validateTrainingRows(dataset.Train, len(featureNames)); err != nil {
		return nil, err
	}

	opts = normalizeTrainOptions(opts)
	model := &ChurnModel{
		Version:      opts.Version,
		FeatureNames: featureNames,
		Weights:      make([]float64, len(featureNames)),
		Cutoff:       opts.Cutoff,
		TrainedAt:    time.Now().UTC(),
	}

	for i := 0; i < opts.Iterations; i++ {
		model.applyGradientStep(dataset.Train, opts.LearningRate)
	}

	metricsRows := dataset.Test
	if len(metricsRows) == 0 {
		metricsRows = dataset.Train
	}
	metrics, err := model.Evaluate(metricsRows)
	if err != nil {
		return nil, err
	}
	model.Metrics = metrics
	return model, nil
}

// PredictProbability returns churn_probability in [0,1].
func (m *ChurnModel) PredictProbability(features []float64) (float64, error) {
	if m == nil {
		return 0, fmt.Errorf("churn model is nil")
	}
	if len(features) != len(m.Weights) {
		return 0, fmt.Errorf("feature length %d does not match model length %d", len(features), len(m.Weights))
	}
	return sigmoid(m.score(features)), nil
}

// PredictCustomer scores one customer feature vector with model metadata.
func (m *ChurnModel) PredictCustomer(vector FeatureVector) (*ChurnPrediction, error) {
	probability, err := m.PredictProbability(vector.ValuesInOrder())
	if err != nil {
		return nil, err
	}
	return &ChurnPrediction{
		OrgID:            vector.OrgID,
		CustomerID:       vector.CustomerID,
		ModelVersion:     m.Version,
		ChurnProbability: probability,
		PredictedAt:       time.Now().UTC(),
	}, nil
}

// StoredVersion converts the trained model into a persistable model artifact.
func (m *ChurnModel) StoredVersion(orgID uuid.UUID) (*repository.ChurnModelVersion, error) {
	if m == nil {
		return nil, fmt.Errorf("churn model is nil")
	}
	if orgID == uuid.Nil {
		return nil, fmt.Errorf("org id is required")
	}
	return &repository.ChurnModelVersion{
		OrgID:        orgID,
		Version:      m.Version,
		FeatureNames: append([]string(nil), m.FeatureNames...),
		Weights:      append([]float64(nil), m.Weights...),
		Bias:         m.Bias,
		Cutoff:       m.Cutoff,
		Metrics: map[string]float64{
			"auc_roc":   m.Metrics.AUCROC,
			"precision": m.Metrics.Precision,
			"recall":    m.Metrics.Recall,
			"f1":        m.Metrics.F1,
		},
		TrainedAt: m.TrainedAt,
	}, nil
}

// Evaluate calculates AUC-ROC, precision, recall, and F1 for labeled rows.
func (m *ChurnModel) Evaluate(rows []TrainingDataRow) (ModelMetrics, error) {
	if len(rows) == 0 {
		return ModelMetrics{}, fmt.Errorf("evaluation rows are required")
	}
	scores := make([]labeledScore, 0, len(rows))
	var truePositive, falsePositive, falseNegative float64
	for _, row := range rows {
		probability, err := m.PredictProbability(row.Features)
		if err != nil {
			return ModelMetrics{}, err
		}
		scores = append(scores, labeledScore{probability: probability, label: row.Label})
		predictedChurn := probability >= m.Cutoff
		actualChurn := row.Label == ChurnLabelChurned
		switch {
		case predictedChurn && actualChurn:
			truePositive++
		case predictedChurn && !actualChurn:
			falsePositive++
		case !predictedChurn && actualChurn:
			falseNegative++
		}
	}
	precision := safeDivide(truePositive, truePositive+falsePositive)
	recall := safeDivide(truePositive, truePositive+falseNegative)
	return ModelMetrics{
		AUCROC:    aucROC(scores),
		Precision: precision,
		Recall:    recall,
		F1:        safeDivide(2*precision*recall, precision+recall),
	}, nil
}

func (m *ChurnModel) applyGradientStep(rows []TrainingDataRow, learningRate float64) {
	weightGradients := make([]float64, len(m.Weights))
	var biasGradient, totalWeight float64
	for _, row := range rows {
		rowWeight := row.ClassWeight
		if rowWeight <= 0 {
			rowWeight = 1
		}
		prediction := sigmoid(m.score(row.Features))
		errorTerm := (prediction - float64(row.Label)) * rowWeight
		for i, value := range row.Features {
			weightGradients[i] += errorTerm * value
		}
		biasGradient += errorTerm
		totalWeight += rowWeight
	}
	if totalWeight == 0 {
		return
	}
	step := learningRate / totalWeight
	for i := range m.Weights {
		m.Weights[i] -= step * weightGradients[i]
	}
	m.Bias -= step * biasGradient
}

func (m *ChurnModel) score(features []float64) float64 {
	score := m.Bias
	for i, value := range features {
		score += m.Weights[i] * value
	}
	return score
}

func normalizeTrainOptions(opts TrainOptions) TrainOptions {
	if opts.Version == "" {
		opts.Version = "churn-logistic-" + time.Now().UTC().Format("20060102150405")
	}
	if opts.Iterations <= 0 {
		opts.Iterations = defaultTrainIterations
	}
	if opts.LearningRate <= 0 {
		opts.LearningRate = defaultLearningRate
	}
	if opts.Cutoff <= 0 || opts.Cutoff >= 1 {
		opts.Cutoff = defaultDecisionCutoff
	}
	return opts
}

func validateTrainingRows(rows []TrainingDataRow, featureCount int) error {
	seen := map[int]struct{}{}
	for _, row := range rows {
		if len(row.Features) != featureCount {
			return fmt.Errorf("training row feature length %d does not match expected %d", len(row.Features), featureCount)
		}
		if row.Label != ChurnLabelChurned && row.Label != ChurnLabelRetained {
			return fmt.Errorf("unsupported churn label %d", row.Label)
		}
		seen[row.Label] = struct{}{}
	}
	if len(seen) < 2 {
		return fmt.Errorf("training data must include churned and retained rows")
	}
	return nil
}

type labeledScore struct {
	probability float64
	label       int
}

func aucROC(scores []labeledScore) float64 {
	var positives, negatives float64
	for _, score := range scores {
		if score.label == ChurnLabelChurned {
			positives++
		} else {
			negatives++
		}
	}
	if positives == 0 || negatives == 0 {
		return 0
	}
	sort.SliceStable(scores, func(i, j int) bool {
		return scores[i].probability > scores[j].probability
	})
	var rankSum, pairRank float64
	for _, score := range scores {
		if score.label == ChurnLabelChurned {
			rankSum += negatives - pairRank
		} else {
			pairRank++
		}
	}
	return rankSum / (positives * negatives)
}

func sigmoid(value float64) float64 {
	if value > maxSigmoidInput {
		return 1
	}
	if value < -maxSigmoidInput {
		return 0
	}
	return 1 / (1 + math.Exp(-value))
}

func safeDivide(numerator, denominator float64) float64 {
	if denominator == 0 {
		return 0
	}
	return numerator / denominator
}
