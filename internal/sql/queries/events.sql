-- name: GetEventByID :one
SELECT * FROM events b WHERE id = $1;

-- name: ListEventsByBrand :many
SELECT * FROM events
WHERE brand_id = $1
ORDER BY start_time
LIMIT $2
OFFSET $3;

-- name: ListEventsByCustomer :many
SELECT * FROM events
WHERE customer_id = $1
ORDER BY start_time
LIMIT $2
OFFSET $3;

-- name: ListEventsByUser :many
SELECT * FROM events
WHERE user_id = $1
ORDER BY start_time
LIMIT $2
OFFSET $3;

-- name: CreateEvent :one
INSERT INTO events (
  customer_id,
  service_id,
  user_id,
  brand_id,
  start_time,
  end_time,
  comment,
  customer_name,
  service_name,
  user_name,
  cost,
  buffer_time,
  created_at,
  updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW(), NOW()
) RETURNING *;

-- name: UpdateEvent :one
UPDATE events
SET
  customer_id = $2,
  service_id = $3,
  user_id = $4,
  brand_id = $5,
  start_time = $6,
  end_time = $7,
  comment = $8,
  customer_name = $9,
  service_name = $10,
  user_name = $11,
  cost = $12,
  buffer_time = $13,
  updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteEvent :exec
DELETE FROM events
WHERE id = $1;

-- name: GetEventsByWeek :many
SELECT *
FROM events
WHERE DATE(start_time) BETWEEN sqlc.arg(start_date) AND sqlc.arg(end_date)
AND brand_id = sqlc.arg(brand_id)
ORDER BY start_time ASC;

-- name: GetEventsByDay :many
SELECT *
FROM events
WHERE DATE(start_time) = $1
AND brand_id = $2
ORDER BY start_time ASC;

-- name: GetUserEventsByWeek :many
SELECT *
FROM events
WHERE DATE(start_time) BETWEEN sqlc.arg(start_date) AND sqlc.arg(end_date)
AND brand_id = sqlc.arg(brand_id)
AND user_id = sqlc.arg(user_id)
ORDER BY start_time ASC;

-- name: GetUserEventsByDay :many
SELECT *
FROM events
WHERE DATE(start_time) = $1
AND brand_id = $2
AND user_id = $3
ORDER BY start_time ASC;

-- name: CheckSpecificTimeslotAvailability :one
WITH service_info AS (
    SELECT s.duration, s.buffer_time
    FROM services s
    WHERE s.id = sqlc.arg(service_id)
),
user_can_provide AS (
    SELECT 1
    FROM user_services us
    WHERE us.user_id = sqlc.arg(user_id)
      AND us.service_id = sqlc.arg(service_id)
)
SELECT
    COALESCE(
        EXISTS (SELECT 1 FROM user_can_provide)
        AND NOT EXISTS (
            SELECT 1
            FROM events b
            WHERE b.user_id = sqlc.arg(user_id)
              AND (
                  (b.start_time < sqlc.arg(end_time) AND b.end_time > sqlc.arg(start_time))
                  OR (b.start_time < (sqlc.arg(end_time) + (INTERVAL '1 minute' * si.buffer_time))
                      AND b.end_time > sqlc.arg(start_time))
              )
        ),
        (sqlc.arg(user_id) IS NULL)
    ) AS is_available
FROM service_info si;
