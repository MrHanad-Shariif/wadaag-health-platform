-- name: CreatePatient :one
INSERT INTO patients (user_id, full_name, date_of_birth, sex, national_id, phone, address, next_of_kin)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: FindPatientByID :one
SELECT * FROM patients WHERE id = $1;

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
