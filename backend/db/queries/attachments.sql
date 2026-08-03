-- name: CreateAttachment :one
INSERT INTO attachments (patient_id, encounter_id, uploaded_by, file_key, filename, mime_type, size, version, root_id)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: FindAttachmentByID :one
SELECT * FROM attachments WHERE id = $1;

-- name: ListLatestAttachmentsByPatient :many
-- One row per logical document — the highest-version row within each
-- root_id/id lineage (COALESCE(root_id, id) groups a root row together with
-- every version that points back at it).
SELECT DISTINCT ON (COALESCE(root_id, id)) *
FROM attachments
WHERE patient_id = $1
ORDER BY COALESCE(root_id, id), version DESC;

-- name: ListAttachmentVersionsByRoot :many
-- rootID must already be resolved to the true root (the id shared by
-- COALESCE(root_id, id) across the whole lineage) — see
-- Service.resolveRootID.
SELECT * FROM attachments
WHERE COALESCE(root_id, id) = $1
ORDER BY version;

-- name: FindMaxAttachmentVersionByRoot :one
SELECT COALESCE(MAX(version), 0)::int AS max_version
FROM attachments
WHERE COALESCE(root_id, id) = $1;
