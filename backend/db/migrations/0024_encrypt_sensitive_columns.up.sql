-- Phase 10 security hardening: application-layer column encryption via
-- pgcrypto, on top of whatever the eventual hosting provider does at the
-- disk/volume level (that broader question is tracked separately). Scope is
-- deliberately narrow: patients.national_id and patient_medical_history's
-- six jsonb columns. Nothing else (phone/address/next_of_kin etc.) changes.
--
-- The encryption key itself is supplied by the Go application as a query
-- parameter on every read/write (see internal/platform/config.go's
-- Config.EncryptionKey, sourced from ENCRYPTION_KEY) — never a DB-level GUC,
-- never hardcoded in the application's own queries. This migration is the
-- one deliberate exception: a plain .up.sql file has no way to read an
-- environment variable, and this environment already has seeded/test data
-- in patients/patient_medical_history that needs re-encrypting in place, not
-- discarding. Rather than invent a migration-time parameter-substitution
-- mechanism (the migrate/migrate CLI used by deploy/docker-compose.yml has
-- none), this literal is the same dev-only default Config.EncryptionKey
-- falls back to when ENCRYPTION_KEY is unset outside production — the exact
-- same precedent as JWT_SECRET's dev-default literal in config.go. This is
-- fine for dev/test data encrypted with the known dev-default key; it is
-- not a production migration carrying a real secret. If ENCRYPTION_KEY is
-- ever set to something other than this literal before this migration runs
-- against real data, this migration must be edited first (or the data
-- re-encrypted after the fact) — same operational rule that already applies
-- to rotating JWT_SECRET against existing sessions.
--
-- literal below: dev-only-insecure-encryption-key-do-not-use-in-production

CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- patients.national_id: TEXT -> BYTEA (pgp_sym_encrypt output)
ALTER TABLE patients ADD COLUMN national_id_enc BYTEA;

UPDATE patients
SET national_id_enc = pgp_sym_encrypt(national_id, 'dev-only-insecure-encryption-key-do-not-use-in-production')
WHERE national_id IS NOT NULL;

ALTER TABLE patients DROP COLUMN national_id;
ALTER TABLE patients RENAME COLUMN national_id_enc TO national_id;

-- patient_medical_history's six jsonb columns: JSONB -> BYTEA. Each column
-- is NOT NULL DEFAULT '[]'::jsonb today (see 0013_extend_patients.up.sql),
-- so every existing row has a non-null value to re-encrypt; no DB-level
-- DEFAULT is carried forward onto the new bytea columns because the
-- application always writes all six fields on every insert/update (a full
-- replacement, never a partial one — see records.Service.UpdateMedicalHistory)
-- so a DB-level default encrypted placeholder is unnecessary.
ALTER TABLE patient_medical_history ADD COLUMN allergies_enc BYTEA;
ALTER TABLE patient_medical_history ADD COLUMN chronic_conditions_enc BYTEA;
ALTER TABLE patient_medical_history ADD COLUMN current_medications_enc BYTEA;
ALTER TABLE patient_medical_history ADD COLUMN past_surgeries_enc BYTEA;
ALTER TABLE patient_medical_history ADD COLUMN family_history_enc BYTEA;
ALTER TABLE patient_medical_history ADD COLUMN vaccination_history_enc BYTEA;

UPDATE patient_medical_history
SET allergies_enc = pgp_sym_encrypt(allergies::text, 'dev-only-insecure-encryption-key-do-not-use-in-production'),
    chronic_conditions_enc = pgp_sym_encrypt(chronic_conditions::text, 'dev-only-insecure-encryption-key-do-not-use-in-production'),
    current_medications_enc = pgp_sym_encrypt(current_medications::text, 'dev-only-insecure-encryption-key-do-not-use-in-production'),
    past_surgeries_enc = pgp_sym_encrypt(past_surgeries::text, 'dev-only-insecure-encryption-key-do-not-use-in-production'),
    family_history_enc = pgp_sym_encrypt(family_history::text, 'dev-only-insecure-encryption-key-do-not-use-in-production'),
    vaccination_history_enc = pgp_sym_encrypt(vaccination_history::text, 'dev-only-insecure-encryption-key-do-not-use-in-production');

ALTER TABLE patient_medical_history ALTER COLUMN allergies_enc SET NOT NULL;
ALTER TABLE patient_medical_history ALTER COLUMN chronic_conditions_enc SET NOT NULL;
ALTER TABLE patient_medical_history ALTER COLUMN current_medications_enc SET NOT NULL;
ALTER TABLE patient_medical_history ALTER COLUMN past_surgeries_enc SET NOT NULL;
ALTER TABLE patient_medical_history ALTER COLUMN family_history_enc SET NOT NULL;
ALTER TABLE patient_medical_history ALTER COLUMN vaccination_history_enc SET NOT NULL;

ALTER TABLE patient_medical_history DROP COLUMN allergies;
ALTER TABLE patient_medical_history DROP COLUMN chronic_conditions;
ALTER TABLE patient_medical_history DROP COLUMN current_medications;
ALTER TABLE patient_medical_history DROP COLUMN past_surgeries;
ALTER TABLE patient_medical_history DROP COLUMN family_history;
ALTER TABLE patient_medical_history DROP COLUMN vaccination_history;

ALTER TABLE patient_medical_history RENAME COLUMN allergies_enc TO allergies;
ALTER TABLE patient_medical_history RENAME COLUMN chronic_conditions_enc TO chronic_conditions;
ALTER TABLE patient_medical_history RENAME COLUMN current_medications_enc TO current_medications;
ALTER TABLE patient_medical_history RENAME COLUMN past_surgeries_enc TO past_surgeries;
ALTER TABLE patient_medical_history RENAME COLUMN family_history_enc TO family_history;
ALTER TABLE patient_medical_history RENAME COLUMN vaccination_history_enc TO vaccination_history;
