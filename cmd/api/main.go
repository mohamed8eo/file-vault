package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"github.com/mohamed8eo/file-vault/internal/db"
	"github.com/mohamed8eo/file-vault/internal/handler"
)

type apiConfig struct {
	dbQueries          *db.Queries
	accessTokenSecret  string
	refreshTokenSecret string
	isProduction       bool
}

func main() {
	godotenv.Load()

	// Connect with db
	conn, err := pgx.Connect(context.Background(), os.Getenv("DB_URL"))
	if err != nil {
		log.Fatalf("error: %s\n", err.Error())
	}
	defer conn.Close(context.Background())

	dbQueries := db.New(conn)
	accessTokenSecret := os.Getenv("ACCESS_TOKEN_SECRET")
	refreshTokenSecret := os.Getenv("REFRESH_TOKEN_SECRET")
	isProduction := os.Getenv("IS_PRODUCTION") == "true"
	cfg := &apiConfig{
		dbQueries:          dbQueries,
		accessTokenSecret:  accessTokenSecret,
		refreshTokenSecret: refreshTokenSecret,
		isProduction:       isProduction,
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	mux := http.NewServeMux()
	handler := handler.NewHandler(
		cfg.dbQueries,
		cfg.accessTokenSecret,
		cfg.refreshTokenSecret,
		cfg.isProduction,
	)

	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello"))
	})
	mux.HandleFunc("POST /auth/sign-up", handler.SignUp)
	mux.HandleFunc("POST /auth/login", handler.Login)
	mux.HandleFunc("POST /auth/refresh", handler.Refresh)
	mux.HandleFunc("POST /auth/logout", handler.Logout)

	log.Println("DB is running")
	log.Printf("server is running on PORT: %s\n", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("error to make server runn,err: %s", err)
	}
}
