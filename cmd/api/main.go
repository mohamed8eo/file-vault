package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"github.com/mohamed8eo/file-vault/internal/db"
	"github.com/mohamed8eo/file-vault/internal/handler"
	"github.com/mohamed8eo/file-vault/internal/middleware"
	"github.com/redis/go-redis/v9"
)

type apiConfig struct {
	dbQueries          *db.Queries
	accessTokenSecret  string
	refreshTokenSecret string
	isProduction       bool
	s3Bucket           string
	s3Region           string
	s3Client           *s3.Client
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	godotenv.Load(".env")

	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		log.Fatal("DB_URL must be set")
	}

	conn, err := pgx.Connect(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("error: %s\n", err.Error())
	}
	defer conn.Close(context.Background())

	dbQueries := db.New(conn)

	accessTokenSecret := os.Getenv("ACCESS_TOKEN_SECRET")
	if accessTokenSecret == "" {
		log.Fatal("accessTokenSecret must be set")
	}

	refreshTokenSecret := os.Getenv("REFRESH_TOKEN_SECRET")
	if refreshTokenSecret == "" {
		log.Fatal("refreshTokenSecret must be set")
	}

	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		log.Fatal("radis must be set")
	}

	isProduction := os.Getenv("IS_PRODUCTION") == "true"

	s3Bucket := os.Getenv("S3_BUCKET")
	if s3Bucket == "" {
		log.Fatal("S3_BUCKET environment variable is not set")
	}

	s3Region := os.Getenv("S3_REGION")
	if s3Region == "" {
		log.Fatal("S3_REGION environment variable is not set")
	}

	awsCfg, err := config.LoadDefaultConfig(
		context.TODO(),
		config.WithRegion(s3Region),
	)
	if err != nil {
		log.Fatalf("error: %s\n", err.Error())
	}

	client := s3.NewFromConfig(awsCfg)

	cfg := &apiConfig{
		dbQueries:          dbQueries,
		accessTokenSecret:  accessTokenSecret,
		refreshTokenSecret: refreshTokenSecret,
		isProduction:       isProduction,
		s3Region:           s3Region,
		s3Bucket:           s3Bucket,
		s3Client:           client,
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	auth := handler.NewHandler(
		cfg.dbQueries,
		cfg.accessTokenSecret,
		cfg.refreshTokenSecret,
		cfg.isProduction,
	)
	uploadHandler := handler.NewUploadHandler(
		cfg.dbQueries,
		cfg.s3Bucket,
		cfg.s3Region,
		cfg.s3Client,
	)

	parseRedisURL, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("error: %s\n", err.Error())
	}

	rdb := redis.NewClient(parseRedisURL)

	authMiddleware := middleware.Auth(cfg.accessTokenSecret)

	// Customize RateLimit
	loginRateLimit := middleware.RateLimit(rdb, 10, time.Minute)
	uploadRateLimit := middleware.RateLimit(rdb, 20, time.Minute)
	generalRateLimit := middleware.RateLimit(rdb, 100, time.Minute)

	mux.Handle("GET /", generalRateLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello"))
	})))

	// authentication route.
	mux.Handle("POST /auth/sign-up", loginRateLimit(http.HandlerFunc(auth.SignUp)))
	mux.Handle("POST /auth/login", loginRateLimit(http.HandlerFunc(auth.Login)))
	mux.Handle("POST /auth/refresh", loginRateLimit(http.HandlerFunc(auth.Refresh)))
	mux.Handle("POST /auth/logout", loginRateLimit(http.HandlerFunc(auth.Logout)))

	// upload route.
	mux.Handle("POST /upload", uploadRateLimit(authMiddleware(http.HandlerFunc(uploadHandler.UploadFile))))
	mux.Handle("GET /files", generalRateLimit(authMiddleware(http.HandlerFunc(uploadHandler.GetFiles))))
	mux.Handle("GET /files/{id}", generalRateLimit(authMiddleware(http.HandlerFunc(uploadHandler.GetFileByID))))
	mux.Handle("DELETE /files/{id}", generalRateLimit(authMiddleware(http.HandlerFunc(uploadHandler.DeleteFile))))

	// wrap entire mux with logging + request ID middleware
	// wrappedMux := middleware.Logging(middleware.RequestID(mux))
	wrappedMux := middleware.RequestID(middleware.Logging(mux))

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: wrappedMux,
	}

	// ShutDown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	go func() {
		slog.Info("server starting", "port", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "err", err)
	}

	slog.Info("server stopped")
}
