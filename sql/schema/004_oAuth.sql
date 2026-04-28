-- +goose Up
ALTER TABLE users
ADD COLUMN provider VARCHAR(50) NOT NULL DEFAULT 'local';

ALTER TABLE users
ADD COLUMN provider_id TEXT NOT NULL DEFAULT '';

ALTER TABLE users
ALTER COLUMN hashed_password
SET DEFAULT '';

-- +goose Down
ALTER TABLE users
DROP COLUMN provider;

ALTER TABLE users
DROP COLUMN provider_id;

ALTER TABLE users
ALTER COLUMN hashed_password
DROP DEFAULT;
