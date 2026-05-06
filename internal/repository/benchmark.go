package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	BenchmarkBucket1To10     = "1-10"
	BenchmarkBucket11To50    = "11-50"
	BenchmarkBucket51To200   = "51-200"
	BenchmarkBucket201To1000 = "201-1000"
	BenchmarkBucket1000Plus  = "1000+"
)

const (
	BenchmarkMetricHealthScore      = "health_score"
	BenchmarkMetricMRRPerCustomer   = "mrr_per_customer"
	BenchmarkMetricChurnRate        = "churn_rate"
	BenchmarkMetricIntegrationUsage = "integration_usage"
)

type BenchmarkContribution struct {
	ID                     uuid.UUID `json:"id"`
	OrgID                  uuid.UUID `json:"org_id"`
	Industry               string    `json:"industry"`
	CompanySizeBucket      string    `json:"company_size_bucket"`
	AvgHealthScore         float64   `json:"avg_health_score"`
	AvgMRR                 int64     `json:"avg_mrr"`
	AvgChurnRate           float64   `json:"avg_churn_rate"`
	ActiveIntegrationCount int       `json:"active_integration_count"`
	CustomerCountBucket    string    `json:"customer_count_bucket"`
	ContributedAt          time.Time `json:"contributed_at"`
}

type BenchmarkContributionPair struct {
	Previous BenchmarkContribution `json:"previous"`
	Current  BenchmarkContribution `json:"current"`
}

type BenchmarkAggregate struct {
	ID                uuid.UUID `json:"id"`
	Industry          string    `json:"industry"`
	CompanySizeBucket string    `json:"company_size_bucket"`
	MetricName        string    `json:"metric_name"`
	P25               float64   `json:"p25"`
	P50               float64   `json:"p50"`
	P75               float64   `json:"p75"`
	P90               float64   `json:"p90"`
	SampleCount       int       `json:"sample_count"`
	QualityScore      float64   `json:"quality_score"`
	QualityLevel      string    `json:"quality_level"`
	CalculatedAt      time.Time `json:"calculated_at"`
}

type BenchmarkRepository struct {
	pool *pgxpool.Pool
}

func NewBenchmarkRepository(pool *pgxpool.Pool) *BenchmarkRepository {
	return &BenchmarkRepository{pool: pool}
}

func (r *BenchmarkRepository) CreateContribution(ctx context.Context, contribution *BenchmarkContribution) error {
	if contribution.ID == uuid.Nil {
		contribution.ID = uuid.New()
	}
	if contribution.ContributedAt.IsZero() {
		contribution.ContributedAt = time.Now().UTC()
	}

	query := `
		INSERT INTO benchmark_contributions (
			id, org_id, industry, company_size_bucket, avg_health_score,
			avg_mrr, avg_churn_rate, active_integration_count,
			customer_count_bucket, contributed_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING contributed_at`

	if err := r.pool.QueryRow(ctx, query,
		contribution.ID,
		contribution.OrgID,
		contribution.Industry,
		contribution.CompanySizeBucket,
		contribution.AvgHealthScore,
		contribution.AvgMRR,
		contribution.AvgChurnRate,
		contribution.ActiveIntegrationCount,
		contribution.CustomerCountBucket,
		contribution.ContributedAt,
	).Scan(&contribution.ContributedAt); err != nil {
		return fmt.Errorf("create benchmark contribution: %w", err)
	}

	return nil
}

func (r *BenchmarkRepository) ListLatestContributions(ctx context.Context) ([]BenchmarkContribution, error) {
	query := `
		SELECT DISTINCT ON (org_id)
			id, org_id, industry, company_size_bucket, avg_health_score,
			avg_mrr, avg_churn_rate, active_integration_count,
			customer_count_bucket, contributed_at
		FROM benchmark_contributions
		ORDER BY org_id, contributed_at DESC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list latest benchmark contributions: %w", err)
	}
	defer rows.Close()

	var contributions []BenchmarkContribution
	for rows.Next() {
		var contribution BenchmarkContribution
		if err := rows.Scan(
			&contribution.ID,
			&contribution.OrgID,
			&contribution.Industry,
			&contribution.CompanySizeBucket,
			&contribution.AvgHealthScore,
			&contribution.AvgMRR,
			&contribution.AvgChurnRate,
			&contribution.ActiveIntegrationCount,
			&contribution.CustomerCountBucket,
			&contribution.ContributedAt,
		); err != nil {
			return nil, fmt.Errorf("scan latest benchmark contribution: %w", err)
		}
		contributions = append(contributions, contribution)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate latest benchmark contributions: %w", err)
	}

	return contributions, nil
}

func (r *BenchmarkRepository) ListLatestContributionPairs(ctx context.Context) ([]BenchmarkContributionPair, error) {
	query := `
		WITH ranked AS (
			SELECT
				id, org_id, industry, company_size_bucket, avg_health_score,
				avg_mrr, avg_churn_rate, active_integration_count,
				customer_count_bucket, contributed_at,
				ROW_NUMBER() OVER (PARTITION BY org_id ORDER BY contributed_at DESC) AS rn
			FROM benchmark_contributions
		)
		SELECT
			prev.id, prev.org_id, prev.industry, prev.company_size_bucket, prev.avg_health_score,
			prev.avg_mrr, prev.avg_churn_rate, prev.active_integration_count,
			prev.customer_count_bucket, prev.contributed_at,
			curr.id, curr.org_id, curr.industry, curr.company_size_bucket, curr.avg_health_score,
			curr.avg_mrr, curr.avg_churn_rate, curr.active_integration_count,
			curr.customer_count_bucket, curr.contributed_at
		FROM ranked curr
		JOIN ranked prev ON prev.org_id = curr.org_id AND prev.rn = 2
		WHERE curr.rn = 1`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list latest benchmark contribution pairs: %w", err)
	}
	defer rows.Close()

	pairs := []BenchmarkContributionPair{}
	for rows.Next() {
		var pair BenchmarkContributionPair
		if err := rows.Scan(
			&pair.Previous.ID,
			&pair.Previous.OrgID,
			&pair.Previous.Industry,
			&pair.Previous.CompanySizeBucket,
			&pair.Previous.AvgHealthScore,
			&pair.Previous.AvgMRR,
			&pair.Previous.AvgChurnRate,
			&pair.Previous.ActiveIntegrationCount,
			&pair.Previous.CustomerCountBucket,
			&pair.Previous.ContributedAt,
			&pair.Current.ID,
			&pair.Current.OrgID,
			&pair.Current.Industry,
			&pair.Current.CompanySizeBucket,
			&pair.Current.AvgHealthScore,
			&pair.Current.AvgMRR,
			&pair.Current.AvgChurnRate,
			&pair.Current.ActiveIntegrationCount,
			&pair.Current.CustomerCountBucket,
			&pair.Current.ContributedAt,
		); err != nil {
			return nil, fmt.Errorf("scan benchmark contribution pair: %w", err)
		}
		pairs = append(pairs, pair)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate benchmark contribution pairs: %w", err)
	}
	return pairs, nil
}

func (r *BenchmarkRepository) DeleteContributionsByOrg(ctx context.Context, orgID uuid.UUID) error {
	query := `DELETE FROM benchmark_contributions WHERE org_id = $1`
	if _, err := r.pool.Exec(ctx, query, orgID); err != nil {
		return fmt.Errorf("delete benchmark contributions by org: %w", err)
	}
	return nil
}

func (r *BenchmarkRepository) CreateAggregate(ctx context.Context, aggregate *BenchmarkAggregate) error {
	if aggregate.ID == uuid.Nil {
		aggregate.ID = uuid.New()
	}
	if aggregate.CalculatedAt.IsZero() {
		aggregate.CalculatedAt = time.Now().UTC()
	}

	query := `
		INSERT INTO benchmark_aggregates (
			id, industry, company_size_bucket, metric_name, p25, p50,
			p75, p90, sample_count, quality_score, quality_level, calculated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING calculated_at`

	if err := r.pool.QueryRow(ctx, query,
		aggregate.ID,
		aggregate.Industry,
		aggregate.CompanySizeBucket,
		aggregate.MetricName,
		aggregate.P25,
		aggregate.P50,
		aggregate.P75,
		aggregate.P90,
		aggregate.SampleCount,
		aggregate.QualityScore,
		aggregate.QualityLevel,
		aggregate.CalculatedAt,
	).Scan(&aggregate.CalculatedAt); err != nil {
		return fmt.Errorf("create benchmark aggregate: %w", err)
	}

	return nil
}

func (r *BenchmarkRepository) ListLatestAggregates(ctx context.Context, industry, companySizeBucket string) ([]BenchmarkAggregate, error) {
	query := `
		SELECT DISTINCT ON (metric_name)
			id, industry, company_size_bucket, metric_name, p25, p50,
			p75, p90, sample_count, quality_score, quality_level, calculated_at
		FROM benchmark_aggregates
		WHERE industry = $1 AND company_size_bucket = $2
		ORDER BY metric_name, calculated_at DESC`

	rows, err := r.pool.Query(ctx, query, industry, companySizeBucket)
	if err != nil {
		return nil, fmt.Errorf("list latest benchmark aggregates: %w", err)
	}
	defer rows.Close()

	aggregates := []BenchmarkAggregate{}
	for rows.Next() {
		var aggregate BenchmarkAggregate
		if err := rows.Scan(
			&aggregate.ID,
			&aggregate.Industry,
			&aggregate.CompanySizeBucket,
			&aggregate.MetricName,
			&aggregate.P25,
			&aggregate.P50,
			&aggregate.P75,
			&aggregate.P90,
			&aggregate.SampleCount,
			&aggregate.QualityScore,
			&aggregate.QualityLevel,
			&aggregate.CalculatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan latest benchmark aggregate: %w", err)
		}
		aggregates = append(aggregates, aggregate)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate latest benchmark aggregates: %w", err)
	}

	return aggregates, nil
}

func (r *BenchmarkRepository) ListAllLatestAggregates(ctx context.Context) ([]BenchmarkAggregate, error) {
	query := `
		SELECT DISTINCT ON (industry, company_size_bucket, metric_name)
			id, industry, company_size_bucket, metric_name, p25, p50,
			p75, p90, sample_count, quality_score, quality_level, calculated_at
		FROM benchmark_aggregates
		ORDER BY industry, company_size_bucket, metric_name, calculated_at DESC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list latest benchmark aggregates: %w", err)
	}
	defer rows.Close()

	aggregates := []BenchmarkAggregate{}
	for rows.Next() {
		var aggregate BenchmarkAggregate
		if err := rows.Scan(
			&aggregate.ID,
			&aggregate.Industry,
			&aggregate.CompanySizeBucket,
			&aggregate.MetricName,
			&aggregate.P25,
			&aggregate.P50,
			&aggregate.P75,
			&aggregate.P90,
			&aggregate.SampleCount,
			&aggregate.QualityScore,
			&aggregate.QualityLevel,
			&aggregate.CalculatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan latest benchmark aggregate: %w", err)
		}
		aggregates = append(aggregates, aggregate)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate latest benchmark aggregates: %w", err)
	}
	return aggregates, nil
}

type BenchmarkMetricsRepository struct {
	customers    *CustomerRepository
	healthScores *HealthScoreRepository
	integrations *IntegrationConnectionRepository
}

func NewBenchmarkMetricsRepository(customers *CustomerRepository, healthScores *HealthScoreRepository, integrations *IntegrationConnectionRepository) *BenchmarkMetricsRepository {
	return &BenchmarkMetricsRepository{customers: customers, healthScores: healthScores, integrations: integrations}
}

func (r *BenchmarkMetricsRepository) CountCustomers(ctx context.Context, orgID uuid.UUID) (int, error) {
	return r.customers.CountByOrg(ctx, orgID)
}

func (r *BenchmarkMetricsRepository) TotalMRR(ctx context.Context, orgID uuid.UUID) (int64, error) {
	return r.customers.TotalMRRByOrg(ctx, orgID)
}

func (r *BenchmarkMetricsRepository) AverageHealthScore(ctx context.Context, orgID uuid.UUID) (float64, error) {
	return r.healthScores.GetAverageScore(ctx, orgID)
}

func (r *BenchmarkMetricsRepository) ChurnRate(ctx context.Context, orgID uuid.UUID) (float64, error) {
	query := `
		SELECT
			CASE WHEN COUNT(*) = 0 THEN 0
			ELSE COUNT(*) FILTER (WHERE risk_level = 'red')::float / COUNT(*)::float
			END
		FROM health_scores
		WHERE org_id = $1`
	var rate float64
	if err := r.healthScores.pool.QueryRow(ctx, query, orgID).Scan(&rate); err != nil {
		return 0, fmt.Errorf("get benchmark churn rate: %w", err)
	}
	return rate, nil
}

func (r *BenchmarkMetricsRepository) ActiveIntegrationCount(ctx context.Context, orgID uuid.UUID) (int, error) {
	return r.integrations.CountActiveByOrg(ctx, orgID)
}
