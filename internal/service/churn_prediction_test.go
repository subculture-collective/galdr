package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/ml"
	"github.com/onnwee/pulse-score/internal/repository"
)

func TestChurnPredictionServiceScoresOrgCustomers(t *testing.T) {
	orgID := uuid.New()
	customerID := uuid.New()
	now := time.Date(2026, 5, 5, 12, 0, 0, 0, time.UTC)
	store := &recordingChurnPredictionStore{}
	svc := NewChurnPredictionService(ChurnPredictionDeps{
		Customers: &fakeChurnCustomerSource{customers: []*repository.Customer{{ID: customerID, OrgID: orgID, Name: "Acme"}}},
		Features: &fakeChurnFeatureSource{features: map[uuid.UUID]*repository.CustomerFeature{
			customerID: {
				OrgID:      orgID,
				CustomerID: customerID,
				Features: map[string]float64{
					ml.FeatureCurrentHealthScore:              0.2,
					ml.FeaturePaymentFailureFrequency90d:      1.0,
					ml.FeaturePaymentSuccessRate90d:           0.1,
					ml.FeatureDaysSinceLastActivity:           0.9,
					ml.FeatureUsageFrequencyChange30d:         0.8,
					ml.FeatureUnresolvedTicketRatio:           0.7,
					ml.FeatureScoreTrajectorySlope30d:         0.6,
					ml.FeatureSupportTicketTrend:              0.5,
					ml.FeatureMRRChangeRate:                   0.4,
					ml.FeatureEngagementEventsPerDay:          0.3,
					ml.FeatureContractTenure:                  0.2,
				},
			},
		}},
		Store: store,
		Now:   func() time.Time { return now },
	})

	if err := svc.ScoreOrg(context.Background(), orgID); err != nil {
		t.Fatalf("score org: %v", err)
	}

	if len(store.predictions) != 1 {
		t.Fatalf("expected 1 prediction, got %d", len(store.predictions))
	}
	prediction := store.predictions[0]
	if prediction.CustomerID != customerID || prediction.OrgID != orgID {
		t.Fatalf("stored prediction for wrong customer/org: %+v", prediction)
	}
	if prediction.Probability <= 0.5 || prediction.Probability > 1 {
		t.Fatalf("expected high churn probability, got %f", prediction.Probability)
	}
	if prediction.Confidence <= 0 || prediction.Confidence > 1 {
		t.Fatalf("expected bounded confidence, got %f", prediction.Confidence)
	}
	if len(prediction.RiskFactors) != 3 {
		t.Fatalf("expected top 3 risk factors, got %d", len(prediction.RiskFactors))
	}
	if prediction.RiskFactors[0].Feature == "" || prediction.RiskFactors[0].Contribution <= 0 {
		t.Fatalf("expected meaningful leading risk factor, got %+v", prediction.RiskFactors[0])
	}
	if prediction.ModelVersion == "" {
		t.Fatal("expected model version")
	}
	if !prediction.PredictedAt.Equal(now) {
		t.Fatalf("expected predicted_at %s, got %s", now, prediction.PredictedAt)
	}
}

type fakeChurnCustomerSource struct {
	customers []*repository.Customer
}

func (s *fakeChurnCustomerSource) ListByOrg(ctx context.Context, orgID uuid.UUID) ([]*repository.Customer, error) {
	return s.customers, nil
}

func (s *fakeChurnCustomerSource) GetByIDAndOrg(ctx context.Context, customerID, orgID uuid.UUID) (*repository.Customer, error) {
	for _, customer := range s.customers {
		if customer.ID == customerID && customer.OrgID == orgID {
			return customer, nil
		}
	}
	return nil, nil
}

type fakeChurnFeatureSource struct {
	features map[uuid.UUID]*repository.CustomerFeature
}

func (s *fakeChurnFeatureSource) GetByCustomerID(ctx context.Context, customerID, orgID uuid.UUID) (*repository.CustomerFeature, error) {
	return s.features[customerID], nil
}

type recordingChurnPredictionStore struct {
	predictions []*repository.ChurnPrediction
}

func (s *recordingChurnPredictionStore) Upsert(ctx context.Context, prediction *repository.ChurnPrediction) error {
	s.predictions = append(s.predictions, prediction)
	return nil
}

func (s *recordingChurnPredictionStore) GetByCustomerID(ctx context.Context, customerID, orgID uuid.UUID) (*repository.ChurnPrediction, error) {
	for _, prediction := range s.predictions {
		if prediction.CustomerID == customerID && prediction.OrgID == orgID {
			return prediction, nil
		}
	}
	return nil, nil
}
