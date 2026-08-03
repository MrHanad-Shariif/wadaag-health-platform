DROP INDEX IF EXISTS idx_consultations_reason_trgm;
DROP INDEX IF EXISTS idx_referrals_reason_trgm;
DROP INDEX IF EXISTS idx_providers_specialty_trgm;
DROP INDEX IF EXISTS idx_facilities_name_trgm;
DROP INDEX IF EXISTS idx_users_full_name_trgm;
DROP INDEX IF EXISTS idx_patients_full_name_trgm;
DROP EXTENSION IF EXISTS pg_trgm;
