package ml

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/onnwee/pulse-score/internal/repository"
)

const (
	ChurnLabelRetained = 0
	ChurnLabelChurned  = 1
)

const defaultLabelWindow = 30 * 24 * time.Hour

// TrainingFeatureSource loads historical feature snapshots for dataset prep.
type TrainingFeatureSource interface {
	ListByOrgBetween(ctx context.Context, orgID uuid.UUID, from, to time.Time) ([]*repository.CustomerFeature, error)
}

// TrainingSubscriptionSource loads subscription history used for churn labels.
type TrainingSubscriptionSource interface {
	ListByCustomer(ctx context.Context, customerID uuid.UUID) ([]*repository.StripeSubscription, error)
}

// TrainingEventSource loads event history used for downgrade labels.
type TrainingEventSource interface {
	ListByCustomerBetween(ctx context.Context, customerID uuid.UUID, from, to time.Time) ([]*repository.CustomerEvent, error)
}

type TrainingDataPreparerDeps struct {
	Features      TrainingFeatureSource
	Subscriptions TrainingSubscriptionSource
	Events        TrainingEventSource
}

// TrainingDataPreparer turns feature snapshots into labeled train/test rows.
type TrainingDataPreparer struct {
	features      TrainingFeatureSource
	subscriptions TrainingSubscriptionSource
	events        TrainingEventSource
}

type TrainingDataRequest struct {
	OrgID       uuid.UUID
	From        time.Time
	To          time.Time
	LabelWindow time.Duration
}

type TrainingDataset struct {
	FeatureNames  []string           `json:"feature_names"`
	Train         []TrainingDataRow  `json:"train"`
	Test          []TrainingDataRow  `json:"test"`
	ClassWeights  map[int]float64    `json:"class_weights"`
	Quality       DataQualitySummary `json:"quality"`
}

type TrainingDataRow struct {
	OrgID        uuid.UUID `json:"org_id"`
	CustomerID   uuid.UUID `json:"customer_id"`
	SnapshotAt   time.Time `json:"snapshot_at"`
	Features     []float64 `json:"features"`
	Label        int       `json:"label"`
	ClassWeight  float64   `json:"class_weight"`
}

type DataQualitySummary struct {
	TotalRows    int      `json:"total_rows"`
	TrainRows    int      `json:"train_rows"`
	TestRows     int      `json:"test_rows"`
	ChurnedRows  int      `json:"churned_rows"`
	RetainedRows int      `json:"retained_rows"`
	SkippedRows  int      `json:"skipped_rows"`
	Warnings     []string `json:"warnings,omitempty"`
}

func NewTrainingDataPreparer(deps TrainingDataPreparerDeps) *TrainingDataPreparer {
	return &TrainingDataPreparer{
		features:      deps.Features,
		subscriptions: deps.Subscriptions,
		events:        deps.Events,
	}
}

func (p *TrainingDataPreparer) Prepare(ctx context.Context, req TrainingDataRequest) (*TrainingDataset, error) {
	if p.features == nil || p.subscriptions == nil || p.events == nil {
		return nil, fmt.Errorf("training data preparer dependencies are required")
	}
	var err error
	req, err = normalizeTrainingDataRequest(req)
	if err != nil {
		return nil, err
	}

	snapshots, err := p.features.ListByOrgBetween(ctx, req.OrgID, req.From, req.To)
	if err != nil {
		return nil, fmt.Errorf("list training feature snapshots: %w", err)
	}
	sort.SliceStable(snapshots, func(i, j int) bool {
		if snapshots[i] == nil {
			return false
		}
		if snapshots[j] == nil {
			return true
		}
		return snapshots[i].CalculatedAt.Before(snapshots[j].CalculatedAt)
	})

	quality := DataQualitySummary{}
	rows := make([]TrainingDataRow, 0, len(snapshots))
	for _, snapshot := range snapshots {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !validTrainingSnapshot(snapshot, req) {
			quality.SkippedRows++
			continue
		}

		labelTo := snapshot.CalculatedAt.Add(req.LabelWindow)
		label, err := p.labelSnapshot(ctx, snapshot.CustomerID, snapshot.CalculatedAt, labelTo)
		if err != nil {
			return nil, err
		}
		rows = append(rows, trainingDataRowFromSnapshot(snapshot, label))
		countTrainingLabel(&quality, label)
	}

	weights := classWeights(quality.ChurnedRows, quality.RetainedRows)
	for i := range rows {
		rows[i].ClassWeight = weights[rows[i].Label]
	}

	train, test := splitTrainTest(rows)
	quality.TotalRows = len(rows)
	quality.TrainRows = len(train)
	quality.TestRows = len(test)
	quality.Warnings = qualityWarnings(quality)

	return &TrainingDataset{
		FeatureNames: FeatureNames(),
		Train:        train,
		Test:         test,
		ClassWeights: weights,
		Quality:      quality,
	}, nil
}

func normalizeTrainingDataRequest(req TrainingDataRequest) (TrainingDataRequest, error) {
	if req.OrgID == uuid.Nil {
		return req, fmt.Errorf("org id is required")
	}
	if req.To.IsZero() {
		req.To = time.Now().UTC()
	}
	if req.From.IsZero() {
		req.From = req.To.AddDate(-1, 0, 0)
	}
	if !req.From.Before(req.To) {
		return req, fmt.Errorf("training data window must have from before to")
	}
	if req.LabelWindow <= 0 {
		req.LabelWindow = defaultLabelWindow
	}
	return req, nil
}

func trainingDataRowFromSnapshot(snapshot *repository.CustomerFeature, label int) TrainingDataRow {
	return TrainingDataRow{
		OrgID:      snapshot.OrgID,
		CustomerID: snapshot.CustomerID,
		SnapshotAt: snapshot.CalculatedAt,
		Features:   valuesFromMap(snapshot.Features),
		Label:      label,
	}
}

func countTrainingLabel(quality *DataQualitySummary, label int) {
	if label == ChurnLabelChurned {
		quality.ChurnedRows++
		return
	}
	quality.RetainedRows++
}

func (p *TrainingDataPreparer) labelSnapshot(ctx context.Context, customerID uuid.UUID, from, to time.Time) (int, error) {
	subscriptions, err := p.subscriptions.ListByCustomer(ctx, customerID)
	if err != nil {
		return ChurnLabelRetained, fmt.Errorf("list customer subscriptions for label: %w", err)
	}
	for _, subscription := range subscriptions {
		if subscription == nil || subscription.CanceledAt == nil {
			continue
		}
		if inClosedWindow(*subscription.CanceledAt, from, to) {
			return ChurnLabelChurned, nil
		}
	}

	events, err := p.events.ListByCustomerBetween(ctx, customerID, from, to)
	if err != nil {
		return ChurnLabelRetained, fmt.Errorf("list customer events for label: %w", err)
	}
	for _, event := range events {
		if event == nil || event.EventType != eventMRRChanged || !inClosedWindow(event.OccurredAt, from, to) {
			continue
		}
		if numberFromEvent(event, "new_mrr_cents") < numberFromEvent(event, "old_mrr_cents") {
			return ChurnLabelChurned, nil
		}
	}
	return ChurnLabelRetained, nil
}

func (d TrainingDataset) ToJSON() ([]byte, error) {
	return json.Marshal(d)
}

func (d TrainingDataset) ToCSV() ([]byte, error) {
	var buf bytes.Buffer
	writer := csv.NewWriter(&buf)
	header := append([]string{"org_id", "customer_id", "snapshot_at"}, d.FeatureNames...)
	header = append(header, "label", "class_weight", "split")
	if err := writer.Write(header); err != nil {
		return nil, err
	}
	if err := writeTrainingRowsCSV(writer, d.FeatureNames, d.Train, "train"); err != nil {
		return nil, err
	}
	if err := writeTrainingRowsCSV(writer, d.FeatureNames, d.Test, "test"); err != nil {
		return nil, err
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeTrainingRowsCSV(writer *csv.Writer, featureNames []string, rows []TrainingDataRow, split string) error {
	for _, row := range rows {
		record := []string{row.OrgID.String(), row.CustomerID.String(), row.SnapshotAt.Format(time.RFC3339)}
		for i := range featureNames {
			value := 0.0
			if i < len(row.Features) {
				value = row.Features[i]
			}
			record = append(record, strconv.FormatFloat(value, 'f', -1, 64))
		}
		record = append(record, strconv.Itoa(row.Label), strconv.FormatFloat(row.ClassWeight, 'f', -1, 64), split)
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	return nil
}

func validTrainingSnapshot(snapshot *repository.CustomerFeature, req TrainingDataRequest) bool {
	if snapshot == nil || snapshot.OrgID != req.OrgID || snapshot.CustomerID == uuid.Nil || snapshot.Features == nil {
		return false
	}
	return !snapshot.CalculatedAt.Before(req.From) && !snapshot.CalculatedAt.After(req.To)
}

func valuesFromMap(features map[string]float64) []float64 {
	values := make([]float64, len(featureNames))
	for i, name := range featureNames {
		values[i] = clamp01(features[name])
	}
	return values
}

func classWeights(churned, retained int) map[int]float64 {
	weights := map[int]float64{ChurnLabelChurned: 0, ChurnLabelRetained: 0}
	total := churned + retained
	if total == 0 {
		return weights
	}
	if churned > 0 {
		weights[ChurnLabelChurned] = float64(total) / (2 * float64(churned))
	}
	if retained > 0 {
		weights[ChurnLabelRetained] = float64(total) / (2 * float64(retained))
	}
	return weights
}

func splitTrainTest(rows []TrainingDataRow) ([]TrainingDataRow, []TrainingDataRow) {
	if len(rows) < 2 {
		return append([]TrainingDataRow(nil), rows...), nil
	}
	split := int(float64(len(rows)) * 0.8)
	if split < 1 {
		split = 1
	}
	if split >= len(rows) {
		split = len(rows) - 1
	}
	return append([]TrainingDataRow(nil), rows[:split]...), append([]TrainingDataRow(nil), rows[split:]...)
}

func qualityWarnings(quality DataQualitySummary) []string {
	warnings := make([]string, 0, 2)
	if quality.TotalRows == 0 {
		warnings = append(warnings, "no valid training rows")
	}
	if quality.ChurnedRows == 0 || quality.RetainedRows == 0 {
		warnings = append(warnings, "training data has only one class")
	}
	return warnings
}

func inClosedWindow(at, from, to time.Time) bool {
	return !at.Before(from) && !at.After(to)
}
