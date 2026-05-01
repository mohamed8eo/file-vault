-- +goose Up
CREATE TABLE share_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    token TEXT NOT NULL UNIQUE,
    created_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ,
    password_hash TEXT,
    max_downloads INT,
    download_count INT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_share_links_token ON share_links(token);
CREATE INDEX idx_share_links_file_id ON share_links(file_id);
CREATE INDEX idx_share_links_created_by ON share_links(created_by);

-- +goose Down
DROP TABLE IF EXISTS share_links;