-- name: CreateUserSession :one
INSERT INTO user_sessions (user_id, expires_at)
VALUES ($1, $2)
RETURNING *;

-- name: GetUserSessionById :one
SELECT * FROM user_sessions WHERE id = $1;

-- name: GetSessionByUserId :one
SELECT * FROM user_sessions WHERE user_id = $1;

-- name: UpdateUserSession :one
UPDATE user_sessions
SET expires_at = $2
WHERE id = $1
RETURNING *;

-- name: CreateCustomerSession :one
INSERT INTO customer_sessions (customer_id, expires_at)
VALUES ($1, $2)
RETURNING *;

-- name: GetCustomerSessionById :one
SELECT * FROM customer_sessions WHERE id = $1;

-- name: GetSessionByCustomerId :one
SELECT * FROM customer_sessions WHERE customer_id = $1;

-- name: UpdateCustomerSession :one
UPDATE customer_sessions
SET expires_at = $2
WHERE id = $1
RETURNING *;

-- name: UpsertUserSession :one
INSERT INTO user_sessions (user_id, expires_at)
VALUES ($1, $2)
ON CONFLICT (user_id)
DO UPDATE SET
    expires_at = EXCLUDED.expires_at
RETURNING *;

-- name: UpsertCustomerSession :one
INSERT INTO customer_sessions (customer_id, expires_at)
VALUES ($1, $2)
ON CONFLICT (customer_id)
DO UPDATE SET
    expires_at = EXCLUDED.expires_at
RETURNING *;
