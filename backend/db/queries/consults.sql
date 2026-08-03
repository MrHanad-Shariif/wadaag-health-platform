-- name: CreateConsultation :one
INSERT INTO consultations (
    patient_id, requesting_provider_id, target_provider_id, reason
)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: FindConsultationByID :one
SELECT * FROM consultations WHERE id = $1;

-- name: ListConsultationsForProvider :many
SELECT * FROM consultations
WHERE requesting_provider_id = $1 OR target_provider_id = $1
ORDER BY updated_at DESC;

-- name: ListAllConsultations :many
-- Platform-wide oversight listing — system_admin has no provider identity
-- to scope ListConsultationsForProvider by, so its inbox view needs an
-- unscoped list instead.
SELECT * FROM consultations ORDER BY updated_at DESC;

-- name: UpdateConsultationStatus :one
UPDATE consultations
SET status = $2, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: CreateConsultationMessage :one
INSERT INTO consultation_messages (
    consultation_id, sender_id, body, attachment_ids
)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListConsultationMessages :many
SELECT * FROM consultation_messages WHERE consultation_id = $1 ORDER BY created_at;

-- name: SearchConsultationsForProvider :many
-- Text-filtered variant of ListConsultationsForProvider — same
-- requesting-or-target-provider scoping, just with a reason ILIKE filter
-- added, so search never surfaces a consultation the caller isn't already
-- a party to.
SELECT * FROM consultations
WHERE (requesting_provider_id = sqlc.arg(provider_id)::uuid OR target_provider_id = sqlc.arg(provider_id)::uuid)
  AND reason ILIKE '%' || sqlc.arg(query)::text || '%'
ORDER BY similarity(reason, sqlc.arg(query)::text) DESC, updated_at DESC
LIMIT sqlc.arg(result_limit)::int;

-- name: SearchAllConsultations :many
-- Text-filtered variant of ListAllConsultations — system_admin oversight
-- only, same as its unfiltered counterpart.
SELECT * FROM consultations
WHERE reason ILIKE '%' || sqlc.arg(query)::text || '%'
ORDER BY similarity(reason, sqlc.arg(query)::text) DESC, updated_at DESC
LIMIT sqlc.arg(result_limit)::int;

-- name: CountPendingConsultsForProvider :one
-- Consultations where provider_id is the target and awaiting their first
-- reply — used by the physician-specific dashboard view (see
-- dashboard.Summary.PendingConsultCount).
SELECT count(*) FROM consultations
WHERE target_provider_id = sqlc.arg(provider_id)::uuid AND status = 'requested';
