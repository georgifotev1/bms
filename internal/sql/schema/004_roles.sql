-- +goose Up
CREATE TABLE IF NOT EXISTS roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    level int NOT NULL DEFAULT 0
);

INSERT INTO
    roles (name, level)
VALUES
    ('user', 1);

INSERT INTO
    roles (name, level)
VALUES
    ('admin', 2);

INSERT INTO
    roles (name, level)
VALUES
    ('owner', 3);

ALTER TABLE IF EXISTS users
ADD COLUMN role VARCHAR(255) REFERENCES roles (name) DEFAULT 'user';

UPDATE users
SET
    role = (
        SELECT
            name
        FROM
            roles
        WHERE
            level = 1
    );

ALTER TABLE users
ALTER COLUMN role
DROP DEFAULT;

ALTER TABLE users
ALTER COLUMN role
SET
    NOT NULL;

ALTER TABLE users ADD CONSTRAINT valid_role CHECK (role IN ('admin', 'user', 'owner'));

-- +goose Down
ALTER TABLE IF EXISTS users
DROP COLUMN role;

DROP TABLE roles;
