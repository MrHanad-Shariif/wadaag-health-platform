-- name: CreateUser :one
INSERT INTO users (email, phone, password_hash, role)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: FindUserByEmailOrPhone :one
SELECT *
FROM users
WHERE email = $1 OR phone = $1;

-- name: FindUserByID :one
SELECT *
FROM users
WHERE id = $1;
