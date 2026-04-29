package main

// @title			File Vault API
// @version		1.0
// @description	A secure file storage REST API with S3 and CloudFront
// @termsOfService	http://swagger.io/terms/

// @contact.name	API Support
// @contact.url		https://github.com/mohamed8eo/file-vault
// @contact.email	support@filevault.local

// @license.name	Apache 2.0
// @license.url		http://www.apache.org/licenses/LICENSE-2.0.html

// @host		localhost:3000
// @BasePath	/

// @securityDefinitions.apikey	ApiKeyAuth
// @in							header
// @name						Authorization
// @description				Type "Bearer" followed by a space and JWT token.

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
	"github.com/mohamed8eo/file-vault/cmd/api/docs"
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
	s3CloudFront       string
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

	s3CloudFront := os.Getenv("CLOUDFRONT_DOMAIN")
	if s3CloudFront == "" {
		log.Fatal("S3_REGION environment variable is not set")
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
		s3CloudFront:       s3CloudFront,
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	// Initialize swagger docs
	docs.SwaggerInfo.Title = "File Vault API"
	docs.SwaggerInfo.Description = "A secure file storage REST API with S3 and CloudFront"
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Host = "localhost:" + port
	docs.SwaggerInfo.Schemes = []string{"http", "https"}

	// Route "/" must be registered AFTER more specific paths to avoid conflicts
	// So register swagger first

	// Swagger UI - use absolute path from project root
	docsPath := "cmd/api/docs"
	mux.Handle("/docs/", http.StripPrefix("/docs/", http.FileServer(http.Dir(docsPath))))
	mux.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, docsPath+"/index.html")
	})

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
		cfg.s3CloudFront,
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

	mux.Handle("GET /health", generalRateLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})))

	// authentication route.
	mux.Handle("POST /auth/sign-up", loginRateLimit(http.HandlerFunc(auth.SignUp)))
	mux.Handle("POST /auth/login", loginRateLimit(http.HandlerFunc(auth.Login)))
	mux.Handle("POST /auth/refresh", loginRateLimit(http.HandlerFunc(auth.Refresh)))
	mux.Handle("POST /auth/logout", loginRateLimit(http.HandlerFunc(auth.Logout)))

	// oAuth route
	mux.Handle("GET /auth/google", loginRateLimit(http.HandlerFunc(auth.GoogleLogin)))
	mux.Handle("GET /auth/google/callback", loginRateLimit(http.HandlerFunc(auth.GoogleCallback)))
	mux.Handle("GET /auth/github", loginRateLimit(http.HandlerFunc(auth.GithubLogin)))
	mux.Handle("GET /auth/github/callback", loginRateLimit(http.HandlerFunc(auth.GithubCallback)))

	// upload route.
	mux.Handle("POST /upload", uploadRateLimit(authMiddleware(http.HandlerFunc(uploadHandler.UploadFile))))
	mux.Handle("POST /upload/image", uploadRateLimit(authMiddleware(http.HandlerFunc(uploadHandler.UploadImage))))
	mux.Handle("POST /upload/video", uploadRateLimit(authMiddleware(http.HandlerFunc(uploadHandler.UploadVideo))))
	mux.Handle("GET /files", generalRateLimit(authMiddleware(http.HandlerFunc(uploadHandler.GetFiles))))
	mux.Handle("GET /files/search", generalRateLimit(authMiddleware(http.HandlerFunc(uploadHandler.SearchFiles))))
	mux.Handle("GET /files/stats", generalRateLimit(authMiddleware(http.HandlerFunc(uploadHandler.GetStorageStats))))
	mux.Handle("GET /files/{id}", generalRateLimit(authMiddleware(http.HandlerFunc(uploadHandler.GetFileByID))))
	mux.Handle("DELETE /files/{id}", generalRateLimit(authMiddleware(http.HandlerFunc(uploadHandler.DeleteFile))))
	mux.Handle("POST /files/delete", generalRateLimit(authMiddleware(http.HandlerFunc(uploadHandler.DeleteFiles))))

	// wrap entire mux with logging + request ID middleware
	// wrappedMux := middleware.Logging(middleware.RequestID(mux))
	logCh := middleware.StartLogWorker(dbQueries)
	wrappedMux := middleware.RequestID(middleware.Logging(logCh)(mux))

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
