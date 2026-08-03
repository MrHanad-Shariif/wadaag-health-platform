-- Reverse of 0024_encrypt_sensitive_columns.up.sql — decrypts back to
-- plaintext TEXT/JSONB using the same dev-default literal key (see the up
-- migration's comment for why this literal, rather than a real secret, is
-- checked into source here).

ALTER TABLE patients ADD COLUMN national_id_plain TEXT;

UPDATE patients
SET national_id_plain = pgp_sym_decrypt(national_id, 'dev-only-insecure-encryption-key-do-not-use-in-production')::text
WHERE national_id IS NOT NULL;

ALTER TABLE patients DROP COLUMN national_id;
ALTER TABLE patients RENAME COLUMN national_id_plain TO national_id;

ALTER TABLE patient_medical_history ADD COLUMN allergies_plain JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE patient_medical_history ADD COLUMN chronic_conditions_plain JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE patient_medical_history ADD COLUMN current_medications_plain JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE patient_medical_history ADD COLUMN past_surgeries_plain JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE patient_medical_history ADD COLUMN family_history_plain JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE patient_medical_history ADD COLUMN vaccination_history_plain JSONB NOT NULL DEFAULT '[]'::jsonb;

UPDATE patient_medical_history
SET allergies_plain = pgp_sym_decrypt(allergies, 'dev-only-insecure-encryption-key-do-not-use-in-production')::text::jsonb,
    chronic_conditions_plain = pgp_sym_decrypt(chronic_conditions, 'dev-only-insecure-encryption-key-do-not-use-in-production')::text::jsonb,
    current_medications_plain = pgp_sym_decrypt(current_medications, 'dev-only-insecure-encryption-key-do-not-use-in-production')::text::jsonb,
    past_surgeries_plain = pgp_sym_decrypt(past_surgeries, 'dev-only-insecure-encryption-key-do-not-use-in-production')::text::jsonb,
    family_history_plain = pgp_sym_decrypt(family_history, 'dev-only-insecure-encryption-key-do-not-use-in-production')::text::jsonb,
    vaccination_history_plain = pgp_sym_decrypt(vaccination_history, 'dev-only-insecure-encryption-key-do-not-use-in-production')::text::jsonb;

ALTER TABLE patient_medical_history DROP COLUMN allergies;
ALTER TABLE patient_medical_history DROP COLUMN chronic_conditions;
ALTER TABLE patient_medical_history DROP COLUMN current_medications;
ALTER TABLE patient_medical_history DROP COLUMN past_surgeries;
ALTER TABLE patient_medical_history DROP COLUMN family_history;
ALTER TABLE patient_medical_history DROP COLUMN vaccination_history;

ALTER TABLE patient_medical_history RENAME COLUMN allergies_plain TO allergies;
ALTER TABLE patient_medical_history RENAME COLUMN chronic_conditions_plain TO chronic_conditions;
ALTER TABLE patient_medical_history RENAME COLUMN current_medications_plain TO current_medications;
ALTER TABLE patient_medical_history RENAME COLUMN past_surgeries_plain TO past_surgeries;
ALTER TABLE patient_medical_history RENAME COLUMN family_history_plain TO family_history;
ALTER TABLE patient_medical_history RENAME COLUMN vaccination_history_plain TO vaccination_history;
