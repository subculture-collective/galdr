package ml

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTrainChurnModelPredictsAndEvaluatesRisk(t *testing.T) {
	base := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	dataset := TrainingDataset{
		FeatureNames: FeatureNames(),
		Train: []TrainingDataRow{
			modelRow(base, 0.05, 0.95, ChurnLabelChurned),
			modelRow(base.Add(time.Hour), 0.15, 0.85, ChurnLabelChurned),
			modelRow(base.Add(2*time.Hour), 0.9, 0.05, ChurnLabelRetained),
			modelRow(base.Add(3*time.Hour), 0.8, 0.1, ChurnLabelRetained),
		},
		Test: []TrainingDataRow{
			modelRow(base.Add(4*time.Hour), 0.1, 0.9, ChurnLabelChurned),
			modelRow(base.Add(5*time.Hour), 0.85, 0.05, ChurnLabelRetained),
		},
		ClassWeights: map[int]float64{ChurnLabelChurned: 1, ChurnLabelRetained: 1},
	}

	model, err := TrainChurnModel(dataset, TrainOptions{Version: "test-model", Iterations: 800, LearningRate: 0.4})
	if err != nil {
		t.Fatalf("train churn model: %v", err)
	}

	if model.Version != "test-model" || len(model.Weights) != len(FeatureNames()) {
		t.Fatalf("unexpected model metadata: %+v", model)
	}

	risky, err := model.PredictProbability(valuesFromMap(map[string]float64{
		FeatureCurrentHealthScore:         0.05,
		FeaturePaymentFailureFrequency90d: 0.95,
	}))
	if err != nil {
		t.Fatalf("predict risky customer: %v", err)
	}
	healthy, err := model.PredictProbability(valuesFromMap(map[string]float64{
		FeatureCurrentHealthScore:         0.9,
		FeaturePaymentFailureFrequency90d: 0.05,
	}))
	if err != nil {
		t.Fatalf("predict healthy customer: %v", err)
	}
	if risky <= healthy || risky < 0.7 || healthy > 0.3 {
		t.Fatalf("risk probabilities not separated: risky=%f healthy=%f", risky, healthy)
	}

	metrics, err := model.Evaluate(dataset.Test)
	if err != nil {
		t.Fatalf("evaluate model: %v", err)
	}
	if metrics.AUCROC < 0.99 || metrics.Precision != 1 || metrics.Recall != 1 || metrics.F1 != 1 {
		t.Fatalf("unexpected evaluation metrics: %+v", metrics)
	}

	orgID := uuid.New()
	stored, err := model.StoredVersion(orgID)
	if err != nil {
		t.Fatalf("store model version: %v", err)
	}
	if stored.OrgID != orgID || stored.Version != "test-model" || stored.Metrics["auc_roc"] < 0.99 {
		t.Fatalf("stored model does not preserve version and metrics: %+v", stored)
	}

	prediction, err := model.PredictCustomer(FeatureVector{
		OrgID:      orgID,
		CustomerID: uuid.New(),
		Values: map[string]float64{
			FeatureCurrentHealthScore:         0.05,
			FeaturePaymentFailureFrequency90d: 0.95,
		},
	})
	if err != nil {
		t.Fatalf("predict customer: %v", err)
	}
	if prediction.ModelVersion != "test-model" || prediction.ChurnProbability < 0.7 || prediction.PredictedAt.IsZero() {
		t.Fatalf("unexpected customer prediction: %+v", prediction)
	}
}

func modelRow(at time.Time, healthScore, failureFrequency float64, label int) TrainingDataRow {
	return TrainingDataRow{
		OrgID:      uuid.New(),
		CustomerID: uuid.New(),
		SnapshotAt: at,
		Features: valuesFromMap(map[string]float64{
			FeatureCurrentHealthScore:         healthScore,
			FeaturePaymentFailureFrequency90d: failureFrequency,
		}),
		Label:       label,
		ClassWeight: 1,
	}
}
