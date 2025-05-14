-- +goose Up
CREATE TABLE booking_status (
    status_id SERIAL PRIMARY KEY,
    status_name VARCHAR(50) UNIQUE NOT NULL
);

INSERT INTO
    booking_status (status_name)
VALUES
    ('pending'),
    ('confirmed'),
    ('completed'),
    ('cancelled'),
    ('no_show');

CREATE TABLE bookings (
    id BIGSERIAL PRIMARY KEY,
    customer_id INTEGER NOT NULL REFERENCES customers (id),
    service_id UUID NOT NULL REFERENCES services (id),
    user_id INTEGER REFERENCES users (id),
    brand_id INTEGER NOT NULL REFERENCES brand (id),
    date TIMESTAMP NOT NULL,
    status_id INTEGER NOT NULL REFERENCES booking_status (status_id),
    comment TEXT,
    created_at TIMESTAMP(0) NOT NULL DEFAULT NOW (),
    updated_at TIMESTAMP(0) NOT NULL DEFAULT NOW ()
);

CREATE INDEX idx_bookings_brand_id ON bookings (brand_id);

CREATE INDEX idx_bookings_datetime ON bookings (date);

CREATE INDEX idx_bookings_status ON bookings (status_id);

-- +goose Down
DROP INDEX idx_bookings_status;

DROP INDEX idx_bookings_datetime;

DROP INDEX idx_bookings_brand_id;

DROP TABLE bookings;

DROP TABLE booking_status;
