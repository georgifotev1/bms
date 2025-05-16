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

-- Modified bookings table with start and end time
CREATE TABLE bookings (
    id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL REFERENCES customers (id),
    service_id UUID NOT NULL REFERENCES services (id),
    user_id BIGINT NOT NULL REFERENCES users (id),
    brand_id INTEGER NOT NULL REFERENCES brand (id),
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    status_id INTEGER NOT NULL REFERENCES booking_status (status_id),
    comment TEXT,
    created_at TIMESTAMP(0) NOT NULL DEFAULT NOW (),
    updated_at TIMESTAMP(0) NOT NULL DEFAULT NOW ()
);

-- Add constraint to ensure end_time is after start_time
ALTER TABLE bookings ADD CONSTRAINT valid_booking_timespan CHECK (end_time > start_time);

-- Create indices
CREATE INDEX idx_bookings_brand_id ON bookings (brand_id);

CREATE INDEX idx_bookings_start_time ON bookings (start_time);

CREATE INDEX idx_bookings_end_time ON bookings (end_time);

CREATE INDEX idx_bookings_status ON bookings (status_id);

CREATE INDEX idx_bookings_user_id ON bookings (user_id);

-- +goose Down
DROP INDEX idx_bookings_user_id;

DROP INDEX idx_bookings_status;

DROP INDEX idx_bookings_end_time;

DROP INDEX idx_bookings_start_time;

DROP INDEX idx_bookings_brand_id;

DROP TABLE bookings;

DROP TABLE booking_status;

ALTER TABLE services
DROP COLUMN IF EXISTS duration_minutes;

ALTER TABLE services
DROP COLUMN IF EXISTS buffer_minutes;
