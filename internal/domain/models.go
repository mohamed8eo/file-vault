package domain

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type User struct {
	ID           pgtype.UUID    `json:"id"`
	Name         string         `json:"name"`
	Email        string         `json:"email"`
	Password     string         `json:"-"`
	VerifiedAt   pgtype.Timestamp `json:"verified_at,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

type File struct {
	ID          pgtype.UUID    `json:"id"`
	UserID      pgtype.UUID    `json:"user_id"`
	Name        string         `json:"name"`
	Size        int64          `json:"size"`
	ContentType string         `json:"content_type"`
	S3Key       string         `json:"s3_key"`
	S3URL       string         `json:"s3_url,omitempty"`
	DownloadURL string         `json:"download_url,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

type RefreshToken struct {
	ID        pgtype.UUID    `json:"id"`
	UserID    pgtype.UUID    `json:"user_id"`
	Token     string         `json:"token"`
	ExpiresAt pgtype.Timestamp `json:"expires_at"`
	CreatedAt time.Time      `json:"created_at"`
}

type OTP struct {
	ID          pgtype.UUID    `json:"id"`
	UserID      pgtype.UUID    `json:"user_id"`
	Otp         string         `json:"otp"`
	OtpExpiresAt pgtype.Timestamp `json:"otp_expires_at"`
	CreatedAt   time.Time      `json:"created_at"`
}

type StorageStats struct {
	TotalFiles   int64   `json:"total_files"`
	TotalSize    int64   `json:"total_size"`
	ImagesCount  int64   `json:"images_count"`
	ImagesSize   int64   `json:"images_size"`
	VideosCount  int64   `json:"videos_count"`
	VideosSize   int64   `json:"videos_size"`
	DocumentsCount int64  `json:"documents_count"`
	DocumentsSize int64   `json:"documents_size"`
}

type PaginationParams struct {
	Limit   int
	Page    int
	Offset  int
	Sort    string
	FileType string
}

type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
}