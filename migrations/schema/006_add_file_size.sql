-- +goose Up
ALTER TABLE files
ADD COLUMN file_size BIGINT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE files
DROP COLUMN file_size;
