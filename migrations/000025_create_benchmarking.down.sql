DROP TABLE IF EXISTS benchmark_aggregates;
DROP TABLE IF EXISTS benchmark_contributions;

ALTER TABLE organizations
    DROP COLUMN IF EXISTS benchmarking_enabled,
    DROP COLUMN IF EXISTS industry,
    DROP COLUMN IF EXISTS company_size;
