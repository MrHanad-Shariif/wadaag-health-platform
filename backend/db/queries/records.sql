-- name: CreatePatient :one
INSERT INTO patients (user_id, full_name, date_of_birth, sex, national_id, phone, address, next_of_kin)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: FindPatientByID :one
SELECT * FROM patients WHERE id = $1;

-- name: ListPatients :many
SELECT * FROM patients ORDER BY created_at DESC;

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

-- name: UpdatePatientGenderBloodGroup :one
UPDATE patients
SET gender = $2,
    blood_group = $3,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpsertPatientMedicalHistory :one
-- Always a full replacement of all six jsonb columns (plus updated_by) —
-- no partial-update merge here, unlike UpsertUserProfile. See
-- Service.UpdateMedicalHistory.
INSERT INTO patient_medical_history (
    patient_id, allergies, chronic_conditions, current_medications,
    past_surgeries, family_history, vaccination_history, updated_by
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (patient_id) DO UPDATE
SET allergies = EXCLUDED.allergies,
    chronic_conditions = EXCLUDED.chronic_conditions,
    current_medications = EXCLUDED.current_medications,
    past_surgeries = EXCLUDED.past_surgeries,
    family_history = EXCLUDED.family_history,
    vaccination_history = EXCLUDED.vaccination_history,
    updated_by = EXCLUDED.updated_by,
    updated_at = now()
RETURNING *;

-- name: FindPatientMedicalHistoryByPatientID :one
-- Returns pgx.ErrNoRows if no row exists yet — expected (a patient with no
-- medical history recorded), handled in the repository layer, not an error
-- condition.
SELECT * FROM patient_medical_history WHERE patient_id = $1;
