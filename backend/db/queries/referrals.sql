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

-- name: ListAllReferrals :many
-- Platform-wide oversight listing — system_admin has no facility affiliation
-- of its own to scope ListReferralsForFacility by, so its inbox view needs
-- an unscoped list instead.
SELECT * FROM referrals ORDER BY created_at DESC;

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

-- name: CountReferralsByStatus :many
SELECT status, count(*) AS total FROM referrals GROUP BY status;

-- name: SearchReferralsForFacility :many
-- Text-filtered variant of ListReferralsForFacility — same
-- receiving_facility_id scoping, just with a reason ILIKE filter added, so
-- search never sees referrals outside a facility's existing inbox.
SELECT * FROM referrals
WHERE receiving_facility_id = sqlc.arg(facility_id)::uuid
  AND reason ILIKE '%' || sqlc.arg(query)::text || '%'
ORDER BY similarity(reason, sqlc.arg(query)::text) DESC, created_at DESC
LIMIT sqlc.arg(result_limit)::int;

-- name: SearchAllReferrals :many
-- Text-filtered variant of ListAllReferrals — system_admin oversight only,
-- same as its unfiltered counterpart.
SELECT * FROM referrals
WHERE reason ILIKE '%' || sqlc.arg(query)::text || '%'
ORDER BY similarity(reason, sqlc.arg(query)::text) DESC, created_at DESC
LIMIT sqlc.arg(result_limit)::int;

-- name: CountReferralsByReceivingProvider :many
-- Powers the admin dashboard's "most active doctors" panel — ranks
-- providers by how many referrals have been routed to them as the
-- receiving_provider_id (only ever set once a referral has been accepted,
-- see AcceptReferral), across every status, most-referred first.
SELECT p.id AS provider_id, u.full_name, count(r.id) AS referral_count
FROM referrals r
JOIN providers p ON p.id = r.receiving_provider_id
JOIN users u ON u.id = p.user_id
GROUP BY p.id, u.full_name
ORDER BY referral_count DESC, p.id
LIMIT sqlc.arg(result_limit)::int;

-- name: CountReferralsPendingForProvider :one
-- "Actionable" referrals currently assigned to a provider as the receiving
-- provider: accepted (about to start) or in_progress (already started but
-- not yet completed). Used by the physician-specific dashboard view (see
-- dashboard.Summary.MyReferralsPendingCount).
SELECT count(*) FROM referrals
WHERE receiving_provider_id = sqlc.arg(provider_id)::uuid
  AND status IN ('accepted', 'in_progress');
