-- +goose Up
CREATE TABLE user_invitations (
    token TEXT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    expiry TIMESTAMP NOT NULL
);

-- +goose Down
DROP TABLE user_invitations;
