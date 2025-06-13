-- +goose Up
CREATE TABLE user_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    user_id BIGINT NOT NULL UNIQUE REFERENCES users (id) ON DELETE CASCADE,
    expires_at TIMESTAMP NOT NULL
);

CREATE TABLE customer_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
    customer_id BIGINT NOT NULL UNIQUE REFERENCES customers (id) ON DELETE CASCADE,
    expires_at TIMESTAMP NOT NULL
);

-- +goose Down
DROP TABLE customer_sessions;

DROP TABLE user_sessions;
