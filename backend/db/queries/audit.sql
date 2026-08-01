-- name: InsertAuditLog :one
INSERT INTO audit_log (actor_user_id, actor_role, action, resource_type, resource_id, patient_id, result, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: ListAuditLogForPatient :many
SELECT * FROM audit_log WHERE patient_id = $1 ORDER BY occurred_at DESC;
