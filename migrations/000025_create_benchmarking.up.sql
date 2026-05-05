ALTER TABLE organizations
    ADD COLUMN benchmarking_enabled BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN company_size INTEGER NOT NULL DEFAULT 0 CHECK (company_size >= 0);

CREATE TABLE benchmark_contributions (
    id                    UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    org_id                UUID NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    industry              VARCHAR(100) NOT NULL,
    company_size_bucket   VARCHAR(20) NOT NULL CHECK (company_size_bucket IN ('1-10', '11-50', '51-200', '201-1000', '1000+')),
    avg_health_score      NUMERIC(5,2) NOT NULL CHECK (avg_health_score >= 0 AND avg_health_score <= 100),
    avg_mrr               BIGINT NOT NULL CHECK (avg_mrr >= 0),
    avg_churn_rate        NUMERIC(5,4) NOT NULL CHECK (avg_churn_rate >= 0 AND avg_churn_rate <= 1),
    customer_count_bucket VARCHAR(20) NOT NULL CHECK (customer_count_bucket IN ('1-10', '11-50', '51-200', '201-1000', '1000+')),
    contributed_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_benchmark_contributions_org_time ON benchmark_contributions (org_id, contributed_at DESC);
CREATE INDEX idx_benchmark_contributions_segment ON benchmark_contributions (industry, company_size_bucket, contributed_at DESC);

CREATE TABLE benchmark_aggregates (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    industry            VARCHAR(100) NOT NULL,
    company_size_bucket VARCHAR(20) NOT NULL CHECK (company_size_bucket IN ('1-10', '11-50', '51-200', '201-1000', '1000+')),
    metric_name         VARCHAR(50) NOT NULL CHECK (metric_name IN ('avg_health_score', 'avg_mrr', 'avg_churn_rate')),
    p25                 NUMERIC(12,4) NOT NULL,
    p50                 NUMERIC(12,4) NOT NULL,
    p75                 NUMERIC(12,4) NOT NULL,
    p90                 NUMERIC(12,4) NOT NULL,
    sample_count        INTEGER NOT NULL CHECK (sample_count >= 0),
    calculated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_benchmark_aggregates_segment_metric ON benchmark_aggregates (industry, company_size_bucket, metric_name, calculated_at DESC);
