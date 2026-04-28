-- +goose Up
CREATE TABLE request_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    method VARCHAR(10) NOT NULL,
    path TEXT NOT NULL,
    status INT NOT NULL,
    latency_ms BIGINT NOT NULL,
    request_id TEXT NOT NULL,
    user_id TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE request_logs;
