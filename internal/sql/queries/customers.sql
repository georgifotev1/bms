-- name: CreateCustomer :one
INSERT INTO customers (name, email, password, phone_number, brand_id) VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: DeleteCustomer :exec
DELETE FROM customers WHERE id = $1;

-- name: GetCustomerByEmail :one
SELECT * FROM customers WHERE email = $1;

-- name: GetCustomerById :one
SELECT * FROM customers WHERE id = $1;
