-- name: InsertAuditLog :one
INSERT INTO audit_log (actor_user_id, actor_role, action, resource_type, resource_id, patient_id, result, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: ListAuditLogForPatient :many
SELECT * FROM audit_log WHERE patient_id = $1 ORDER BY occurred_at DESC;

-- name: ListAuditEntries :many
-- General, filtered listing for the system_admin-only audit browser (see
-- audit.Handler's "GET /" route) — every filter is optional; a NULL/absent
-- arg matches every row for that column. result_limit is always supplied
-- by the caller (audit.Service caps it before it ever reaches here, see
-- Service.ListEntries).
SELECT * FROM audit_log
WHERE (sqlc.narg(actor_user_id)::uuid IS NULL OR actor_user_id = sqlc.narg(actor_user_id))
  AND (sqlc.narg(resource_type)::text IS NULL OR resource_type = sqlc.narg(resource_type))
  AND (sqlc.narg(result)::text IS NULL OR result = sqlc.narg(result)::audit_result)
ORDER BY occurred_at DESC
LIMIT sqlc.arg(result_limit)::int;
