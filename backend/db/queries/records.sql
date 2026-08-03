-- name: CreatePatient :one
-- national_id is encrypted at rest (pgcrypto pgp_sym_encrypt) — see
-- db/migrations/0024_encrypt_sensitive_columns.up.sql. The key is supplied
-- by the application on every call, never a DB-level setting.
INSERT INTO patients (user_id, full_name, date_of_birth, sex, national_id, phone, address, next_of_kin)
VALUES ($1, $2, $3, $4, pgp_sym_encrypt(sqlc.narg(national_id)::text, sqlc.arg(encryption_key)::text), $5, $6, $7)
RETURNING id, user_id, full_name, date_of_birth, sex,
          CASE WHEN national_id IS NULL THEN NULL ELSE pgp_sym_decrypt(national_id, sqlc.arg(encryption_key)::text)::text END AS national_id,
          phone, address, next_of_kin, version, created_at, updated_at, gender, blood_group;

-- name: FindPatientByID :one
-- SELECT * would return national_id's raw encrypted bytes — explicit column
-- list so national_id can be decrypted in-query instead (see CreatePatient's
-- comment on why the key is a query param, not a DB-level setting).
SELECT id, user_id, full_name, date_of_birth, sex,
       CASE WHEN national_id IS NULL THEN NULL ELSE pgp_sym_decrypt(national_id, sqlc.arg(encryption_key)::text)::text END AS national_id,
       phone, address, next_of_kin, version, created_at, updated_at, gender, blood_group
FROM patients WHERE id = $1;

-- name: ListPatients :many
SELECT id, user_id, full_name, date_of_birth, sex,
       CASE WHEN national_id IS NULL THEN NULL ELSE pgp_sym_decrypt(national_id, sqlc.arg(encryption_key)::text)::text END AS national_id,
       phone, address, next_of_kin, version, created_at, updated_at, gender, blood_group
FROM patients ORDER BY created_at DESC;

-- name: FindPatientByUserID :one
-- The reverse of FindPatientByID: resolves a patient-role user's own
-- patient_id from their user_id, e.g. so a patient can book their own
-- appointment (see records.Service.GetOwnPatientRecord /
-- appointments.Service.Book).
SELECT id, user_id, full_name, date_of_birth, sex,
       CASE WHEN national_id IS NULL THEN NULL ELSE pgp_sym_decrypt(national_id, sqlc.arg(encryption_key)::text)::text END AS national_id,
       phone, address, next_of_kin, version, created_at, updated_at, gender, blood_group
FROM patients WHERE user_id = $1;

-- name: CountPatients :one
SELECT count(*) FROM patients;

-- name: CreateEncounter :one
INSERT INTO encounters (patient_id, facility_id, provider_id, type, notes, occurred_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: FindEncounterByID :one
SELECT * FROM encounters WHERE id = $1;

-- name: ListEncountersByPatient :many
SELECT * FROM encounters WHERE patient_id = $1 ORDER BY occurred_at DESC;

-- name: CreateClinicalObservation :one
INSERT INTO clinical_observations (encounter_id, type, payload, recorded_by)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListObservationsByEncounter :many
SELECT * FROM clinical_observations WHERE encounter_id = $1 ORDER BY recorded_at;

-- name: CountEncounters :one
SELECT count(*) FROM encounters;

-- name: CountEncountersForProviderToday :one
-- Powers the physician-specific dashboard view's "patients seen today"
-- count (see dashboard.Summary.PatientsTodayCount) — encounters this
-- provider recorded whose occurred_at falls on the current calendar date.
SELECT count(*) FROM encounters
WHERE provider_id = sqlc.arg(provider_id)::uuid AND occurred_at::date = current_date;

-- name: UpdatePatientGenderBloodGroup :one
UPDATE patients
SET gender = $2,
    blood_group = $3,
    updated_at = now()
WHERE id = $1
RETURNING id, user_id, full_name, date_of_birth, sex,
          CASE WHEN national_id IS NULL THEN NULL ELSE pgp_sym_decrypt(national_id, sqlc.arg(encryption_key)::text)::text END AS national_id,
          phone, address, next_of_kin, version, created_at, updated_at, gender, blood_group;

-- name: UpsertPatientMedicalHistory :one
-- Always a full replacement of all six jsonb columns (plus updated_by) —
-- no partial-update merge here, unlike UpsertUserProfile. See
-- Service.UpdateMedicalHistory. All six columns are encrypted at rest
-- (pgcrypto pgp_sym_encrypt) — see
-- db/migrations/0024_encrypt_sensitive_columns.up.sql. The Go layer passes
-- these in as raw JSON bytes ([]byte), same as before this migration;
-- encrypting the ::text form of that raw JSON and decrypting back to
-- ::text::jsonb on the way out keeps that []byte shape unchanged from the
-- caller's perspective.
INSERT INTO patient_medical_history (
    patient_id, allergies, chronic_conditions, current_medications,
    past_surgeries, family_history, vaccination_history, updated_by
)
VALUES (
    $1,
    pgp_sym_encrypt($2::text, sqlc.arg(encryption_key)::text),
    pgp_sym_encrypt($3::text, sqlc.arg(encryption_key)::text),
    pgp_sym_encrypt($4::text, sqlc.arg(encryption_key)::text),
    pgp_sym_encrypt($5::text, sqlc.arg(encryption_key)::text),
    pgp_sym_encrypt($6::text, sqlc.arg(encryption_key)::text),
    pgp_sym_encrypt($7::text, sqlc.arg(encryption_key)::text),
    $8
)
ON CONFLICT (patient_id) DO UPDATE
SET allergies = EXCLUDED.allergies,
    chronic_conditions = EXCLUDED.chronic_conditions,
    current_medications = EXCLUDED.current_medications,
    past_surgeries = EXCLUDED.past_surgeries,
    family_history = EXCLUDED.family_history,
    vaccination_history = EXCLUDED.vaccination_history,
    updated_by = EXCLUDED.updated_by,
    updated_at = now()
RETURNING id, patient_id,
          pgp_sym_decrypt(allergies, sqlc.arg(encryption_key)::text)::text::jsonb AS allergies,
          pgp_sym_decrypt(chronic_conditions, sqlc.arg(encryption_key)::text)::text::jsonb AS chronic_conditions,
          pgp_sym_decrypt(current_medications, sqlc.arg(encryption_key)::text)::text::jsonb AS current_medications,
          pgp_sym_decrypt(past_surgeries, sqlc.arg(encryption_key)::text)::text::jsonb AS past_surgeries,
          pgp_sym_decrypt(family_history, sqlc.arg(encryption_key)::text)::text::jsonb AS family_history,
          pgp_sym_decrypt(vaccination_history, sqlc.arg(encryption_key)::text)::text::jsonb AS vaccination_history,
          updated_by, created_at, updated_at;

-- name: FindPatientMedicalHistoryByPatientID :one
-- Returns pgx.ErrNoRows if no row exists yet — expected (a patient with no
-- medical history recorded), handled in the repository layer, not an error
-- condition.
SELECT id, patient_id,
       pgp_sym_decrypt(allergies, sqlc.arg(encryption_key)::text)::text::jsonb AS allergies,
       pgp_sym_decrypt(chronic_conditions, sqlc.arg(encryption_key)::text)::text::jsonb AS chronic_conditions,
       pgp_sym_decrypt(current_medications, sqlc.arg(encryption_key)::text)::text::jsonb AS current_medications,
       pgp_sym_decrypt(past_surgeries, sqlc.arg(encryption_key)::text)::text::jsonb AS past_surgeries,
       pgp_sym_decrypt(family_history, sqlc.arg(encryption_key)::text)::text::jsonb AS family_history,
       pgp_sym_decrypt(vaccination_history, sqlc.arg(encryption_key)::text)::text::jsonb AS vaccination_history,
       updated_by, created_at, updated_at
FROM patient_medical_history WHERE patient_id = $1;

-- name: SearchPatientsUnscoped :many
-- Unscoped patient-directory search — same "bypasses consent grants"
-- caveat as ListPatients, so callers must gate this to system_admin only
-- (see records.Service.SearchPatients / search.Service.Search).
-- national_id is deliberately NOT matched here: it's encrypted at rest with
-- a non-deterministic cipher (pgp_sym_encrypt produces different ciphertext
-- for the same plaintext on every call), so there's no way to ILIKE-match
-- or even exact-match it without a separate deterministic hash column,
-- which is out of scope for this task. This is an accepted capability loss
-- — national_id is no longer searchable, only full_name/phone are.
SELECT id, user_id, full_name, date_of_birth, sex,
       CASE WHEN national_id IS NULL THEN NULL ELSE pgp_sym_decrypt(national_id, sqlc.arg(encryption_key)::text)::text END AS national_id,
       phone, address, next_of_kin, version, created_at, updated_at, gender, blood_group
FROM patients
WHERE full_name ILIKE '%' || sqlc.arg(query)::text || '%'
   OR phone ILIKE '%' || sqlc.arg(query)::text || '%'
ORDER BY similarity(full_name, sqlc.arg(query)::text) DESC, created_at DESC
LIMIT sqlc.arg(result_limit)::int;

-- name: SearchPatientsForFacility :many
-- Facility-scoped patient search: only patients this facility already has
-- an active, unexpired consent grant for (the same boundary
-- consent.Checker.HasAccess enforces for a single patient) are
-- discoverable — a physician/hospital_admin cannot use search to find a
-- patient at another facility they have no standing or referral-driven
-- access to. See records.Service.SearchPatients.
-- national_id is deliberately NOT matched here — see SearchPatientsUnscoped's
-- comment on why an encrypted column can't support partial/exact-match
-- search without a separate deterministic hash column (out of scope).
SELECT p.id, p.user_id, p.full_name, p.date_of_birth, p.sex,
       CASE WHEN p.national_id IS NULL THEN NULL ELSE pgp_sym_decrypt(p.national_id, sqlc.arg(encryption_key)::text)::text END AS national_id,
       p.phone, p.address, p.next_of_kin, p.version, p.created_at, p.updated_at, p.gender, p.blood_group
FROM patients p
JOIN consent_grants cg ON cg.patient_id = p.id
WHERE cg.grantee_type = 'facility'
  AND cg.grantee_id = sqlc.arg(facility_id)::uuid
  AND cg.status = 'active'
  AND (cg.expires_at IS NULL OR cg.expires_at > now())
  AND (
    p.full_name ILIKE '%' || sqlc.arg(query)::text || '%'
    OR p.phone ILIKE '%' || sqlc.arg(query)::text || '%'
  )
GROUP BY p.id
ORDER BY similarity(p.full_name, sqlc.arg(query)::text) DESC, p.created_at DESC
LIMIT sqlc.arg(result_limit)::int;
