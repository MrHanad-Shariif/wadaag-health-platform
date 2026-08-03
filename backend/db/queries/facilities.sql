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

-- name: FindProviderByID :one
SELECT * FROM providers WHERE id = $1;

-- name: UpdateProviderProfile :one
-- Self-editable provider profile fields. Deliberately excludes
-- verification_status (admin-only, see UpdateProviderVerificationStatus)
-- and specialty/license_number (set at creation, not editable via this
-- profile-update surface).
UPDATE providers
SET qualifications = $2,
    years_experience = $3,
    consultation_fee = $4,
    languages = $5,
    certificates = $6,
    areas_of_expertise = $7,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateProviderVerificationStatus :one
-- Admin-only field, kept as its own query so the self-edit path
-- (UpdateProviderProfile) can never touch it.
UPDATE providers
SET verification_status = $2, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: CountFacilities :one
SELECT count(*) FROM facilities;

-- name: UpdateFacilityLogoAndHours :one
UPDATE facilities
SET logo_url = $2, working_hours = $3, referral_policies = $4, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: CreateBranch :one
INSERT INTO branches (facility_id, name, address, phone)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListBranchesByFacility :many
SELECT * FROM branches WHERE facility_id = $1 ORDER BY name;

-- name: FindBranchByID :one
SELECT * FROM branches WHERE id = $1;

-- name: UpdateBranch :one
UPDATE branches
SET name = $2, address = $3, phone = $4, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteBranch :exec
DELETE FROM branches WHERE id = $1;

-- name: CreateDepartment :one
INSERT INTO departments (facility_id, name)
VALUES ($1, $2)
RETURNING *;

-- name: ListDepartmentsByFacility :many
SELECT * FROM departments WHERE facility_id = $1 ORDER BY name;

-- name: FindDepartmentByID :one
SELECT * FROM departments WHERE id = $1;

-- name: UpdateDepartment :one
UPDATE departments
SET name = $2, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteDepartment :exec
DELETE FROM departments WHERE id = $1;

-- name: SearchFacilities :many
-- Directory-style search, open to any authenticated user (no sensitive
-- data on a facility's name/region/district) — see search.Service.Search.
SELECT * FROM facilities
WHERE name ILIKE '%' || sqlc.arg(query)::text || '%'
ORDER BY similarity(name, sqlc.arg(query)::text) DESC, name
LIMIT sqlc.arg(result_limit)::int;

-- name: SearchProviders :many
-- Directory-style doctor search by the user's full_name or the provider's
-- specialty — open to any authenticated user, same reasoning as
-- SearchFacilities. Joins users only to read full_name; no other
-- user fields are exposed.
SELECT p.*, u.full_name AS user_full_name FROM providers p
JOIN users u ON u.id = p.user_id
WHERE u.full_name ILIKE '%' || sqlc.arg(query)::text || '%'
   OR p.specialty ILIKE '%' || sqlc.arg(query)::text || '%'
ORDER BY similarity(coalesce(u.full_name, ''), sqlc.arg(query)::text) DESC, u.full_name
LIMIT sqlc.arg(result_limit)::int;
