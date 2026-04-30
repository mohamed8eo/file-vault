-- name: SaveOTP :exec
UPDATE users
SET
    otp = $1,
    otp_expires_at = $2
WHERE
    id = $3;

-- name: MarkUserVerified :exec
UPDATE users
SET
    verified_at = NOW(),
    otp = NULL,
    otp_expires_at = NULL
WHERE
    id = $1;
