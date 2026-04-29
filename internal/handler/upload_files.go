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

func (h *UploadHanlder) UploadImage(w http.ResponseWriter, r *http.Request) {
	if err := h.uploadSingle(w, r, imageConfig, FileTypeImage); err != nil {
		http.Error(w, err.Error(), err.(*httpError).code)
		return
	}
}

func (h *UploadHanlder) UploadVideo(w http.ResponseWriter, r *http.Request) {
	if err := h.uploadSingle(w, r, videoConfig, FileTypeVideo); err != nil {
		http.Error(w, err.Error(), err.(*httpError).code)
		return
	}
}

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
