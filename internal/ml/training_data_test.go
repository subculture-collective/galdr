package ml

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

func TestTrainingDataPreparerLabelsSplitsAndWeightsRows(t *testing.T) {
	now := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	orgID := uuid.New()
	churnedCustomerID := uuid.New()
	downgradedCustomerID := uuid.New()
	healthyCustomerID := uuid.New()
	lateChurnCustomerID := uuid.New()

	features := []*repository.CustomerFeature{
		featureSnapshot(orgID, churnedCustomerID, now.Add(-40*24*time.Hour), 0.80),
		featureSnapshot(orgID, downgradedCustomerID, now.Add(-35*24*time.Hour), 0.70),
		featureSnapshot(orgID, healthyCustomerID, now.Add(-20*24*time.Hour), 0.60),
		featureSnapshot(orgID, lateChurnCustomerID, now.Add(-10*24*time.Hour), 0.50),
	}
	canceledAt := features[0].CalculatedAt.Add(10 * 24 * time.Hour)
	lateCanceledAt := features[3].CalculatedAt.Add(45 * 24 * time.Hour)

	preparer := NewTrainingDataPreparer(TrainingDataPreparerDeps{
		Features: &trainingFeatureSource{features: features},
		Subscriptions: &trainingSubscriptionSource{subscriptions: map[uuid.UUID][]*repository.StripeSubscription{
			churnedCustomerID: {
				&repository.StripeSubscription{CustomerID: churnedCustomerID, Status: "canceled", CanceledAt: &canceledAt},
			},
			lateChurnCustomerID: {
				&repository.StripeSubscription{CustomerID: lateChurnCustomerID, Status: "canceled", CanceledAt: &lateCanceledAt},
			},
		}},
		Events: &trainingEventSource{events: map[uuid.UUID][]*repository.CustomerEvent{
			downgradedCustomerID: {event(eventMRRChanged, features[1].CalculatedAt.Add(12*24*time.Hour), map[string]any{"old_mrr_cents": 100_000, "new_mrr_cents": 70_000})},
		}},
	})

	dataset, err := preparer.Prepare(context.Background(), TrainingDataRequest{
		OrgID:       orgID,
		From:        now.Add(-60 * 24 * time.Hour),
		To:          now,
		LabelWindow: 30 * 24 * time.Hour,
	})
	if err != nil {
		t.Fatalf("prepare training data: %v", err)
	}

	if len(dataset.Train) != 3 || len(dataset.Test) != 1 {
		t.Fatalf("split train/test lengths = %d/%d, want 3/1", len(dataset.Train), len(dataset.Test))
	}
	if dataset.Train[0].Label != ChurnLabelChurned || dataset.Train[1].Label != ChurnLabelChurned || dataset.Train[2].Label != ChurnLabelRetained {
		t.Fatalf("unexpected train labels: %+v", dataset.Train)
	}
	if dataset.Test[0].Label != ChurnLabelRetained {
		t.Fatalf("unexpected test label: %+v", dataset.Test[0])
	}
	if dataset.ClassWeights[ChurnLabelChurned] != 1 || dataset.ClassWeights[ChurnLabelRetained] != 1 {
		t.Fatalf("balanced class weights = %+v, want 1 each", dataset.ClassWeights)
	}
	if dataset.Quality.TotalRows != 4 || dataset.Quality.ChurnedRows != 2 || dataset.Quality.RetainedRows != 2 {
		t.Fatalf("unexpected quality summary: %+v", dataset.Quality)
	}
	if len(dataset.FeatureNames) != len(FeatureNames()) || len(dataset.Train[0].Features) != len(FeatureNames()) {
		t.Fatalf("training rows do not use stable feature order")
	}
}

func TestTrainingDatasetExportsAndWeightsImbalancedClasses(t *testing.T) {
	orgID := uuid.New()
	base := time.Date(2026, 5, 4, 12, 0, 0, 0, time.UTC)
	dataset := TrainingDataset{
		FeatureNames: FeatureNames(),
		Train: []TrainingDataRow{
			{OrgID: orgID, CustomerID: uuid.New(), SnapshotAt: base, Features: valuesFromMap(map[string]float64{FeatureCurrentHealthScore: 0.1}), Label: ChurnLabelChurned, ClassWeight: 2},
			{OrgID: orgID, CustomerID: uuid.New(), SnapshotAt: base.Add(time.Hour), Features: valuesFromMap(map[string]float64{FeatureCurrentHealthScore: 0.8}), Label: ChurnLabelRetained, ClassWeight: 0.67},
		},
		Test: []TrainingDataRow{
			{OrgID: orgID, CustomerID: uuid.New(), SnapshotAt: base.Add(2 * time.Hour), Features: valuesFromMap(map[string]float64{FeatureCurrentHealthScore: 0.9}), Label: ChurnLabelRetained, ClassWeight: 0.67},
		},
		ClassWeights: classWeights(1, 2),
	}

	if dataset.ClassWeights[ChurnLabelChurned] != 1.5 || dataset.ClassWeights[ChurnLabelRetained] != 0.75 {
		t.Fatalf("imbalanced weights = %+v, want churned 1.5 retained 0.75", dataset.ClassWeights)
	}

	csvBytes, err := dataset.ToCSV()
	if err != nil {
		t.Fatalf("export csv: %v", err)
	}
	csvText := string(csvBytes)
	if !strings.Contains(csvText, "org_id,customer_id,snapshot_at") || !strings.Contains(csvText, ",label,class_weight,split") {
		t.Fatalf("csv export missing expected header: %q", csvText)
	}
	if !strings.Contains(csvText, ",train\n") || !strings.Contains(csvText, ",test\n") {
		t.Fatalf("csv export missing split values: %q", csvText)
	}

	jsonBytes, err := dataset.ToJSON()
	if err != nil {
		t.Fatalf("export json: %v", err)
	}
	var decoded TrainingDataset
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Fatalf("json export does not decode: %v", err)
	}
	if len(decoded.Train) != 2 || len(decoded.Test) != 1 || len(decoded.FeatureNames) != len(FeatureNames()) {
		t.Fatalf("json export decoded unexpected dataset: %+v", decoded)
	}
}

func featureSnapshot(orgID, customerID uuid.UUID, calculatedAt time.Time, score float64) *repository.CustomerFeature {
	return &repository.CustomerFeature{
		OrgID:        orgID,
		CustomerID:   customerID,
		CalculatedAt: calculatedAt,
		Features: map[string]float64{
			FeatureCurrentHealthScore:         score,
			FeaturePaymentFailureFrequency90d: 1 - score,
		},
	}
}

type trainingFeatureSource struct {
	features []*repository.CustomerFeature
}

func (s *trainingFeatureSource) ListByOrgBetween(_ context.Context, _ uuid.UUID, _, _ time.Time) ([]*repository.CustomerFeature, error) {
	return s.features, nil
}

type trainingSubscriptionSource struct {
	subscriptions map[uuid.UUID][]*repository.StripeSubscription
}

func (s *trainingSubscriptionSource) ListByCustomer(_ context.Context, customerID uuid.UUID) ([]*repository.StripeSubscription, error) {
	return s.subscriptions[customerID], nil
}

type trainingEventSource struct {
	events map[uuid.UUID][]*repository.CustomerEvent
}

func (s *trainingEventSource) ListByCustomerBetween(_ context.Context, customerID uuid.UUID, _, _ time.Time) ([]*repository.CustomerEvent, error) {
	return s.events[customerID], nil
}
