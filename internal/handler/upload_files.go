package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mohamed8eo/file-vault/internal/db"
	"github.com/mohamed8eo/file-vault/internal/middleware"
)

type UploadHanlder struct {
	dbQueries *db.Queries
	s3Bucket  string
	s3Region  string
	s3Client  *s3.Client
}

func NewUploadHandler(
	dbQueries *db.Queries,
	s3Bucket string,
	s3Region string,
	s3Client *s3.Client,
) *UploadHanlder {
	return &UploadHanlder{
		dbQueries: dbQueries,
		s3Bucket:  s3Bucket,
		s3Region:  s3Region,
		s3Client:  s3Client,
	}
}

func (h *UploadHanlder) UploadFile(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	maxReader := 10 << 20
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxReader))

	if err := r.ParseMultipartForm(int64(maxReader)); err != nil {
		http.Error(w, err.Error(), http.StatusRequestEntityTooLarge)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	mediaType, _, err := mime.ParseMediaType(header.Header.Get("Content-Type"))
	if err != nil {
		http.Error(w, "Invalide Content Type", http.StatusBadRequest)
		return
	}

	key := fmt.Sprintf("uploads/%d-%s", time.Now().Unix(), header.Filename)

	input := &s3.PutObjectInput{
		Bucket:      &h.s3Bucket,
		Key:         &key,
		Body:        file,
		ContentType: &mediaType,
	}

	_, err = h.s3Client.PutObject(context.TODO(), input)
	if err != nil {
		http.Error(w, "S3 upload failed", http.StatusInternalServerError)
		return
	}

	// Store in db
	_, err = h.dbQueries.CreateFile(r.Context(), db.CreateFileParams{
		UserID: pgtype.UUID{
			Bytes: userID,
			Valid: true,
		},
		FileName: header.Filename,
		FileUrl:  key,
	})
	if err != nil {
		http.Error(w, "failed to save file record", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("File upload successfully!"))
}

func (h *UploadHanlder) GetFiles(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(uuid.UUID)

	files, err := h.dbQueries.GetFilesByUser(r.Context(), pgtype.UUID{
		Bytes: userID,
		Valid: true,
	})
	if err != nil {
		http.Error(w, "failed to fetch files", http.StatusInternalServerError)
		return
	}

	presignClient := s3.NewPresignClient(h.s3Client)

	type fileResponse struct {
		ID        string `json:"id"`
		FileName  string `json:"file_name"`
		FileURL   string `json:"file_url"`
		CreatedAt string `json:"created_at"`
	}

	result := []fileResponse{}

	for _, f := range files {
		presigned, err := presignClient.PresignGetObject(context.TODO(), &s3.GetObjectInput{
			Bucket: &h.s3Bucket,
			Key:    &f.FileUrl,
		}, s3.WithPresignExpires(10*time.Minute))
		if err != nil {
			http.Error(w, "failed to generate url", http.StatusInternalServerError)
			return
		}

		result = append(result, fileResponse{
			ID:        f.ID.String(),
			FileName:  f.FileName,
			FileURL:   presigned.URL,
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
		ID: pgtype.UUID{
			Bytes: fileIDUUID,
			Valid: true,
		},
		UserID: pgtype.UUID{
			Bytes: userID,
			Valid: true,
		},
	})
	if err != nil {
		http.Error(w, "failed to generate url", http.StatusInternalServerError)
		return
	}

	presignClient := s3.NewPresignClient(h.s3Client)

	presign, err := presignClient.PresignGetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: &h.s3Bucket,
		Key:    &file.FileUrl,
	}, s3.WithPresignExpires(10*time.Minute))
	if err != nil {
		http.Error(w, "failed to generate url", http.StatusInternalServerError)
		return
	}

	type response struct {
		Message  string `json:"message"`
		FileURL  string `json:"file_url"`
		FileName string `json:"file_name"`
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(&response{
		Message:  "Get file URL Successfully",
		FileURL:  presign.URL,
		FileName: file.FileName,
	}); err != nil {
		http.Error(w, "failed to encode the res json", http.StatusInternalServerError)
		return
	}
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

	// get File data from db
	file, err := h.dbQueries.GetFileByID(r.Context(), db.GetFileByIDParams{
		UserID: pgtype.UUID{
			Bytes: userID,
			Valid: true,
		},
		ID: pgtype.UUID{
			Bytes: fileIDUUID,
			Valid: true,
		},
	})
	if err != nil {
		http.NotFound(w, r)
		return
	}

	// Delete file from S3
	_, err = h.s3Client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: &h.s3Bucket,
		Key:    &file.FileUrl,
	})
	if err != nil {
		http.Error(w, "failed to delete from storage", http.StatusInternalServerError)
		return
	}

	// Delete file from db
	if err := h.dbQueries.DeleteFileByID(r.Context(), db.DeleteFileByIDParams{
		UserID: pgtype.UUID{
			Bytes: userID,
			Valid: true,
		},
		ID: pgtype.UUID{
			Bytes: fileIDUUID,
			Valid: true,
		},
	}); err != nil {
		http.Error(w, "failed to delete the file", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
