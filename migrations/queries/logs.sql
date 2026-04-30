-- name: CreateRequestLog :exec
INSERT INTO
    request_logs (
        method,
        path,
        status,
        latency_ms,
        request_id,
        user_id
    )
VALUES
    ($1, $2, $3, $4, $5, $6);

-- name: GetRequestLogs :many
SELECT
    *
FROM
    request_logs
ORDER BY
    created_at DESC
LIMIT
    100;

-- name: GetRequestLogsByUser :many
SELECT
    *
FROM
    request_logs
WHERE
    user_id = $1
ORDER BY
    created_at DESC
LIMIT
    100;

-- name: GetRequestLogsByStatus :many
SELECT
    *
FROM
    request_logs
WHERE
    status = $1
ORDER BY
    created_at DESC
LIMIT
    100;
