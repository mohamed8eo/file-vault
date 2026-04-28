package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mohamed8eo/file-vault/internal/db"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

func googleOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GOOGLE_REDIRECT_URL"),
		Scopes:       []string{"email", "profile"},
		Endpoint:     google.Endpoint,
	}
}

func githubOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
		ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		RedirectURL:  os.Getenv("GITHUB_REDIRECT_URL"),
		Scopes:       []string{"user:email"},
		Endpoint:     github.Endpoint,
	}
}

func (h *Handler) GoogleLogin(w http.ResponseWriter, r *http.Request) {
	cli := r.URL.Query().Get("cli")
	state := "web"
	if cli == "true" {
		state = "cli"
	}
	url := googleOAuthConfig().AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *Handler) GoogleCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}

	cfg := googleOAuthConfig()
	token, err := cfg.Exchange(context.Background(), code)
	if err != nil {
		http.Error(w, "failed to exchange code", http.StatusInternalServerError)
		return
	}

	client := cfg.Client(context.Background(), token)
	res, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		http.Error(w, "failed to get user info", http.StatusInternalServerError)
		return
	}

	defer res.Body.Close()

	var userInfo struct {
		ID    string `json:"id"`
		Email string `json:"email"`
		Name  string `json:"name"`
	}
	json.NewDecoder(res.Body).Decode(&userInfo)

	h.handleOAuthUser(w, r, userInfo.Email, userInfo.Name, userInfo.ID, "google", state)
}

func (h *Handler) GithubLogin(w http.ResponseWriter, r *http.Request) {
	cli := r.URL.Query().Get("cli")
	state := "web"
	if cli == "true" {
		state = "cli"
	}
	url := githubOAuthConfig().AuthCodeURL(state, oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func (h *Handler) GithubCallback(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}

	cfg := githubOAuthConfig()
	token, err := cfg.Exchange(context.TODO(), code)
	if err != nil {
		http.Error(w, "failed to exchange code", http.StatusInternalServerError)
		return
	}

	client := cfg.Client(context.TODO(), token)
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		http.Error(w, "failed to get user info", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var userInfo struct {
		ID    int    `json:"id"`
		Email string `json:"email"`
		Name  string `json:"login"`
	}
	json.NewDecoder(resp.Body).Decode(&userInfo)

	h.handleOAuthUser(w, r, userInfo.Email, userInfo.Name, fmt.Sprintf("%d", userInfo.ID), "github", state)
}

func (h *Handler) handleOAuthUser(w http.ResponseWriter, r *http.Request, email, name, providerID, provider, state string) {
	user, err := h.queries.GetUserByEmail(r.Context(), email)
	if err != nil && err != pgx.ErrNoRows {
		http.Error(w, "failed to get user", http.StatusInternalServerError)
		return
	}
	if err == pgx.ErrNoRows {
		user, err = h.queries.CreateUser(r.Context(), db.CreateUserParams{
			Email:          email,
			Name:           name,
			HashedPassword: "",
			Provider:       provider,
			ProviderID:     providerID,
		})
		if err != nil {
			http.Error(w, "failed to create user", http.StatusInternalServerError)
			return
		}
	}

	accessToken, refreshToken, err := h.generateTokens(user.ID.String())
	if err != nil {
		http.Error(w, "failed to generate tokens", http.StatusInternalServerError)
		return
	}

	h.queries.DeleteAllUserRefreshTokens(r.Context(), user.ID)
	h.queries.CreateRefreshToken(r.Context(), db.CreateRefreshTokenParams{
		Token:  refreshToken,
		UserID: user.ID,
		ExpiresAt: pgtype.Timestamp{
			Time:  time.Now().Add(refreshTokenExpiry),
			Valid: true,
		},
	})

	if state == "cli" {
		http.Redirect(w, r, fmt.Sprintf("http://localhost:9999?token=%s&refresh=%s", accessToken, refreshToken), http.StatusTemporaryRedirect)
		return
	}

	h.setRefreshCookie(w, refreshToken, time.Now().Add(refreshTokenExpiry))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"access_token": accessToken,
	})
}
