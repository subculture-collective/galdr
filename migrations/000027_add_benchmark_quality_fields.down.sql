ALTER TABLE benchmark_aggregates
    DROP COLUMN IF EXISTS quality_level,
    DROP COLUMN IF EXISTS quality_score;
