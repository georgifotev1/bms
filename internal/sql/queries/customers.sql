-- name: CreateCustomer :one
INSERT INTO customers (name, email, password, phone_number, brand_id) VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: CreateGuestCustomer :one
INSERT INTO customers (name, email, phone_number, brand_id) VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: DeleteCustomer :exec
DELETE FROM customers WHERE id = $1;

-- name: GetCustomerByEmail :one
SELECT * FROM customers WHERE email = $1;

-- name: GetCustomerById :one
SELECT * FROM customers WHERE id = $1;

-- name: GetCustomerByNameAndPhone :one
SELECT * FROM customers WHERE name = $1 AND phone_number = $2;

-- name: GetCustomersByBrand :many
SELECT * FROM customers WHERE brand_id = $1;
