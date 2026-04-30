-- +goose Up
ALTER TABLE users
ADD COLUMN otp TEXT;

ALTER TABLE users
ADD COLUMN otp_expires_at TIMESTAMP;

ALTER TABLE users
ADD COLUMN verified_at TIMESTAMP;

-- +goose Down
ALTER TABLE users
DROP COLUMN otp;

ALTER TABLE users
DROP COLUMN otp_expires_at;

ALTER TABLE users
DROP COLUMN verified_at;
