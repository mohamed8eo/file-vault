package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mohamed8eo/file-vault/internal/auth"
	"github.com/mohamed8eo/file-vault/internal/db"
)

type Handler struct {
	queries            *db.Queries
	accessTokenSecret  string
	refreshTokenSecret string
	isProduction       bool
}

const refreshTokenExpiry = 30 * 24 * time.Hour

func NewHandler(queries *db.Queries,
	accessTokenSecret string,
	refreshTokenSecret string,
	isProduction bool,
) *Handler {
	return &Handler{
		queries:            queries,
		accessTokenSecret:  accessTokenSecret,
		refreshTokenSecret: refreshTokenSecret,
		isProduction:       isProduction,
	}
}

func (h *Handler) SignUp(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	var req request

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Email == "" || req.Password == "" {
		http.Error(w, "name & email & password are require", http.StatusBadRequest)
		return
	}

	// check if user Already exist or not
	user, err := h.queries.GetUserByEmail(r.Context(), req.Email)
	if err != nil && err != pgx.ErrNoRows {
		http.Error(w, "failed to check user", http.StatusInternalServerError)
		return
	}
	if user.Email != "" {
		http.Error(w, "email already exists", http.StatusConflict)
		return
	}

	hashPassword, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, "failed to hash password", http.StatusInternalServerError)
		return
	}

	userData, err := h.queries.CreateUser(r.Context(), db.CreateUserParams{
		Email:          req.Email,
		Name:           req.Name,
		HashedPassword: hashPassword,
	})
	if err != nil {
		http.Error(w, "failed to create user", http.StatusInternalServerError)
		return
	}

	if h.accessTokenSecret == "" || h.refreshTokenSecret == "" {
		http.Error(w, "server misconfiguration", http.StatusInternalServerError)
		return
	}

	userIDStr := userData.ID.String()

	accessToken, refreshToken, err := h.generateTokens(userIDStr)
	if err != nil {
		http.Error(w, "failed to generate tokens", http.StatusInternalServerError)
		return
	}

	if _, err := h.queries.CreateRefreshToken(r.Context(), db.CreateRefreshTokenParams{
		Token:  refreshToken,
		UserID: userData.ID,
		ExpiresAt: pgtype.Timestamp{
			Time:  time.Now().Add(30 * 24 * time.Hour),
			Valid: true,
		},
	}); err != nil {
		http.Error(w, "falied to create refresh token on db", http.StatusInternalServerError)
		return
	}

	type response struct {
		Name        string `json:"name"`
		Email       string `json:"email"`
		AccessToken string `json:"access_token"`
	}

	res := &response{
		Name:        userData.Name,
		Email:       userData.Email,
		AccessToken: accessToken,
	}

	h.setRefreshCookie(w, refreshToken, time.Now().Add(refreshTokenExpiry))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(res)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	type request struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	var req request

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Password == "" {
		http.Error(w, " email & password are require", http.StatusBadRequest)
		return
	}

	userData, err := h.queries.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		http.Error(w, "invalid email or password", http.StatusUnauthorized)
		return
	}

	match, err := auth.CheckHash(userData.HashedPassword, req.Password)
	if err != nil {
		http.Error(w, "failed to check the hash password", http.StatusInternalServerError)
		return
	}

	if !match {
		http.Error(w, "invalid email or password", http.StatusUnauthorized)
		return
	}
	userIDStr := userData.ID.String()

	accessToken, refreshToken, err := h.generateTokens(userIDStr)
	if err != nil {
		http.Error(w, "failed to generate tokens", http.StatusInternalServerError)
		return
	}

	h.queries.DeleteAllUserRefreshTokens(r.Context(), userData.ID)

	if _, err := h.queries.CreateRefreshToken(r.Context(), db.CreateRefreshTokenParams{
		Token:  refreshToken,
		UserID: userData.ID,
		ExpiresAt: pgtype.Timestamp{
			Time:  time.Now().Add(30 * 24 * time.Hour),
			Valid: true,
		},
	}); err != nil {
		http.Error(w, "falied to create refresh token on db", http.StatusInternalServerError)
		return
	}

	type response struct {
		Name        string `json:"name"`
		Email       string `json:"email"`
		AccessToken string `json:"access_token"`
	}

	res := &response{
		Name:        userData.Name,
		Email:       userData.Email,
		AccessToken: accessToken,
	}

	h.setRefreshCookie(w, refreshToken, time.Now().Add(refreshTokenExpiry))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		http.Error(w, "missing refresh token", http.StatusUnauthorized)
		return
	}

	refreshToken := cookie.Value
	if err := h.queries.DeleteRefreshToken(r.Context(), refreshToken); err != nil {
		http.Error(w, "failed to delete the refresh token from db", http.StatusInternalServerError)
		return
	}

	h.setRefreshCookie(w, "", time.Unix(0, 0))
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("refresh_token")
	if err != nil {
		http.Error(w, "missing refresh token", http.StatusUnauthorized)
		return
	}

	refreshToken := cookie.Value
	_, err = auth.ValidateJWT(refreshToken, h.refreshTokenSecret)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	dbToken, err := h.queries.GetRefreshToken(r.Context(), refreshToken)
	if err == pgx.ErrNoRows {
		http.Error(w, "invalid refresh token", http.StatusUnauthorized)
		return
	}
	if err != nil {
		http.Error(w, "failed to get refresh token", http.StatusInternalServerError)
		return
	}

	h.queries.DeleteRefreshToken(r.Context(), refreshToken)

	accessToken, refreshToken, err := h.generateTokens(dbToken.UserID.String())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if _, err := h.queries.CreateRefreshToken(r.Context(), db.CreateRefreshTokenParams{
		Token:  refreshToken,
		UserID: dbToken.UserID,
		ExpiresAt: pgtype.Timestamp{
			Time:  time.Now().Add(refreshTokenExpiry),
			Valid: true,
		},
	}); err != nil {
		http.Error(w, "failed to create refresh token on db", http.StatusInternalServerError)
		return
	}

	type response struct {
		AccessToken string `json:"access_token"`
	}

	res := &response{
		AccessToken: accessToken,
	}

	h.setRefreshCookie(w, refreshToken, time.Now().Add(refreshTokenExpiry))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res)
}

func (h *Handler) generateTokens(userID string) (string, string, error) {
	accessToken, err := auth.MakeToken(
		string(auth.AccessToken),
		h.accessTokenSecret,
		userID,
		15*time.Minute,
	)
	if err != nil {
		return "", "", err
	}

	refreshToken, err := auth.MakeToken(
		string(auth.RefreshToken),
		h.refreshTokenSecret,
		userID,
		30*24*time.Hour,
	)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (h *Handler) setRefreshCookie(w http.ResponseWriter, token string, expiresAt time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		HttpOnly: true,
		Secure:   h.isProduction,
		SameSite: http.SameSiteLaxMode,
		Path:     "/",
		Expires:  expiresAt,
	})
}
