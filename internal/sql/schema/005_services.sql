-- +goose Up
CREATE TABLE services (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    title VARCHAR(100) NOT NULL,
    description VARCHAR(255) NOT NULL,
    duration INTERVAL NOT NULL,
    buffer_time INTERVAL,
    cost DECIMAL(10, 2),
    is_visible BOOLEAN DEFAULT true,
    logo_url VARCHAR(255),
    brand_id INTEGER NOT NULL REFERENCES brand (id) ON DELETE CASCADE,
    created_at TIMESTAMP(0) NOT NULL DEFAULT NOW (),
    updated_at TIMESTAMP(0) NOT NULL DEFAULT NOW ()
);

CREATE TABLE user_services (
    user_id BIGINT REFERENCES users (id) ON DELETE CASCADE,
    service_id UUID REFERENCES services (id) ON DELETE CASCADE,
    created_at TIMESTAMP(0) NOT NULL DEFAULT NOW (),
    updated_at TIMESTAMP(0) NOT NULL DEFAULT NOW (),
    PRIMARY KEY (user_id, service_id)
);

-- +goose Down
DROP TABLE services;
