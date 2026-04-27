package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type responseWriter struct {
	http.ResponseWriter
	status  int
	request *http.Request
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func Logging(next http.Handler) http.Handler {
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

		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.status,
			"latency_ms", time.Since(start).Milliseconds(),
			"request_id", requestID,
			"user_id", userIDStr,
		)
	})
}
