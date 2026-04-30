package main

import (
	"net/http"
	"time"

	"github.com/mohamed8eo/file-vault/internal/handler"
	"github.com/mohamed8eo/file-vault/internal/middleware"
	"github.com/redis/go-redis/v9"
)

func registerRoutes(mux *http.ServeMux, auth *handler.Handler, uploadHandler *handler.UploadHanlder, rdb *redis.Client, accessTokenSecret string) {

	authMiddleware := middleware.Auth(accessTokenSecret)

	loginRL := middleware.RateLimit(rdb, 10, time.Minute)
	uploadRL := middleware.RateLimit(rdb, 20, time.Minute)
	generalRL := middleware.RateLimit(rdb, 100, time.Minute)

	loginStack := middleware.CreateStack(loginRL)
	uploadStack := middleware.CreateStack(uploadRL, authMiddleware)
	fileStack := middleware.CreateStack(generalRL, authMiddleware)

	mux.Handle("GET /health", generalRL(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})))

	// Auth
	mux.Handle("POST /auth/sign-up", loginStack(http.HandlerFunc(auth.SignUp)))
	mux.Handle("POST /auth/login", loginStack(http.HandlerFunc(auth.Login)))
	mux.Handle("POST /auth/verify-otp", loginStack(http.HandlerFunc(auth.VerifyOTP)))
	mux.Handle("POST /auth/refresh", loginStack(http.HandlerFunc(auth.Refresh)))
	mux.Handle("POST /auth/logout", loginStack(http.HandlerFunc(auth.Logout)))

	// OAuth
	mux.Handle("GET /auth/google", loginStack(http.HandlerFunc(auth.GoogleLogin)))
	mux.Handle("GET /auth/google/callback", loginStack(http.HandlerFunc(auth.GoogleCallback)))
	mux.Handle("GET /auth/github", loginStack(http.HandlerFunc(auth.GithubLogin)))
	mux.Handle("GET /auth/github/callback", loginStack(http.HandlerFunc(auth.GithubCallback)))

	// Upload
	mux.Handle("POST /upload", uploadStack(http.HandlerFunc(uploadHandler.UploadFile)))
	mux.Handle("POST /upload/image", uploadStack(http.HandlerFunc(uploadHandler.UploadImage)))
	mux.Handle("POST /upload/video", uploadStack(http.HandlerFunc(uploadHandler.UploadVideo)))

	// Files
	mux.Handle("GET /files", fileStack(http.HandlerFunc(uploadHandler.GetFiles)))
	mux.Handle("GET /files/search", fileStack(http.HandlerFunc(uploadHandler.SearchFiles)))
	mux.Handle("GET /files/stats", fileStack(http.HandlerFunc(uploadHandler.GetStorageStats)))
	mux.Handle("GET /files/{id}", fileStack(http.HandlerFunc(uploadHandler.GetFileByID)))
	mux.Handle("DELETE /files/{id}", fileStack(http.HandlerFunc(uploadHandler.DeleteFile)))
	mux.Handle("POST /files/delete", fileStack(http.HandlerFunc(uploadHandler.DeleteFiles)))
}