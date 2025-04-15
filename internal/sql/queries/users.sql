-- name: CreateUser :one
INSERT INTO users (name, email, password) VALUES ($1, $2, $3)
RETURNING *;

-- name: VerifyUser :one
UPDATE users SET
verified = $1,
updated_at = NOW()
WHERE id = $2
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;
