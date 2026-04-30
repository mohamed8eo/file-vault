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

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jackc/pgx/v5"
	"github.com/mohamed8eo/file-vault/cmd/api/docs"
	"github.com/mohamed8eo/file-vault/internal/config"
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

	// Use centralized config
	cfg := config.Load()

	dbURL := cfg.DBURL
	if dbURL == "" {
		log.Fatal("DB_URL must be set")
	}
	if dbURL == "" {
		log.Fatal("DB_URL must be set")
	}

	conn, err := pgx.Connect(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("error: %s\n", err.Error())
	}
	defer conn.Close(context.Background())

	dbQueries := db.New(conn)

	accessTokenSecret := cfg.AccessTokenSecret
	if accessTokenSecret == "" {
		log.Fatal("accessTokenSecret must be set")
	}

	refreshTokenSecret := cfg.RefreshTokenSecret
	if refreshTokenSecret == "" {
		log.Fatal("refreshTokenSecret must be set")
	}

	redisURL := cfg.RedisURL
	if redisURL == "" {
		log.Fatal("REDIS_URL must be set")
	}

	isProduction := cfg.IsProduction

	s3Bucket := cfg.S3Bucket
	if s3Bucket == "" {
		log.Fatal("S3_BUCKET environment variable is not set")
	}

	s3Region := cfg.S3Region
	if s3Region == "" {
		log.Fatal("S3_REGION environment variable is not set")
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(
		context.TODO(),
		awsconfig.WithRegion(s3Region),
	)
	if err != nil {
		log.Fatalf("error: %s\n", err.Error())
	}

	s3CloudFront := cfg.CloudFrontURL
	if s3CloudFront == "" {
		log.Fatal("CLOUDFRONT_DOMAIN environment variable is not set")
	}

	client := s3.NewFromConfig(awsCfg)

	appCfg := &apiConfig{
		dbQueries:          dbQueries,
		accessTokenSecret:  accessTokenSecret,
		refreshTokenSecret: refreshTokenSecret,
		isProduction:       isProduction,
		s3Region:           s3Region,
		s3Bucket:           s3Bucket,
		s3Client:           client,
		s3CloudFront:       s3CloudFront,
	}

	port := cfg.Port
	if port == "" {
		port = "3000"
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
		appCfg.dbQueries,
		appCfg.accessTokenSecret,
		appCfg.refreshTokenSecret,
		appCfg.isProduction,
	)
	uploadHandler := handler.NewUploadHandler(
		appCfg.dbQueries,
		appCfg.s3Bucket,
		appCfg.s3Region,
		appCfg.s3CloudFront,
		appCfg.s3Client,
	)

	parseRedisURL, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Fatalf("error: %s\n", err.Error())
	}

	rdb := redis.NewClient(parseRedisURL)

	authMiddleware := middleware.Auth(appCfg.accessTokenSecret)

	loginRL := middleware.RateLimit(rdb, 10, time.Minute)
	uploadRL := middleware.RateLimit(rdb, 20, time.Minute)
	generalRL := middleware.RateLimit(rdb, 100, time.Minute)

	// Prebuilt stacks
	loginStack := middleware.CreateStack(loginRL)
	uploadStack := middleware.CreateStack(uploadRL, authMiddleware)
	fileStack := middleware.CreateStack(generalRL, authMiddleware)

	mux.Handle("GET /health", generalRL(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})))

	// -- Auth --
	mux.Handle("POST /auth/sign-up", loginStack(http.HandlerFunc(auth.SignUp)))
	mux.Handle("POST /auth/login", loginStack(http.HandlerFunc(auth.Login)))
	mux.Handle("POST /auth/verify-otp", loginStack(http.HandlerFunc(auth.VerifyOTP)))
	mux.Handle("POST /auth/refresh", loginStack(http.HandlerFunc(auth.Refresh)))
	mux.Handle("POST /auth/logout", loginStack(http.HandlerFunc(auth.Logout)))

	// -- OAuth --
	mux.Handle("GET /auth/google", loginStack(http.HandlerFunc(auth.GoogleLogin)))
	mux.Handle("GET /auth/google/callback", loginStack(http.HandlerFunc(auth.GoogleCallback)))
	mux.Handle("GET /auth/github", loginStack(http.HandlerFunc(auth.GithubLogin)))
	mux.Handle("GET /auth/github/callback", loginStack(http.HandlerFunc(auth.GithubCallback)))

	// -- Upload --
	mux.Handle("POST /upload", uploadStack(http.HandlerFunc(uploadHandler.UploadFile)))
	mux.Handle("POST /upload/image", uploadStack(http.HandlerFunc(uploadHandler.UploadImage)))
	mux.Handle("POST /upload/video", uploadStack(http.HandlerFunc(uploadHandler.UploadVideo)))

	// -- Files --
	mux.Handle("GET /files", fileStack(http.HandlerFunc(uploadHandler.GetFiles)))
	mux.Handle("GET /files/search", fileStack(http.HandlerFunc(uploadHandler.SearchFiles)))
	mux.Handle("GET /files/stats", fileStack(http.HandlerFunc(uploadHandler.GetStorageStats)))
	mux.Handle("GET /files/{id}", fileStack(http.HandlerFunc(uploadHandler.GetFileByID)))
	mux.Handle("DELETE /files/{id}", fileStack(http.HandlerFunc(uploadHandler.DeleteFile)))
	mux.Handle("POST /files/delete", fileStack(http.HandlerFunc(uploadHandler.DeleteFiles)))

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
