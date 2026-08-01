-- name: CreateReferral :one
INSERT INTO referrals (
    patient_id, referring_provider_id, referring_facility_id, receiving_facility_id,
    specialty_requested, urgency, reason, clinical_summary_encounter_id, status
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'routed')
RETURNING *;

-- name: FindReferralByID :one
SELECT * FROM referrals WHERE id = $1;

-- name: ListReferralsForFacility :many
SELECT * FROM referrals WHERE receiving_facility_id = $1 ORDER BY created_at DESC;

-- name: ListReferralsByPatient :many
SELECT * FROM referrals WHERE patient_id = $1 ORDER BY created_at DESC;

-- name: AcceptReferral :one
UPDATE referrals
SET status = 'accepted', receiving_provider_id = $3, version = version + 1, updated_at = now()
WHERE id = $1 AND version = $2 AND status = 'routed'
RETURNING *;

-- name: UpdateReferralStatus :one
UPDATE referrals
SET status = $3, version = version + 1, updated_at = now()
WHERE id = $1 AND version = $2
RETURNING *;

-- name: CreateReferralStatusEvent :one
INSERT INTO referral_status_events (referral_id, from_status, to_status, actor_user_id, note)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListReferralStatusEvents :many
SELECT * FROM referral_status_events WHERE referral_id = $1 ORDER BY occurred_at;
