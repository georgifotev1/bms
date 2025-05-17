-- name: GetBookingByID :one
SELECT * FROM bookings b WHERE id = $1;

-- name: ListBookingsByBrand :many
SELECT * FROM bookings
WHERE brand_id = $1
ORDER BY start_time
LIMIT $2
OFFSET $3;

-- name: ListBookingsByCustomer :many
SELECT * FROM bookings
WHERE customer_id = $1
ORDER BY start_time
LIMIT $2
OFFSET $3;

-- name: ListBookingsByUser :many
SELECT * FROM bookings
WHERE user_id = $1
ORDER BY start_time
LIMIT $2
OFFSET $3;

-- name: CreateBooking :one
INSERT INTO bookings (
  customer_id,
  service_id,
  user_id,
  brand_id,
  start_time,
  end_time,
  comment,
  created_at,
  updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, NOW(), NOW()
) RETURNING *;

-- name: UpdateBookingDetails :one
UPDATE bookings
SET service_id = $2,
    user_id = $3,
    start_time = $4,
    end_time = $5,
    comment = $6,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteBooking :exec
DELETE FROM bookings
WHERE id = $1;

-- name: GetBookingsByTimeRange :many
SELECT * FROM bookings
WHERE brand_id = $1
  AND start_time >= $2
  AND end_time <= $3
ORDER BY start_time;

-- name: GetActiveBookingsForUser :many
SELECT *
FROM bookings
WHERE user_id = $1
  AND start_time <= $2
  AND end_time >= $2;

-- name: GetAvailableTimeslots :many
WITH
-- Parameters: brand_id, date, service_id
service_info AS (
    SELECT s.id, s.duration, s.buffer_time
    FROM services s
    WHERE s.id = sqlc.arg(service_id)
),
-- Get all bookings for the given date and brand
daily_bookings AS (
    SELECT b.start_time, b.end_time, b.user_id
    FROM bookings b
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
            sqlc.arg(date)::date + '09:00:00'::time,
            sqlc.arg(date)::date + '17:00:00'::time,
            '15 minutes'::interval
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
            SELECT 1 FROM daily_bookings db
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
            FROM bookings b
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
