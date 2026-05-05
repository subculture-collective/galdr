ALTER TABLE benchmark_aggregates
    ADD COLUMN quality_score NUMERIC(5,2) NOT NULL DEFAULT 0 CHECK (quality_score >= 0 AND quality_score <= 100),
    ADD COLUMN quality_level VARCHAR(20) NOT NULL DEFAULT 'low' CHECK (quality_level IN ('low', 'medium', 'high'));
