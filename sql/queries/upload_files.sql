-- name: CreateFile :one
INSERT INTO
    files (user_id, file_name, file_url)
VALUES
    ($1, $2, $3)
RETURNING
    *;

-- name: GetFilesByUser :many
SELECT
    *
FROM
    files
WHERE
    user_id = $1
ORDER BY
    created_at DESC;

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
