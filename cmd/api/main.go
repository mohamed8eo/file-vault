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

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.Load()

	if cfg.DBURL == "" {
		log.Fatal("DB_URL must be set")
	}
	if cfg.AccessTokenSecret == "" {
		log.Fatal("ACCESS_TOKEN_SECRET must be set")
	}
	if cfg.RefreshTokenSecret == "" {
		log.Fatal("REFRESH_TOKEN_SECRET must be set")
	}
	if cfg.RedisURL == "" {
		log.Fatal("REDIS_URL must be set")
	}
	if cfg.S3Bucket == "" {
		log.Fatal("S3_BUCKET must be set")
	}
	if cfg.S3Region == "" {
		log.Fatal("S3_REGION must be set")
	}
	if cfg.CloudFrontURL == "" {
		log.Fatal("CLOUDFRONT_DOMAIN must be set")
	}

	conn, err := pgx.Connect(context.Background(), cfg.DBURL)
	if err != nil {
		log.Fatalf("error: %s\n", err.Error())
	}
	defer conn.Close(context.Background())

	dbQueries := db.New(conn)

	awsCfg, err := awsconfig.LoadDefaultConfig(
		context.TODO(),
		awsconfig.WithRegion(cfg.S3Region),
	)
	if err != nil {
		log.Fatalf("error: %s\n", err.Error())
	}

	s3Client := s3.NewFromConfig(awsCfg)

	port := cfg.Port
	if port == "" {
		port = "3000"
	}

	mux := http.NewServeMux()

	docs.SwaggerInfo.Title = "File Vault API"
	docs.SwaggerInfo.Description = "A secure file storage REST API with S3 and CloudFront"
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.Host = "localhost:" + port
	docs.SwaggerInfo.Schemes = []string{"http", "https"}

	docsPath := "cmd/api/docs"
	mux.Handle("/docs/", http.StripPrefix("/docs/", http.FileServer(http.Dir(docsPath))))
	mux.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, docsPath+"/index.html")
	})

	auth := handler.NewHandler(dbQueries, cfg)
	uploadHandler := handler.NewUploadHandler(
		dbQueries,
		cfg.S3Bucket,
		cfg.S3Region,
		cfg.CloudFrontURL,
		s3Client,
	)

	parseRedisURL, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		log.Fatalf("error: %s\n", err.Error())
	}

	rdb := redis.NewClient(parseRedisURL)

	registerRoutes(mux, auth, uploadHandler, rdb, cfg.AccessTokenSecret)

	logCh := middleware.StartLogWorker(dbQueries)
	wrappedMux := middleware.RequestID(middleware.Logging(logCh)(mux))

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: wrappedMux,
	}

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
