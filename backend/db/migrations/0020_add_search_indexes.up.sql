-- Trigram indexes backing internal/search's ILIKE/similarity() queries —
-- Postgres full-text search via pg_trgm rather than an external search
-- service, per the roadmap. If pg_trgm cannot be enabled in some
-- environment, the search queries still work (just unindexed, slower);
-- this migration is not load-bearing for correctness, only performance.
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX idx_patients_full_name_trgm ON patients USING gin (full_name gin_trgm_ops);
CREATE INDEX idx_users_full_name_trgm ON users USING gin (full_name gin_trgm_ops);
CREATE INDEX idx_facilities_name_trgm ON facilities USING gin (name gin_trgm_ops);
CREATE INDEX idx_providers_specialty_trgm ON providers USING gin (specialty gin_trgm_ops);
CREATE INDEX idx_referrals_reason_trgm ON referrals USING gin (reason gin_trgm_ops);
CREATE INDEX idx_consultations_reason_trgm ON consultations USING gin (reason gin_trgm_ops);
