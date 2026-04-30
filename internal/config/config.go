package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	// Server
	Port string

	// Database
	DBURL string

	// JWT
	AccessTokenSecret  string
	RefreshTokenSecret string

	// Environment
	IsProduction bool

	// Redis
	RedisURL string

	// AWS S3
	S3Bucket      string
	S3Region      string
	CloudFrontURL string

	// OAuth
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string

	GithubClientID     string
	GithubClientSecret string
	GithubRedirectURL  string

	// Email (Resend)
	ResendAPIKey string
	DevMode      bool
}

func Load() *Config {
	godotenv.Load(".env")

	return &Config{
		Port:                  getEnv("PORT", "3000"),
		DBURL:                 getEnv("DB_URL", ""),
		AccessTokenSecret:     getEnv("ACCESS_TOKEN_SECRET", ""),
		RefreshTokenSecret:    getEnv("REFRESH_TOKEN_SECRET", ""),
		IsProduction:          getEnv("IS_PRODUCTION", "false") == "true",
		RedisURL:              getEnv("REDIS_URL", ""),
		S3Bucket:              getEnv("S3_BUCKET", ""),
		S3Region:              getEnv("S3_REGION", ""),
		CloudFrontURL:         getEnv("CLOUDFRONT_DOMAIN", ""),
		GoogleClientID:         getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret:     getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:      getEnv("GOOGLE_REDIRECT_URL", ""),
		GithubClientID:         getEnv("GITHUB_CLIENT_ID", ""),
		GithubClientSecret:     getEnv("GITHUB_CLIENT_SECRET", ""),
		GithubRedirectURL:      getEnv("GITHUB_REDIRECT_URL", ""),
		ResendAPIKey:          getEnv("RESEND_API_KEY", ""),
		DevMode:               getEnv("DEV_MODE", "false") == "true",
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}