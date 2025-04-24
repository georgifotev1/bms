-- +goose Up
CREATE TABLE brand (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    page_url VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    email VARCHAR(255),
    phone VARCHAR(20),
    country VARCHAR(100),
    state VARCHAR(100),
    zip_code VARCHAR(20),
    city VARCHAR(100),
    address TEXT,
    logo_url VARCHAR(255),
    banner_url VARCHAR(255),
    currency VARCHAR(3),
    created_at TIMESTAMP(0) NOT NULL DEFAULT NOW (),
    updated_at TIMESTAMP(0) NOT NULL DEFAULT NOW ()
);

CREATE TABLE brand_social_link (
    id SERIAL PRIMARY KEY,
    brand_id INTEGER NOT NULL REFERENCES brand (id) ON DELETE CASCADE,
    platform VARCHAR(50) NOT NULL,
    url VARCHAR(255) NOT NULL,
    display_name VARCHAR(100),
    created_at TIMESTAMP(0) NOT NULL DEFAULT NOW (),
    updated_at TIMESTAMP(0) NOT NULL DEFAULT NOW (),
    UNIQUE (brand_id, platform)
);

CREATE TABLE brand_working_hours (
    id SERIAL PRIMARY KEY,
    brand_id INTEGER NOT NULL REFERENCES brand (id) ON DELETE CASCADE,
    day_of_week INTEGER NOT NULL, -- 0-6 for Sunday-Saturday
    open_time TIME,
    close_time TIME,
    is_closed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP(0) NOT NULL DEFAULT NOW (),
    updated_at TIMESTAMP(0) NOT NULL DEFAULT NOW (),
    UNIQUE (brand_id, day_of_week)
);

ALTER TABLE users
ADD COLUMN brand_id INTEGER REFERENCES brand (id);

-- +goose Down
ALTER TABLE users
DROP COLUMN brand_id;

DROP TABLE brand_working_hours;

DROP TABLE brand_social_link;

DROP TABLE brand;
