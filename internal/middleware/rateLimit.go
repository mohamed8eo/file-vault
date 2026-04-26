package middleware

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

func RateLimit(rdb *redis.Client, limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.Background()
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}
			key := "ratelimit:ip:" + ip

			if userID, ok := r.Context().Value(UserIDKey).(string); ok && userID != "" {
				key = "ratelimit:user:" + userID
			}

			now := time.Now().UnixMilli()
			windowStart := now - window.Milliseconds()

			pipe := rdb.Pipeline()
			pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", windowStart))
			pipe.ZCard(ctx, key)
			pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: now})
			pipe.Expire(ctx, key, window)
			results, err := pipe.Exec(ctx)
			if err != nil {
				http.Error(w, "rate limit error", http.StatusInternalServerError)
				return
			}

			count := results[1].(*redis.IntCmd).Val()
			if count >= int64(limit) {
				w.Header().Set("Retry-After", "60")
				http.Error(w, "too many requests", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
