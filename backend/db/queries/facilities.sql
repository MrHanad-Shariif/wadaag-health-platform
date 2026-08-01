-- name: CreateFacility :one
INSERT INTO facilities (name, type, region, district, phone, address)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListFacilities :many
SELECT * FROM facilities ORDER BY name;

-- name: FindFacilityByID :one
SELECT * FROM facilities WHERE id = $1;

-- name: CreateProvider :one
INSERT INTO providers (user_id, facility_id, specialty, license_number)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: FindProviderByUserID :one
SELECT * FROM providers WHERE user_id = $1;

-- name: CountFacilities :one
SELECT count(*) FROM facilities;
