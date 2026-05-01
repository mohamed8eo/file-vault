-- name: CreateShareLink :one
INSERT INTO share_links (file_id, token, created_by, expires_at, password_hash, max_downloads)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, file_id, token, created_by, expires_at, password_hash, max_downloads, download_count, created_at;

-- name: GetShareLinkByToken :one
SELECT id, file_id, token, created_by, expires_at, password_hash, max_downloads, download_count, created_at
FROM share_links
WHERE token = $1;

-- name: GetShareLinksByFileID :many
SELECT id, file_id, token, created_by, expires_at, password_hash, max_downloads, download_count, created_at
FROM share_links
WHERE file_id = $1
ORDER BY created_at DESC;

-- name: GetShareLinksByUser :many
SELECT sl.id, sl.file_id, sl.token, sl.created_by, sl.expires_at, sl.password_hash, sl.max_downloads, sl.download_count, sl.created_at, f.file_name
FROM share_links sl
JOIN files f ON sl.file_id = f.id
WHERE sl.created_by = $1
ORDER BY sl.created_at DESC
LIMIT $2 OFFSET $3;

-- name: IncrementDownloadCount :exec
UPDATE share_links
SET download_count = download_count + 1
WHERE id = $1;

-- name: DeleteShareLink :exec
DELETE FROM share_links
WHERE id = $1 AND created_by = $2;

-- name: GetShareLinkByID :one
SELECT id, file_id, token, created_by, expires_at, password_hash, max_downloads, download_count, created_at
FROM share_links
WHERE id = $1 AND created_by = $2;