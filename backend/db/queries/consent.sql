-- name: CreateConsentGrant :one
INSERT INTO consent_grants (patient_id, grantee_type, grantee_id, scope, scope_ref, granted_via, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: FindActiveGrants :many
SELECT * FROM consent_grants
WHERE patient_id = $1
  AND status = 'active'
  AND (expires_at IS NULL OR expires_at > now())
  AND (
    (grantee_type = 'provider' AND grantee_id = $2) OR
    (grantee_type = 'facility' AND grantee_id = $3)
  );

-- name: ListConsentGrantsForPatient :many
SELECT * FROM consent_grants WHERE patient_id = $1 ORDER BY granted_at DESC;

-- name: RevokeConsentGrant :one
UPDATE consent_grants
SET status = 'revoked', revoked_at = now()
WHERE id = $1 AND patient_id = $2 AND status = 'active'
RETURNING *;
