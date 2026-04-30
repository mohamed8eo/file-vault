package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mohamed8eo/file-vault/internal/auth"
	"github.com/stretchr/testify/assert"
)

func TestAuthMiddleware(t *testing.T) {
	secret := "test-secret-key-12345678901234567890"
	userID := "550e8400-e29b-41d4-a716-446655440000"

	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
		expectedUserID string
	}{
		{
			name:           "valid token",
			authHeader:      "Bearer " + createTestToken(secret, userID, "access-token"),
			expectedStatus: http.StatusOK,
			expectedUserID: userID,
		},
		{
			name:           "no authorization header",
			authHeader:      "",
			expectedStatus: http.StatusUnauthorized,
			expectedUserID: "",
		},
		{
			name:           "invalid token format - no Bearer",
			authHeader:      "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9",
			expectedStatus: http.StatusUnauthorized,
			expectedUserID: "",
		},
		{
			name:           "invalid token",
			authHeader:      "Bearer invalid-token",
			expectedStatus: http.StatusUnauthorized,
			expectedUserID: "",
		},
		{
			name:           "wrong secret token",
			authHeader:      "Bearer " + createTestToken("wrong-secret", userID, "access-token"),
			expectedStatus: http.StatusUnauthorized,
			expectedUserID: "",
		},
		{
			name:           "refresh token instead of access token",
			authHeader:      "Bearer " + createTestToken(secret, userID, "refresh-token"),
			expectedStatus: http.StatusOK, // Both are valid token types
			expectedUserID: userID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test handler that extracts user ID
			var extractedUserID string
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				userIDVal := r.Context().Value(UserIDKey)
				if userIDVal != nil {
					extractedUserID = userIDVal.(uuid.UUID).String()
				}
				w.WriteHeader(http.StatusOK)
			})

			// Apply auth middleware
			wrapped := Auth(secret)(handler)

			// Create request
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			// Execute
			rr := httptest.NewRecorder()
			wrapped.ServeHTTP(rr, req)

			// Assert
			assert.Equal(t, tt.expectedStatus, rr.Code)
			if tt.expectedStatus == http.StatusOK {
				assert.Equal(t, tt.expectedUserID, extractedUserID)
			}
		})
	}
}

func TestRequestIDMiddleware(t *testing.T) {
	tests := []struct {
		name string
	}{
		{"generates unique request ID"},
		{"adds request ID to header"},
		{"adds request ID to context"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requestID string
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestID = r.Context().Value(RequestIDKey).(string)
				w.WriteHeader(http.StatusOK)
			})

			wrapped := RequestID(handler)

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rr := httptest.NewRecorder()
			wrapped.ServeHTTP(rr, req)

			assert.NotEmpty(t, requestID)
			assert.Len(t, requestID, 36) // UUID length
			assert.Equal(t, requestID, rr.Header().Get("X-Request-ID"))
		})
	}
}

func TestRequestIDUniqueness(t *testing.T) {
	ids := make(map[string]bool)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Context().Value(RequestIDKey).(string)
		ids[id] = true
		w.WriteHeader(http.StatusOK)
	})

	wrapped := RequestID(handler)

	// Generate many request IDs
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rr := httptest.NewRecorder()
		wrapped.ServeHTTP(rr, req)
	}

	// All should be unique
	assert.Len(t, ids, 100)
}

func TestCreateStack(t *testing.T) {
	callOrder := []int{}

	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callOrder = append(callOrder, 1)
			next.ServeHTTP(w, r)
		})
	}

	middleware2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callOrder = append(callOrder, 2)
			next.ServeHTTP(w, r)
		})
	}

	middleware3 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			callOrder = append(callOrder, 3)
			next.ServeHTTP(w, r)
		})
	}

	// Create stack - middleware applied in order
	stack := CreateStack(middleware1, middleware2, middleware3)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callOrder = append(callOrder, 4)
	})

	wrapped := stack(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)

	// Middleware should be called in order: 1 -> 2 -> 3 -> handler(4)
	assert.Equal(t, []int{1, 2, 3, 4}, callOrder)
}

func TestAuthMiddlewareWithRefreshTokenSecret(t *testing.T) {
	// Test that refresh token secret also works for auth middleware
	secret := "refresh-token-secret-key-1234567890"
	userID := "550e8400-e29b-41d4-a716-446655440000"

	token := createTestToken(secret, userID, "refresh-token")

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := Auth(secret)(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rr := httptest.NewRecorder()
	wrapped.ServeHTTP(rr, req)

	// Should succeed because token is valid
	assert.Equal(t, http.StatusOK, rr.Code)
}

// Helper function to create test tokens
func createTestToken(secret, userID, issuer string) string {
	token, _ := auth.MakeToken(issuer, secret, userID, 15*time.Minute)
	return token
}