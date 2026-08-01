-- name: CreateRole :one
INSERT INTO roles (name, description, full_access)
VALUES ($1, $2, $3)
RETURNING *;

-- name: FindRoleByID :one
SELECT * FROM roles WHERE id = $1;

-- name: ListRoles :many
SELECT * FROM roles ORDER BY name;

-- name: UpdateRole :one
UPDATE roles
SET name = $2, description = $3, full_access = $4, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteRole :exec
DELETE FROM roles WHERE id = $1;

-- name: ListPermissions :many
SELECT * FROM permissions ORDER BY resource, action;

-- name: ReplaceRolePermissions :exec
DELETE FROM role_permissions WHERE role_id = $1;

-- name: InsertRolePermissions :exec
INSERT INTO role_permissions (role_id, permission_id)
SELECT $1, unnest($2::uuid[]);

-- name: ListPermissionsForRole :many
SELECT p.id, p.resource, p.action
FROM role_permissions rp
JOIN permissions p ON p.id = rp.permission_id
WHERE rp.role_id = $1
ORDER BY p.resource, p.action;

-- name: GetUserAccess :one
SELECT r.id AS role_id, r.full_access
FROM users u
JOIN roles r ON u.role_id = r.id
WHERE u.id = $1;

-- name: ListUsersWithRole :many
SELECT u.id, u.email, u.phone, u.role, u.status, u.created_at, u.role_id, r.name AS role_name
FROM users u
LEFT JOIN roles r ON u.role_id = r.id
ORDER BY u.created_at DESC;

-- name: FindUserWithRoleByID :one
SELECT u.id, u.email, u.phone, u.role, u.status, u.created_at, u.role_id, r.name AS role_name
FROM users u
LEFT JOIN roles r ON u.role_id = r.id
WHERE u.id = $1;

-- name: UpdateUserRoleID :one
UPDATE users SET role_id = $2, updated_at = now() WHERE id = $1
RETURNING *;

-- name: UpdateUserStatus :one
UPDATE users SET status = $2, updated_at = now() WHERE id = $1
RETURNING *;

-- name: CountUsers :one
SELECT count(*) FROM users;
