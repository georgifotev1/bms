
-- name: CreateBrand :one
INSERT INTO brand (name, page_url)
VALUES ($1, $2) RETURNING *;

-- name: UpdateBrand :one
UPDATE brand
SET name = $1,
    page_url = $2,
    description = $3,
    email = $4,
    phone = $5,
    country = $6,
    state = $7,
    zip_code = $8,
    city = $9,
    address = $10,
    logo_url = $11,
    banner_url = $12,
    currency = $13,
    updated_at = NOW()
WHERE id = $14
RETURNING *;

-- name: AddBrandSocialLink :one
INSERT INTO brand_social_link (
    brand_id, platform, url, display_name
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: DeleteBrandSocialLink :exec
DELETE FROM brand_social_link WHERE id = $1 AND brand_id = $2;

-- name: UpdateBrandWorkingHours :one
INSERT INTO brand_working_hours (
    brand_id, day_of_week, open_time, close_time, is_closed
) VALUES (
    $1, $2, $3, $4, $5
) ON CONFLICT (brand_id, day_of_week) DO UPDATE
SET open_time = EXCLUDED.open_time,
    close_time = EXCLUDED.close_time,
    is_closed = EXCLUDED.is_closed,
    updated_at = NOW()
RETURNING *;

-- name: GetBrandWorkingHours :many
SELECT * FROM brand_working_hours
WHERE brand_id = $1
ORDER BY day_of_week;

-- name: GetBrandProfile :one
SELECT
    b.*,
    COALESCE(
        (SELECT json_agg(sl) FROM brand_social_link sl WHERE sl.brand_id = b.id),
        '[]'
    ) AS social_links,
    COALESCE(
        (SELECT json_agg(wh) FROM brand_working_hours wh WHERE wh.brand_id = b.id),
        '[]'
    ) AS working_hours
FROM brand b
WHERE b.id = $1;

-- name: UpdateBrandPartial :one
UPDATE brand
SET
    name = COALESCE(sqlc.narg(name), name),
    page_url = COALESCE(sqlc.narg(page_url), page_url),
    description = COALESCE(sqlc.narg(description), description),
    email = COALESCE(sqlc.narg(email), email),
    phone = COALESCE(sqlc.narg(phone), phone),
    country = COALESCE(sqlc.narg(country), country),
    state = COALESCE(sqlc.narg(state), state),
    zip_code = COALESCE(sqlc.narg(zip_code), zip_code),
    city = COALESCE(sqlc.narg(city), city),
    address = COALESCE(sqlc.narg(address), address),
    logo_url = COALESCE(sqlc.narg(logo_url), logo_url),
    banner_url = COALESCE(sqlc.narg(banner_url), banner_url),
    currency = COALESCE(sqlc.narg(currency), currency),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: GetBrandUsers :many
SELECT * FROM users WHERE brand_id = $1;

-- name: AssociateUserWithBrand :exec
UPDATE users SET brand_id = $1 WHERE id = $2 RETURNING *;

-- name: GetBrandByUrl :one
SELECT page_url FROM brand WHERE page_url = $1;
