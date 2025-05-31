-- +goose Up
CREATE TABLE events (
    id BIGSERIAL PRIMARY KEY,
    customer_id BIGINT NOT NULL REFERENCES customers (id),
    service_id UUID NOT NULL REFERENCES services (id),
    user_id BIGINT NOT NULL REFERENCES users (id),
    brand_id INTEGER NOT NULL REFERENCES brand (id),
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    customer_name VARCHAR(50) NOT NULL,
    service_name VARCHAR(50) NOT NULL,
    user_name VARCHAR(50) NOT NULL,
    comment TEXT,
    created_at TIMESTAMP(0) NOT NULL DEFAULT NOW (),
    updated_at TIMESTAMP(0) NOT NULL DEFAULT NOW ()
);

-- Add constraint to ensure end_time is after start_time
ALTER TABLE events ADD CONSTRAINT valid_event_timespan CHECK (end_time > start_time);

-- Create indices
CREATE INDEX idx_events_brand_id ON events (brand_id);

CREATE INDEX idx_events_start_time ON events (start_time);

CREATE INDEX idx_events_end_time ON events (end_time);

CREATE INDEX idx_events_user_id ON events (user_id);

-- +goose Down
DROP INDEX idx_events_user_id;

DROP INDEX idx_events_end_time;

DROP INDEX idx_events_start_time;

DROP INDEX idx_events_brand_id;

DROP TABLE events;
