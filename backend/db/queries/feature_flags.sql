-- name: ListFeatureFlags :many
SELECT * FROM feature_flags ORDER BY key;

-- name: UpsertFeatureFlag :one
INSERT INTO feature_flags (key, enabled, description)
VALUES ($1, $2, $3)
ON CONFLICT (key) DO UPDATE
SET enabled = excluded.enabled,
    description = COALESCE(excluded.description, feature_flags.description),
    updated_at = now()
RETURNING *;
