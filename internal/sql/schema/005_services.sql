-- +goose Up
CREATE TABLE services (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    title VARCHAR(100) NOT NULL,
    description VARCHAR(255),
    duration BIGINT NOT NULL,
    buffer_time BIGINT,
    cost DECIMAL(10, 2),
    is_visible BOOLEAN NOT NULL DEFAULT true,
    image_url VARCHAR(255),
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
DROP TABLE user_services;

DROP TABLE services;
