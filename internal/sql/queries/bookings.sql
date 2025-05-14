-- name: GetBookingsByBrand :many
SELECT
    b.id,
    b.customer_id,
    b.service_id,
    b.user_id,
    b.brand_id,
    b.date,
    bs.status_name,
    b.comment,
    b.created_at,
    b.updated_at
FROM
    bookings b
JOIN
    booking_status bs ON b.status_id = bs.status_id
WHERE
    b.brand_id = $1
ORDER BY
    b.date DESC
LIMIT $2
OFFSET $3;

-- name: GetBookingByID :one
SELECT
    b.id,
    b.customer_id,
    b.service_id,
    b.user_id,
    b.brand_id,
    b.date,
    bs.status_name,
    b.comment,
    b.created_at,
    b.updated_at
FROM
    bookings b
JOIN
    booking_status bs ON b.status_id = bs.status_id
WHERE
    b.id = $1
    AND b.brand_id = $2;

-- name: CreateBooking :one
INSERT INTO bookings (
    customer_id,
    service_id,
    user_id,
    brand_id,
    date,
    status_id,
    comment
) VALUES (
    $1, $2, $3, $4, $5,
    (SELECT status_id FROM booking_status WHERE status_name = $6),
    $7
)
RETURNING *;

-- name: UpdateBookingStatus :one
UPDATE bookings
SET
    status_id = (SELECT status_id FROM booking_status WHERE status_name = $2),
    updated_at = NOW()
WHERE
    id = $1
    AND brand_id = $3
RETURNING *;

-- name: UpdateBookingDetails :one
UPDATE bookings
SET
    date = COALESCE($2, date),
    user_id = COALESCE($3, user_id),
    comment = COALESCE($4, comment),
    updated_at = NOW()
WHERE
    id = $1
    AND brand_id = $5
RETURNING *;

-- name: DeleteBooking :exec
DELETE FROM bookings
WHERE
    id = $1
    AND brand_id = $2;

-- name: GetBookingsByDateRange :many
SELECT
    b.id,
    b.customer_id,
    b.service_id,
    b.user_id,
    b.brand_id,
    b.date,
    bs.status_name,
    b.comment,
    b.created_at,
    b.updated_at
FROM
    bookings b
JOIN
    booking_status bs ON b.status_id = bs.status_id
WHERE
    b.brand_id = $1
    AND b.date BETWEEN $2 AND $3
ORDER BY
    b.date ASC;

-- name: GetCustomerBookings :many
SELECT
    b.id,
    b.customer_id,
    b.service_id,
    b.user_id,
    b.brand_id,
    b.date,
    bs.status_name,
    b.comment,
    b.created_at,
    b.updated_at
FROM
    bookings b
JOIN
    booking_status bs ON b.status_id = bs.status_id
WHERE
    b.customer_id = $1
    AND b.brand_id = $2
ORDER BY
    b.date DESC
LIMIT $3
OFFSET $4;

-- name: GetUserBookingsByDateRange :many
SELECT
    b.id,
    b.customer_id,
    b.service_id,
    b.user_id,
    b.brand_id,
    b.date,
    bs.status_name,
    b.comment,
    b.created_at,
    b.updated_at
FROM
    bookings b
JOIN
    booking_status bs ON b.status_id = bs.status_id
WHERE
    b.brand_id = $1
    AND b.user_id = $2
    AND b.date BETWEEN $3 AND $4
ORDER BY
    b.date ASC;
