package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mohamed8eo/file-vault/internal/db"
)

type responseWriter struct {
	http.ResponseWriter
	status  int
	request *http.Request
}

type logEntry struct {
	method    string
	path      string
	status    int
	latencyMs int64
	requestID string
	userID    string
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func StartLogWorker(query *db.Queries) chan<- logEntry {
	ch := make(chan logEntry, 100)

	go func() {
		for entry := range ch {
			query.CreateRequestLog(context.Background(), db.CreateRequestLogParams{
				Method:    entry.method,
				Path:      entry.path,
				Status:    int32(entry.status),
				LatencyMs: entry.latencyMs,
				RequestID: entry.requestID,
				UserID:    entry.userID,
			})
		}
	}()

	return ch
}

func Logging(logCh chan<- logEntry) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, status: http.StatusOK, request: r}
			next.ServeHTTP(rw, r)

			finalRequest := rw.request
			requestID, _ := finalRequest.Context().Value(RequestIDKey).(string)
			userID, _ := finalRequest.Context().Value(UserIDKey).(uuid.UUID)
			userIDStr := ""
			if userID != (uuid.UUID{}) {
				userIDStr = userID.String()
			}

			latency := time.Since(start).Milliseconds()

			slog.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rw.status,
				"latency_ms", latency,
				"request_id", requestID,
				"user_id", userIDStr,
			)

			// non-blocking send to channel
			select {
			case logCh <- logEntry{
				method:    r.Method,
				path:      r.URL.Path,
				status:    rw.status,
				latencyMs: latency,
				requestID: requestID,
				userID:    userIDStr,
			}:
			default:
				// channel full, skip — never block the request
			}
		})
	}
}
