-- name: CreateSavedSearch :one
INSERT INTO saved_searches (user_id, name, query, filters)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListSavedSearchesForUser :many
SELECT * FROM saved_searches WHERE user_id = $1 ORDER BY created_at DESC;

-- name: FindSavedSearchByID :one
SELECT * FROM saved_searches WHERE id = $1;

-- name: DeleteSavedSearch :execrows
DELETE FROM saved_searches WHERE id = $1 AND user_id = $2;
