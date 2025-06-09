-- name: CreateService :one
INSERT INTO services (
    title,
    description,
    duration,
    buffer_time,
    cost,
    is_visible,
    image_url,
    brand_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetService :one
SELECT * FROM services
WHERE id = $1;

-- name: ListServicesWithProviders :many
SELECT
    services.id,
    services.title,
    services.description,
    services.duration,
    services.buffer_time,
    services.cost,
    services.is_visible,
    services.image_url,
    services.brand_id,
    services.created_at,
    services.updated_at,
    users.id as provider_id
FROM services
LEFT JOIN user_services us ON services.id = us.service_id
LEFT JOIN users ON us.user_id = users.id
WHERE services.brand_id = $1
ORDER BY services.title, users.name;

-- name: ListVisibleServices :many
SELECT * FROM services
WHERE brand_id = $1 AND is_visible = true
ORDER BY created_at DESC;

-- name: UpdateService :one
UPDATE services
SET
    title = $2,
    description = $3,
    duration = $4,
    buffer_time = $5,
    cost = $6,
    is_visible = $7,
    image_url = $8,
    brand_id = $9,
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteService :exec
DELETE FROM services
WHERE id = $1;

-- name: AssignServiceToUser :exec
INSERT INTO user_services (
    user_id,
    service_id
) VALUES (
    $1, $2
);

-- name: RemoveUsersFromService :exec
DELETE FROM user_services
WHERE service_id = $1;

-- name: ListUserServices :many
SELECT s.*
FROM services s
JOIN user_services us ON s.id = us.service_id
WHERE us.user_id = $1;
