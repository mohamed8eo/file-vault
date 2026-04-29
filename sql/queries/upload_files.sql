-- name: CreateFile :one
INSERT INTO
    files (user_id, file_name, file_url, file_size)
VALUES
    ($1, $2, $3, $4)
RETURNING
    *;

-- name: GetFilesByUser :many
SELECT
    id, user_id, file_name, file_url, file_size, created_at
FROM
    files
WHERE
    user_id = $1
ORDER BY
    created_at DESC
LIMIT
    $2
OFFSET
    $3;

-- name: GetFilesFiltered :many
SELECT id, user_id, file_name, file_url, file_size, created_at
FROM files
WHERE user_id = $1
AND (COALESCE($4, '') = '' OR file_name ILIKE '%' || $4 || '%')
AND (
    COALESCE($5, '') = '' OR
    ($5 = 'image' AND (file_url ILIKE '%.jpg' OR file_url ILIKE '%.jpeg' OR file_url ILIKE '%.png' OR file_url ILIKE '%.gif' OR file_url ILIKE '%.webp' OR file_url ILIKE '%.svg' OR file_url ILIKE '%.JPG' OR file_url ILIKE '%.PNG')) OR
    ($5 = 'video' AND (file_url ILIKE '%.mp4' OR file_url ILIKE '%.webm' OR file_url ILIKE '%.mov' OR file_url ILIKE '%.avi' OR file_url ILIKE '%.mkv' OR file_url ILIKE '%.MP4')) OR
    ($5 = 'document' AND (file_url ILIKE '%.pdf' OR file_url ILIKE '%.doc' OR file_url ILIKE '%.txt' OR file_url ILIKE '%.PDF'))
)
ORDER BY
    CASE WHEN $2 = 'name' THEN file_name END ASC,
    CASE WHEN $2 = 'size' THEN file_size END DESC,
    CASE WHEN $2 = '' OR $2 = 'date' THEN created_at END DESC
LIMIT $3 OFFSET $6;

-- name: GetFileByID :one
SELECT
    *
FROM
    files
WHERE
    id = $1
    AND user_id = $2;

-- name: DeleteFileByID :exec
DELETE FROM files
WHERE
    id = $1
    AND user_id = $2;

-- name: DeleteFilesByIDs :many
DELETE FROM files
WHERE
    user_id = $1
    AND id = ANY($2::uuid[])
RETURNING id;

-- name: SearchFiles :many
SELECT
    *
FROM
    files
WHERE
    user_id = $1
    AND file_name ILIKE '%' || $2 || '%'
ORDER BY
    created_at DESC
LIMIT
    $3;
