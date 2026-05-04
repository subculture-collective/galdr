DROP INDEX IF EXISTS idx_organizations_industry;

ALTER TABLE organizations
    DROP CONSTRAINT IF EXISTS organizations_industry_check,
    DROP COLUMN IF EXISTS industry;
