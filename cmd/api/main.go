package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"github.com/mohamed8eo/file-vault/internal/db"
	"github.com/mohamed8eo/file-vault/internal/handler"
	"github.com/mohamed8eo/file-vault/internal/middleware"
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
		log.Fatal("refreshTokenSecret  must be set")
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
	uploadHanlder := handler.NewUploadHandler(
		cfg.dbQueries,
		cfg.s3Bucket,
		cfg.s3Region,
		cfg.s3Client,
	)

	authMiddleware := middleware.Auth(cfg.accessTokenSecret)

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello"))
	})
	// authentication route.
	mux.HandleFunc("POST /auth/sign-up", auth.SignUp)
	mux.HandleFunc("POST /auth/login", auth.Login)
	mux.HandleFunc("POST /auth/refresh", auth.Refresh)
	mux.HandleFunc("POST /auth/logout", auth.Logout)

	mux.Handle("POST /upload", authMiddleware(http.HandlerFunc(uploadHanlder.UploadFile)))
	mux.Handle("GET /files", authMiddleware(http.HandlerFunc(uploadHanlder.GetFiles)))
	mux.Handle("GET /files/{id}", authMiddleware(http.HandlerFunc(uploadHanlder.GetFileByID)))
	mux.Handle("DELETE /files/{id}", authMiddleware(http.HandlerFunc(uploadHanlder.DeleteFile)))

	log.Println("DB is running")
	log.Printf("server is running on PORT: %s\n", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("error to make server runn,err: %s", err)
	}
}
