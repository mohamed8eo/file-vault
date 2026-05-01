package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mohamed8eo/file-vault/internal/auth"
	"github.com/mohamed8eo/file-vault/internal/config"
	"github.com/mohamed8eo/file-vault/internal/db"
	"github.com/mohamed8eo/file-vault/internal/middleware"

	"github.com/google/uuid"
)

type ShareLinkHandler struct {
	dbQueries *db.Queries
	config    *config.Config
}

func NewShareLinkHandler(queries *db.Queries, cfg *config.Config) *ShareLinkHandler {
	return &ShareLinkHandler{
		dbQueries: queries,
		config:    cfg,
	}
}

// @Summary		Create share link
// @Description	Create a shareable link for a file with optional expiration, password, and download limits
// @Tags			Share Links
// @Accept			json
// @Produce		json
// @Security		ApiKeyAuth
// @Param			id	path		string	true	"File ID"
// @Param			share	body		object	true	"Share link options"
// @Success		201	{object}	map[string]string
// @Failure		400	{object}	map[string]string
// @Failure		401	{object}	map[string]string
// @Failure		404	{object}	map[string]string
// @Router			/files/{id}/share [post]
func (h *ShareLinkHandler) CreateShareLink(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	fileIDStr := r.PathValue("id")
	if fileIDStr == "" {
		http.Error(w, "file ID is required", http.StatusBadRequest)
		return
	}

	fileID, err := uuid.Parse(fileIDStr)
	if err != nil {
		http.Error(w, "invalid file ID", http.StatusBadRequest)
		return
	}

	file, err := h.dbQueries.GetFileByID(r.Context(), db.GetFileByIDParams{
		ID:     pgtype.UUID{Bytes: fileID, Valid: true},
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
	})
	if err == pgx.ErrNoRows {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "failed to get file", http.StatusInternalServerError)
		return
	}

	type request struct {
		ExpiresAt    *time.Time `json:"expires_at"`
		Password     string     `json:"password"`
		MaxDownloads *int       `json:"max_downloads"`
	}

	var req request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	token := generateShareToken()

	var passwordHash pgtype.Text
	if req.Password != "" {
		hash, err := auth.HashPassword(req.Password)
		if err != nil {
			http.Error(w, "failed to hash password", http.StatusInternalServerError)
			return
		}
		passwordHash = pgtype.Text{String: hash, Valid: true}
	}

	var expiresAt pgtype.Timestamptz
	if req.ExpiresAt != nil {
		expiresAt = pgtype.Timestamptz{Time: *req.ExpiresAt, Valid: true}
	}

	var maxDownloads pgtype.Int4
	if req.MaxDownloads != nil {
		maxDownloads = pgtype.Int4{Int32: int32(*req.MaxDownloads), Valid: true}
	}

	shareLink, err := h.dbQueries.CreateShareLink(r.Context(), db.CreateShareLinkParams{
		FileID:        pgtype.UUID{Bytes: fileID, Valid: true},
		Token:         token,
		CreatedBy:     pgtype.UUID{Bytes: userID, Valid: true},
		ExpiresAt:     expiresAt,
		PasswordHash: passwordHash,
		MaxDownloads: maxDownloads,
	})
	if err != nil {
		http.Error(w, "failed to create share link", http.StatusInternalServerError)
		return
	}

	shareURL := fmt.Sprintf("%s/s/%s", h.config.APIBaseURL, shareLink.Token)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
		"message":       "share link created",
		"share_url":    shareURL,
		"token":         shareLink.Token,
		"expires_at":   shareLink.ExpiresAt.Time,
		"max_downloads": shareLink.MaxDownloads.Int32,
		"file_name":    file.FileName,
	})
}

// @Summary		Get shared file
// @Description	Access a shared file via token (public, no auth required)
// @Tags			Share Links
// @Produce		json
// @Param			token	path		string	true	"Share token"
// @Success		200	{object}	map[string]interface{}
// @Failure		400	{object}	map[string]string
// @Failure		404	{object}	map[string]string
// @Router			/share/{token} [get]
func (h *ShareLinkHandler) GetSharedFile(w http.ResponseWriter, r *http.Request) {
	token := r.PathValue("token")
	if token == "" {
		http.Error(w, "token is required", http.StatusBadRequest)
		return
	}

	shareLink, err := h.dbQueries.GetShareLinkByToken(r.Context(), token)
	if err == pgx.ErrNoRows {
		http.Error(w, "share link not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "failed to get share link", http.StatusInternalServerError)
		return
	}

	if shareLink.ExpiresAt.Valid && shareLink.ExpiresAt.Time.Before(time.Now()) {
		http.Error(w, "share link has expired", http.StatusGone)
		return
	}

	if shareLink.MaxDownloads.Valid && shareLink.MaxDownloads.Int32 > 0 {
		if shareLink.DownloadCount.Int32 >= shareLink.MaxDownloads.Int32 {
			http.Error(w, "download limit reached", http.StatusForbidden)
			return
		}
	}

	file, err := h.dbQueries.GetFileByID(r.Context(), db.GetFileByIDParams{
		ID: shareLink.FileID,
	})
	if err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}

	if shareLink.PasswordHash.Valid && shareLink.PasswordHash.String != "" {
		type pwdRequest struct {
			Password string `json:"password"`
		}
		var pwdReq pwdRequest
		if err := json.NewDecoder(r.Body).Decode(&pwdReq); err == nil && pwdReq.Password != "" {
			match, err := auth.CheckHash(shareLink.PasswordHash.String, pwdReq.Password)
			if err != nil || !match {
				http.Error(w, "invalid password", http.StatusUnauthorized)
				return
			}
		} else {
			http.Error(w, "password required", http.StatusUnauthorized)
			return
		}
	}

	h.dbQueries.IncrementDownloadCount(r.Context(), shareLink.ID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"file_name":   file.FileName,
		"file_size":   file.FileSize,
		"file_url":    file.FileUrl,
		"downloads":  shareLink.DownloadCount.Int32 + 1,
		"expires_at":  shareLink.ExpiresAt.Time,
	})
}

// @Summary		List share links for file
// @Description	Get all share links created by user for a specific file
// @Tags			Share Links
// @Produce		json
// @Security		ApiKeyAuth
// @Param			id	path		string	true	"File ID"
// @Success		200	{object}	[]map[string]interface{}
// @Failure		401	{object}	map[string]string
// @Router			/files/{id}/shares [get]
func (h *ShareLinkHandler) ListShareLinks(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	fileIDStr := r.PathValue("id")
	if fileIDStr == "" {
		http.Error(w, "file ID is required", http.StatusBadRequest)
		return
	}

	fileID, err := uuid.Parse(fileIDStr)
	if err != nil {
		http.Error(w, "invalid file ID", http.StatusBadRequest)
		return
	}

	_, err = h.dbQueries.GetFileByID(r.Context(), db.GetFileByIDParams{
		ID:     pgtype.UUID{Bytes: fileID, Valid: true},
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
	})
	if err == pgx.ErrNoRows {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "failed to get file", http.StatusInternalServerError)
		return
	}

	shareLinks, err := h.dbQueries.GetShareLinksByFileID(r.Context(), pgtype.UUID{Bytes: fileID, Valid: true})
	if err != nil {
		http.Error(w, "failed to get share links", http.StatusInternalServerError)
		return
	}

	type shareLinkResponse struct {
		ID             uuid.UUID  `json:"id"`
		Token          string     `json:"token"`
		ShareURL       string     `json:"share_url"`
		ExpiresAt      time.Time  `json:"expires_at"`
		HasPassword    bool       `json:"has_password"`
		MaxDownloads   int32      `json:"max_downloads"`
		DownloadCount  int32      `json:"download_count"`
		CreatedAt      time.Time  `json:"created_at"`
	}

	response := make([]shareLinkResponse, len(shareLinks))
	for i, sl := range shareLinks {
		var expiresAt time.Time
		if sl.ExpiresAt.Valid {
			expiresAt = sl.ExpiresAt.Time
		}
		response[i] = shareLinkResponse{
			ID:            sl.ID.Bytes,
			Token:         sl.Token,
			ShareURL:      fmt.Sprintf("%s/s/%s", h.config.APIBaseURL, sl.Token),
			ExpiresAt:     expiresAt,
			HasPassword:   sl.PasswordHash.Valid && sl.PasswordHash.String != "",
			MaxDownloads:  sl.MaxDownloads.Int32,
			DownloadCount: sl.DownloadCount.Int32,
			CreatedAt:     sl.CreatedAt.Time,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// @Summary		Delete share link
// @Description	Revoke a share link
// @Tags			Share Links
// @Produce		json
// @Security		ApiKeyAuth
// @Param			id	path		string	true	"Share link ID"
// @Success		200	{object}	map[string]string
// @Failure		401	{object}	map[string]string
// @Failure		404	{object}	map[string]string
// @Router			/share/{id} [delete]
func (h *ShareLinkHandler) DeleteShareLink(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	shareLinkIDStr := r.PathValue("id")
	if shareLinkIDStr == "" {
		http.Error(w, "share link ID is required", http.StatusBadRequest)
		return
	}

	shareLinkID, err := uuid.Parse(shareLinkIDStr)
	if err != nil {
		http.Error(w, "invalid share link ID", http.StatusBadRequest)
		return
	}

	err = h.dbQueries.DeleteShareLink(r.Context(), db.DeleteShareLinkParams{
		ID:        pgtype.UUID{Bytes: shareLinkID, Valid: true},
		CreatedBy: pgtype.UUID{Bytes: userID, Valid: true},
	})
	if err == pgx.ErrNoRows {
		http.Error(w, "share link not found", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "failed to delete share link", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "share link deleted",
	})
}

func generateShareToken() string {
	return fmt.Sprintf("fv-%s", uuid.New().String()[:18])
}