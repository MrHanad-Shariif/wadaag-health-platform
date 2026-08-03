ALTER TABLE facilities DROP COLUMN IF EXISTS working_hours;
ALTER TABLE facilities DROP COLUMN IF EXISTS logo_url;

DROP INDEX IF EXISTS idx_departments_facility;
DROP TABLE IF EXISTS departments;

DROP INDEX IF EXISTS idx_branches_facility;
DROP TABLE IF EXISTS branches;
