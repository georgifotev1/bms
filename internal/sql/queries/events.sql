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

-- name: GetAvailableTimeslots :many
WITH
-- Parameters: brand_id, date, service_id
service_info AS (
    SELECT s.id, s.duration, s.buffer_time
    FROM services s
    WHERE s.id = sqlc.arg(service_id)
),
-- Get all events for the given date and brand
daily_events AS (
    SELECT b.start_time, b.end_time, b.user_id
    FROM events b
    WHERE b.brand_id = sqlc.arg(brand_id)
      AND DATE(b.start_time) = sqlc.arg(date)
),
-- Get all users (staff) for the brand
staff AS (
    SELECT u.id
    FROM users u
    WHERE u.brand_id = sqlc.arg(brand_id)
    AND u.verified = true
),
-- Generate time slots for the day (e.g., every 15 min from 9am to 5pm)
time_slots AS (
    SELECT
        generate_series(
            sqlc.arg(date)::date + sqlc.arg(start_time)::time,
            sqlc.arg(date)::date + sqlc.arg(end_time)::time,
            (INTERVAL '1 minute' * (SELECT duration FROM service_info))
        ) AS slot_start
),
-- Apply service duration to get slot end times
service_slots AS (
    SELECT
        ts.slot_start,
        ts.slot_start + (INTERVAL '1 minute' * (SELECT duration FROM service_info)) AS slot_end,
        (SELECT buffer_time FROM service_info) AS buffer_time
    FROM time_slots ts
),
-- Check availability for each staff member and time slot
staff_availability AS (
    SELECT
        s.id AS user_id,
        ss.slot_start,
        ss.slot_end,
        CASE WHEN EXISTS (
            SELECT 1 FROM daily_events db
            WHERE db.user_id = s.id
              AND (
                  -- Overlapping condition
                  (db.start_time < ss.slot_end AND db.end_time > ss.slot_start)
                  -- Also consider buffer time after appointment
                  OR (db.start_time < (ss.slot_end + (INTERVAL '1 minute' * ss.buffer_time))
                      AND db.end_time > ss.slot_start)
              )
        ) THEN false ELSE true END AS is_available
    FROM staff s
    CROSS JOIN service_slots ss
)
-- Final available time slots with at least one available staff
SELECT
    slot_start,
    slot_end,
    ARRAY_AGG(user_id) AS available_staff_ids,
    COUNT(user_id) AS available_staff_count
FROM staff_availability
WHERE is_available = true
GROUP BY slot_start, slot_end
HAVING COUNT(user_id) > 0
ORDER BY slot_start;

-- name: CheckSpecificTimeslotAvailability :one
WITH service_info AS (
    SELECT s.duration, s.buffer_time
    FROM services s
    WHERE s.id = sqlc.arg(service_id)
)
SELECT
    COALESCE(
        NOT EXISTS (
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
