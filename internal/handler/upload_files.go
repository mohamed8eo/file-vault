package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mohamed8eo/file-vault/internal/db"
	"github.com/mohamed8eo/file-vault/internal/middleware"
)

type FileType string

const (
	FileTypeImage FileType = "image"
	FileTypeVideo FileType = "video"
	FileTypeOther FileType = "other"
)

type UploadHanlder struct {
	dbQueries    *db.Queries
	s3Bucket     string
	s3Region     string
	s3Client     *s3.Client
	s3CloudFront string
}

func NewUploadHandler(
	dbQueries *db.Queries,
	s3Bucket string,
	s3Region string,
	s3CloudFront string,
	s3Client *s3.Client,
) *UploadHanlder {
	return &UploadHanlder{
		dbQueries:    dbQueries,
		s3Bucket:     s3Bucket,
		s3Region:     s3Region,
		s3Client:     s3Client,
		s3CloudFront: s3CloudFront,
	}
}

type uploadConfig struct {
	maxSize     int64
	allowedExt  []string
	allowedMime []string
	prefix      string
}

var (
	imageConfig = uploadConfig{
		maxSize:     10 << 20,
		allowedExt:  []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".svg"},
		allowedMime: []string{"image/jpeg", "image/png", "image/gif", "image/webp", "image/svg+xml"},
		prefix:      "images",
	}
	videoConfig = uploadConfig{
		maxSize:     500 << 20,
		allowedExt:  []string{".mp4", ".webm", ".mov", ".avi", ".mkv"},
		allowedMime: []string{"video/mp4", "video/webm", "video/quicktime", "video/x-msvideo", "video/x-matroska"},
		prefix:      "videos",
	}
	fileConfig = uploadConfig{
		maxSize:     50 << 20,
		allowedExt:  []string{},
		allowedMime: []string{},
		prefix:      "files",
	}
)

// @Summary		Upload image
// @Description	Upload an image file (jpg, png, gif, webp, svg - max 10MB)
// @Tags			Files
// @Accept		multipart/form-data
// @Produce		json
// @Security		ApiKeyAuth
// @Param		file	formData	file	true	"Image file"
// @Success		200	{object}	map[string]interface{}
// @Failure		400	{object}	map[string]string
// @Failure		413	{object}	map[string]string
// @Router		/upload/image [post]
func (h *UploadHanlder) UploadImage(w http.ResponseWriter, r *http.Request) {
	if err := h.uploadSingle(w, r, imageConfig, FileTypeImage); err != nil {
		http.Error(w, err.Error(), err.(*httpError).code)
		return
	}
}

// @Summary		Upload video
// @Description	Upload a video file (mp4, webm, mov, avi, mkv - max 500MB)
// @Tags			Files
// @Accept		multipart/form-data
// @Produce		json
// @Security		ApiKeyAuth
// @Param		file	formData	file	true	"Video file"
// @Success		200	{object}	map[string]interface{}
// @Failure		400	{object}	map[string]string
// @Failure		413	{object}	map[string]string
// @Router		/upload/video [post]
func (h *UploadHanlder) UploadVideo(w http.ResponseWriter, r *http.Request) {
	if err := h.uploadSingle(w, r, videoConfig, FileTypeVideo); err != nil {
		http.Error(w, err.Error(), err.(*httpError).code)
		return
	}
}

// @Summary		Upload file
// @Description	Upload any file type (max 50MB)
// @Tags			Files
// @Accept		multipart/form-data
// @Produce		json
// @Security		ApiKeyAuth
// @Param		file	formData	file	true	"File to upload"
// @Success		200	{object}	map[string]interface{}
// @Failure		400	{object}	map[string]string
// @Failure		413	{object}	map[string]string
// @Router		/upload [post]
func (h *UploadHanlder) UploadFile(w http.ResponseWriter, r *http.Request) {
	if err := h.uploadSingle(w, r, fileConfig, FileTypeOther); err != nil {
		http.Error(w, err.Error(), err.(*httpError).code)
		return
	}
}

type httpError struct {
	code int
	msg  string
}

func (e *httpError) Error() string {
	return e.msg
}

func (h *UploadHanlder) uploadSingle(w http.ResponseWriter, r *http.Request, cfg uploadConfig, fileType FileType) error {
	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		return &httpError{code: http.StatusUnauthorized, msg: "unauthorized"}
	}

	r.Body = http.MaxBytesReader(w, r.Body, cfg.maxSize)

	if err := r.ParseMultipartForm(cfg.maxSize); err != nil {
		return &httpError{code: http.StatusRequestEntityTooLarge, msg: "file too large"}
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		return &httpError{code: http.StatusBadRequest, msg: "invalid file"}
	}
	defer file.Close()

	contentType, err := detectContentType(file, header.Filename)
	if err != nil {
		return &httpError{code: http.StatusInternalServerError, msg: "failed to detect content type"}
	}

	if !isAllowedType(contentType, header.Filename, cfg) {
		return &httpError{code: http.StatusNotAcceptable, msg: "file type not allowed"}
	}

	key := fmt.Sprintf("%s/%d-%s", cfg.prefix, time.Now().Unix(), sanitizeFilename(header.Filename))
	fileSize := header.Size

	input := &s3.PutObjectInput{
		Bucket:      &h.s3Bucket,
		Key:         &key,
		Body:        file,
		ContentType: &contentType,
	}

	_, err = h.s3Client.PutObject(context.TODO(), input)
	if err != nil {
		return &httpError{code: http.StatusInternalServerError, msg: "upload failed"}
	}

	fileURL := fmt.Sprintf("https://%s/%s", h.s3CloudFront, key)

	createdFile, err := h.dbQueries.CreateFile(r.Context(), db.CreateFileParams{
		UserID:   pgtype.UUID{Bytes: userID, Valid: true},
		FileName: header.Filename,
		FileUrl:  fileURL,
		FileSize: fileSize,
	})
	if err != nil {
		return &httpError{code: http.StatusInternalServerError, msg: "failed to save record"}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"id":      createdFile.ID.String(),
		"message": "uploaded successfully",
		"url":     fileURL,
	})
	return nil
}

func detectContentType(file io.ReadSeeker, filename string) (string, error) {
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err.Error() != "EOF" {
		return "", err
	}
	file.Seek(0, 0)

	mediaType := http.DetectContentType(buffer[:n])
	if mediaType == "application/octet-stream" {
		ext := strings.ToLower(filepath.Ext(filename))
		switch ext {
		case ".jpg", ".jpeg":
			mediaType = "image/jpeg"
		case ".png":
			mediaType = "image/png"
		case ".gif":
			mediaType = "image/gif"
		case ".webp":
			mediaType = "image/webp"
		case ".svg":
			mediaType = "image/svg+xml"
		case ".mp4":
			mediaType = "video/mp4"
		case ".webm":
			mediaType = "video/webm"
		case ".mov":
			mediaType = "video/quicktime"
		case ".avi":
			mediaType = "video/x-msvideo"
		case ".mkv":
			mediaType = "video/x-matroska"
		}
	}
	return mediaType, nil
}

func isAllowedType(contentType, filename string, cfg uploadConfig) bool {
	if len(cfg.allowedMime) == 0 {
		return true
	}
	ext := strings.ToLower(filepath.Ext(filename))
	for _, allowed := range cfg.allowedExt {
		if ext == allowed {
			return true
		}
	}
	for _, allowed := range cfg.allowedMime {
		if contentType == allowed {
			return true
		}
	}
	return false
}

func sanitizeFilename(name string) string {
	ext := filepath.Ext(name)
	name = strings.TrimSuffix(name, ext)
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	name = strings.ToLower(name)
	return name + ext
}

// @Summary		List files
// @Description	Get list of user's files with pagination, sorting and filtering
// @Tags			Files
// @Produce		json
// @Security		ApiKeyAuth
// @Param		limit	query		int		false	"Number of files (max 100)"
// @Param		offset	query		int		false	"Number of files to skip"
// @Param		page	query		int		false	"Page number"
// @Param		sort	query		string	false	"Sort by: date, name, size"
// @Param		type	query		string	false	"Filter by type: image, video, document"
// @Success		200	{array}		map[string]interface{}
// @Router		/files [get]
func (h *UploadHanlder) GetFiles(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(uuid.UUID)

	limit := 20
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		v, err := strconv.Atoi(l)
		if err != nil || v <= 0 || v > 100 {
			http.Error(w, "invalid limit", http.StatusBadRequest)
			return
		}
		limit = v
	}

	if o := r.URL.Query().Get("offset"); o != "" {
		v, err := strconv.Atoi(o)
		if err != nil || v < 0 {
			http.Error(w, "invalid offset", http.StatusBadRequest)
			return
		}
		offset = v
	}

	sort := r.URL.Query().Get("sort")
	if sort != "" && sort != "date" && sort != "name" && sort != "size" {
		http.Error(w, "invalid sort (date, name, size)", http.StatusBadRequest)
		return
	}

	fileType := r.URL.Query().Get("type")
	if fileType != "" && fileType != "image" && fileType != "video" && fileType != "document" {
		http.Error(w, "invalid type (image, video, document)", http.StatusBadRequest)
		return
	}

	query := r.URL.Query().Get("q")

	files, err := h.dbQueries.GetFilesFiltered(r.Context(), db.GetFilesFilteredParams{
		UserID:  pgtype.UUID{Bytes: userID, Valid: true},
		Column2: sort,
		Limit:   int32(limit),
		Column4: query,
		Column5: fileType,
		Offset:  int32(offset),
	})
	if err != nil {
		http.Error(w, "failed to fetch files", http.StatusInternalServerError)
		return
	}

	type fileResponse struct {
		ID        string `json:"id"`
		FileName  string `json:"file_name"`
		FileURL   string `json:"file_url"`
		FileSize  int64  `json:"file_size"`
		CreatedAt string `json:"created_at"`
	}

	result := []fileResponse{}

	for _, f := range files {
		result = append(result, fileResponse{
			ID:        f.ID.String(),
			FileName:  f.FileName,
			FileURL:   f.FileUrl,
			FileSize:  f.FileSize,
			CreatedAt: f.CreatedAt.Time.Format(time.RFC3339),
		})
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

// @Summary		Search files
// @Description	Search files by name
// @Tags			Files
// @Produce		json
// @Security		ApiKeyAuth
// @Param		q		query		string	true	"Search query"
// @Param		limit	query		int		false	"Number of results"
// @Success		200	{array}		map[string]interface{}
// @Router		/files/search [get]
func (h *UploadHanlder) SearchFiles(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(uuid.UUID)

	query := r.URL.Query().Get("q")
	if len(query) < 2 {
		http.Error(w, "search query must be at least 2 characters", http.StatusBadRequest)
		return
	}

	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		v, _ := strconv.Atoi(l)
		if v > 0 && v <= 100 {
			limit = v
		}
	}

	files, err := h.dbQueries.SearchFiles(r.Context(), db.SearchFilesParams{
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
		Column2: pgtype.Text{
			String: query,
			Valid:  true,
		},
		Limit: int32(limit),
	})
	if err != nil {
		http.Error(w, "failed to search files", http.StatusInternalServerError)
		return
	}

	type fileResponse struct {
		ID        string `json:"id"`
		FileName  string `json:"file_name"`
		FileURL   string `json:"file_url"`
		FileSize  int64  `json:"file_size"`
		CreatedAt string `json:"created_at"`
	}

	result := []fileResponse{}
	for _, f := range files {
		result = append(result, fileResponse{
			ID:        f.ID.String(),
			FileName:  f.FileName,
			FileURL:   f.FileUrl,
			FileSize:  f.FileSize,
			CreatedAt: f.CreatedAt.Time.Format(time.RFC3339),
		})
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}

// @Summary		Get file by ID
// @Description	Get file details including CloudFront URL
// @Tags			Files
// @Produce		json
// @Security		ApiKeyAuth
// @Param		id	path		string	true	"File ID"
// @Success		200	{object}	map[string]interface{}
// @Failure		404	{object}	map[string]string
// @Router		/files/{id} [get]
func (h *UploadHanlder) GetFileByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	fileIDStr := r.PathValue("id")
	if fileIDStr == "" {
		http.Error(w, "fileID require", http.StatusBadRequest)
		return
	}

	fileIDUUID, err := uuid.Parse(fileIDStr)
	if err != nil {
		http.Error(w, "failed to parse the file ID", http.StatusInternalServerError)
		return
	}

	file, err := h.dbQueries.GetFileByID(r.Context(), db.GetFileByIDParams{
		ID:     pgtype.UUID{Bytes: fileIDUUID, Valid: true},
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
	})
	if err != nil {
		if err == pgx.ErrNoRows {
			http.Error(w, "file not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to fetch file", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"message":   "Get file URL Successfully",
		"file_url":  file.FileUrl,
		"file_name": file.FileName,
		"file_size": file.FileSize,
	})
}

// @Summary		Delete file
// @Description	Delete a single file from S3 and database
// @Tags			Files
// @Produce		json
// @Security		ApiKeyAuth
// @Param		id	path		string	true	"File ID"
// @Success		204	{object}	map[string]string
// @Failure		404	{object}	map[string]string
// @Router		/files/{id} [delete]
func (h *UploadHanlder) DeleteFile(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	fileIDStr := r.PathValue("id")
	if fileIDStr == "" {
		http.Error(w, "fileID require", http.StatusBadRequest)
		return
	}

	fileIDUUID, err := uuid.Parse(fileIDStr)
	if err != nil {
		http.Error(w, "failed to parse the file ID", http.StatusInternalServerError)
		return
	}

	file, err := h.dbQueries.GetFileByID(r.Context(), db.GetFileByIDParams{
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
		ID:     pgtype.UUID{Bytes: fileIDUUID, Valid: true},
	})
	if err != nil {
		http.NotFound(w, r)
		return
	}

	_, err = h.s3Client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: &h.s3Bucket,
		Key:    &file.FileUrl,
	})
	if err != nil {
		http.Error(w, "failed to delete from storage", http.StatusInternalServerError)
		return
	}

	if err := h.dbQueries.DeleteFileByID(r.Context(), db.DeleteFileByIDParams{
		UserID: pgtype.UUID{Bytes: userID, Valid: true},
		ID:     pgtype.UUID{Bytes: fileIDUUID, Valid: true},
	}); err != nil {
		http.Error(w, "failed to delete the file", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// @Summary		Get storage stats
// @Description	Get storage statistics (total files, size breakdown by type)
// @Tags			Files
// @Produce		json
// @Security		ApiKeyAuth
// @Success		200	{object}	map[string]interface{}
// @Router		/files/stats [get]
func (h *UploadHanlder) GetStorageStats(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(uuid.UUID)

	stats, err := h.dbQueries.GetStorageStats(r.Context(), pgtype.UUID{Bytes: userID, Valid: true})
	if err != nil {
		http.Error(w, "failed to get stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"total_files": stats.TotalFiles,
		"total_size":  stats.TotalSize,
		"images": map[string]int64{
			"count": stats.ImageCount,
			"size":  stats.ImageSize,
		},
		"videos": map[string]int64{
			"count": stats.VideoCount,
			"size":  stats.VideoSize,
		},
		"documents": map[string]int64{
			"count": stats.DocumentCount,
			"size":  stats.DocumentSize,
		},
	})
}

// @Summary		Bulk delete files
// @Description	Delete multiple files at once
// @Tags			Files
// @Accept		json
// @Produce		json
// @Security		ApiKeyAuth
// @Param		ids	body		object	true	"Array of file IDs"
// @Success		200	{object}	map[string]interface{}
// @Router		/files/delete [post]
func (h *UploadHanlder) DeleteFiles(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if len(req.IDs) == 0 {
		http.Error(w, "no file IDs provided", http.StatusBadRequest)
		return
	}

	if len(req.IDs) > 50 {
		http.Error(w, "max 50 files at once", http.StatusBadRequest)
		return
	}

	// Convert string IDs to UUIDs
	var ids []pgtype.UUID
	for _, idStr := range req.IDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			http.Error(w, "invalid file ID: "+idStr, http.StatusBadRequest)
			return
		}
		ids = append(ids, pgtype.UUID{Bytes: id, Valid: true})
	}

	deleted, err := h.dbQueries.DeleteFilesByIDs(r.Context(), db.DeleteFilesByIDsParams{
		UserID:  pgtype.UUID{Bytes: userID, Valid: true},
		Column2: ids,
	})
	if err != nil {
		http.Error(w, "failed to delete files", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"deleted": len(deleted),
		"message": fmt.Sprintf("deleted %d file(s)", len(deleted)),
	})
}
