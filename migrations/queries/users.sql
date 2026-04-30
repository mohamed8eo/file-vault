-- name: CreateUser :one
INSERT INTO
    users (email, name, hashed_password, provider, provider_id)
VALUES
    ($1, $2, $3, $4, $5)
RETURNING
    *;

-- name: GetUserByID :one
SELECT
    *
FROM
    users
WHERE
    id = $1;

-- name: GetUserByEmail :one
SELECT
    *
FROM
    users
WHERE
    email = $1;

-- name: GetUsers :many
SELECT
    *
FROM
    users
LIMIT
    30;
