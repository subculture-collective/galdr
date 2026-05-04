ALTER TABLE organizations
    ADD COLUMN IF NOT EXISTS industry VARCHAR(50);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'organizations_industry_check'
    ) THEN
        ALTER TABLE organizations
            ADD CONSTRAINT organizations_industry_check
            CHECK (industry IS NULL OR industry IN ('SaaS', 'E-commerce', 'Fintech', 'Healthcare', 'Education', 'Media', 'Marketplace', 'Agency', 'Other'));
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_organizations_industry
    ON organizations (industry)
    WHERE industry IS NOT NULL;
