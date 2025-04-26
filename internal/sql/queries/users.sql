-- name: CreateUser :one
INSERT INTO users (name, email, password, role, verified, brand_id) VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: VerifyUser :exec
UPDATE users SET
verified = TRUE,
updated_at = NOW()
WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserById :one
SELECT * FROM users WHERE id = $1;
