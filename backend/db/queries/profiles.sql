-- name: GetUserProfile :one
SELECT *
FROM user_profiles
WHERE user_id = $1;

-- name: UpsertUserProfile :one
-- Partial update semantics live in the repository (Go), not here: the
-- repository always re-reads-then-writes the merged bio/languages/
-- availability_status, so this query itself is a plain full upsert.
INSERT INTO user_profiles (user_id, bio, languages, availability_status)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id) DO UPDATE
SET bio = EXCLUDED.bio,
    languages = EXCLUDED.languages,
    availability_status = EXCLUDED.availability_status,
    updated_at = now()
RETURNING *;

-- name: UpsertUserProfilePhoto :one
INSERT INTO user_profiles (user_id, photo_url)
VALUES ($1, $2)
ON CONFLICT (user_id) DO UPDATE
SET photo_url = EXCLUDED.photo_url,
    updated_at = now()
RETURNING *;

-- name: SetUserProfilePresence :exec
INSERT INTO user_profiles (user_id, is_online, last_seen_at)
VALUES ($1, $2, $3)
ON CONFLICT (user_id) DO UPDATE
SET is_online = EXCLUDED.is_online,
    last_seen_at = EXCLUDED.last_seen_at,
    updated_at = now();
