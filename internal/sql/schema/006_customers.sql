-- +goose Up
CREATE TABLE customers (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    password BYTEA NOT NULL,
    phone_number VARCHAR(20),
    brand_id INTEGER NOT NULL REFERENCES brand (id),
    created_at TIMESTAMP(0) NOT NULL DEFAULT NOW (),
    updated_at TIMESTAMP(0) NOT NULL DEFAULT NOW ()
);

CREATE INDEX idx_customers_email ON customers (email);

-- +goose Down
DROP INDEX idx_customers_email;

DROP TABLE customers;
