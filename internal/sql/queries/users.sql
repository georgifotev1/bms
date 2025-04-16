-- name: CreateUser :one
INSERT INTO users (name, email, password) VALUES ($1, $2, $3)
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
