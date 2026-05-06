UPDATE marketplace_connectors
SET status = 'submitted'
WHERE status IN ('under_review', 'rejected');

ALTER TABLE marketplace_connectors
DROP CONSTRAINT IF EXISTS marketplace_connectors_status_check;

ALTER TABLE marketplace_connectors
ADD CONSTRAINT marketplace_connectors_status_check
CHECK (status IN ('draft', 'submitted', 'approved', 'published', 'deprecated'));
