package middleware

import (
	"context"
	"net/http"

	"github.com/mohamed8eo/file-vault/internal/auth"
)

type contextKey string

const UserIDKey contextKey = "userID"

func Auth(accessTokenSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authorizationHeader := r.Header.Get("Authorization")
			accessToken, err := auth.GetBearerToken(authorizationHeader)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			userID, err := auth.ValidateJWT(accessTokenSecret, accessToken)
			if err != nil {
				http.Error(w, err.Error(), http.StatusUnauthorized)
				return
			}
			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			newR := r.WithContext(ctx)

			if rw, ok := w.(*responseWriter); ok {
				rw.request = newR
			}

			next.ServeHTTP(w, newR)
		})
	}
}
