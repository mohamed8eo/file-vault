-- name: CreateRefreshToken :one
INSERT INTO
    refresh_tokens (token, user_id, expires_at)
VALUES
    ($1, $2, $3)
RETURNING
    *;

-- name: GetRefreshToken :one
SELECT
    *
FROM
    refresh_tokens
WHERE
    token = $1;

-- name: DeleteRefreshToken :exec
DELETE FROM refresh_tokens
WHERE
    token = $1;

-- name: DeleteAllUserRefreshTokens :exec
DELETE FROM refresh_tokens
WHERE
    user_id = $1;
